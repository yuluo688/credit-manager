package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/lockfile"
	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	PluginID      = "credit-manager"
	PluginName    = "CPA Credit Manager"
	PluginVersion = "1.8.2"
	// CallerScopeMetadataKey mirrors sdk/cliproxy/executor.CallerScopeMetadataKey.
	CallerScopeMetadataKey = "caller_scope"
)

const staleCleanupInterval = time.Minute

// Service is the process-wide plugin runtime.
type Service struct {
	cfg                config.Config
	peppers            config.PepperSet
	store              *store.Store
	authMu             authMutex
	authCond           *sync.Cond
	authPending        map[string]*pendingAuthCapture
	warmupHolds        map[string]store.AuthIdentity
	cleanupMu          sync.Mutex
	lastCleanup        time.Time
	authQuotaMu        sync.RWMutex
	authQuotaSource    AuthQuotaSource
	authQuotaRefreshMu sync.Mutex
	warmupMu           sync.RWMutex
	warmupExecutor     AuthWarmupExecutor
	warmupCancel       context.CancelFunc
	warmupSems         map[string]chan struct{}
	warmupSemLimit     int
	authPickCursor     map[string]int
	directorySyncer    ModelDirectorySyncer
	directoryIDsMu     sync.Mutex
	lastDirectoryIDs   []string
}

// authMutex is copy-safe because it shares its underlying lock. A same-store
// reconfigure publishes a new Service while in-flight requests still use the
// old one, so both instances must serialize access to the same auth map.
type authMutex struct {
	shared *sync.Mutex
}

func (m *authMutex) init() {
	if m.shared == nil {
		m.shared = &sync.Mutex{}
	}
}

func (m *authMutex) Lock() {
	m.init()
	m.shared.Lock()
}

func (m *authMutex) Unlock() {
	m.init()
	m.shared.Unlock()
}

var current atomic.Pointer[Service]

func Current() *Service { return current.Load() }

func Replace(svc *Service) {
	if old := current.Swap(svc); old != nil {
		_ = old.Close()
	}
}

func Shutdown() {
	if old := current.Swap(nil); old != nil {
		_ = old.Close()
	}
}

func Open(ctx context.Context, cfg config.Config) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	peppers, err := cfg.LoadPeppers()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath, err := store.CanonicalDatabasePath(cfg.DatabasePath())
	if err != nil {
		return nil, err
	}
	st, err := store.OpenLocked(ctx, dbPath, store.OpenOptions{BusyTimeout: cfg.BusyTimeout}, lockfile.New())
	if err != nil {
		return nil, err
	}
	svc := &Service{cfg: cfg, peppers: peppers, store: st, authPending: make(map[string]*pendingAuthCapture), warmupHolds: make(map[string]store.AuthIdentity)}
	svc.authMu.init()
	svc.authCond = sync.NewCond(&svc.authMu)
	if err := svc.ensureBootstrap(ctx); err != nil {
		_ = svc.Close()
		return nil, err
	}
	if _, err := svc.cleanupStaleReservations(ctx, true); err != nil {
		_ = svc.Close()
		return nil, fmt.Errorf("release stale reservations: %w", err)
	}
	svc.RefreshModelDirectory(ctx)
	return svc, nil
}

func (s *Service) Close() error {
	if s == nil || s.store == nil {
		return nil
	}
	s.StopAuthWarmup()
	return s.store.Close()
}

func (s *Service) Config() config.Config { return s.cfg }
func (s *Service) Store() *store.Store   { return s.store }
func (s *Service) Peppers() config.PepperSet {
	return s.peppers
}

// SetAuthQuotaSource attaches the host bridge used to inspect auth files and
// make authenticated quota requests. It may be called after Open or Configure.
func (s *Service) SetAuthQuotaSource(source AuthQuotaSource) {
	if s == nil {
		return
	}
	s.authQuotaMu.Lock()
	s.authQuotaSource = source
	s.authQuotaMu.Unlock()
}

// EnsureDataDir is exported for configuration validation helpers.
func EnsureDataDir(path string) error {
	return os.MkdirAll(filepath.Clean(path), 0o700)
}

// Guard serializes reconfigure so exclusive DB lock handoff cannot race itself.
var reconfigureMu sync.Mutex

// Configure applies host register/reconfigure YAML.
// Same database path reuses the open store (no second exclusive lock).
// Changing the database path requires a CPA restart so in-flight requests never
// retain a pointer to a store that has been closed underneath them.
func Configure(ctx context.Context, rawYAML []byte) error {
	reconfigureMu.Lock()
	defer reconfigureMu.Unlock()
	cfg, err := config.ParseYAML(rawYAML)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	peppers, err := cfg.LoadPeppers()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	dbPath, err := store.CanonicalDatabasePath(cfg.DatabasePath())
	if err != nil {
		return err
	}

	if old := current.Load(); old != nil {
		oldDBPath, err := store.CanonicalDatabasePath(old.cfg.DatabasePath())
		if err != nil {
			return err
		}
		if oldDBPath != dbPath {
			return fmt.Errorf("changing database path while CPA is running is not supported; restart CPA first")
		}
		// Keep the locked SQLite writer. Opening a second handle deadlocks on *.db.lock.
		next := &Service{cfg: cfg, peppers: peppers, store: old.store, authCond: old.authCond, authPending: old.authPending, warmupHolds: old.warmupHolds}
		next.authMu.shared = old.authMu.shared
		next.SetAuthQuotaSource(old.authQuotaSourceValue())
		next.SetAuthWarmupExecutor(old.authWarmupExecutorValue())
		next.SetModelDirectorySyncer(old.directorySyncer)
		old.directoryIDsMu.Lock()
		next.lastDirectoryIDs = append([]string(nil), old.lastDirectoryIDs...)
		old.directoryIDsMu.Unlock()
		if err := next.ensureBootstrap(ctx); err != nil {
			return err
		}
		if _, err := next.cleanupStaleReservations(ctx, true); err != nil {
			return fmt.Errorf("release stale reservations: %w", err)
		}
		if !current.CompareAndSwap(old, next) {
			return fmt.Errorf("service replaced concurrently during reconfigure")
		}
		old.StopAuthWarmup()
		next.StartAuthWarmup()
		next.RefreshModelDirectory(ctx)
		// Leave old.store attached: in-flight callers may still hold *old.
		// Ownership of Close stays with the published Service / Shutdown.
		return nil
	}

	// First start: acquire the exclusive writer lock for the canonical path.
	st, err := store.OpenLocked(ctx, dbPath, store.OpenOptions{BusyTimeout: cfg.BusyTimeout}, lockfile.New())
	if err != nil {
		return err
	}
	svc := &Service{cfg: cfg, peppers: peppers, store: st, authPending: make(map[string]*pendingAuthCapture), warmupHolds: make(map[string]store.AuthIdentity)}
	svc.authMu.init()
	svc.authCond = sync.NewCond(&svc.authMu)
	if err := svc.ensureBootstrap(ctx); err != nil {
		_ = svc.Close()
		return err
	}
	if _, err := svc.cleanupStaleReservations(ctx, true); err != nil {
		_ = svc.Close()
		return fmt.Errorf("release stale reservations: %w", err)
	}
	current.Store(svc)
	svc.RefreshModelDirectory(ctx)
	return nil
}
