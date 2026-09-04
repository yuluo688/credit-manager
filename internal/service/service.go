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
	PluginVersion = "1.7.1"
	// CallerScopeMetadataKey mirrors sdk/cliproxy/executor.CallerScopeMetadataKey.
	CallerScopeMetadataKey = "caller_scope"
)

const staleCleanupInterval = time.Minute

// Service is the process-wide plugin runtime.
type Service struct {
	cfg                config.Config
	peppers            config.PepperSet
	store              *store.Store
	authMu             sync.Mutex
	authCond           *sync.Cond
	authPending        map[string]*pendingAuthCapture
	cleanupMu          sync.Mutex
	lastCleanup        time.Time
	authQuotaMu        sync.RWMutex
	authQuotaSource    AuthQuotaSource
	authQuotaRefreshMu sync.Mutex
	authPickCursor     map[string]int
	directorySyncer    ModelDirectorySyncer
	directoryIDsMu     sync.Mutex
	lastDirectoryIDs   []string
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
	dbPath := cfg.DatabasePath()
	st, err := store.OpenLocked(ctx, dbPath, store.OpenOptions{BusyTimeout: cfg.BusyTimeout}, lockfile.New())
	if err != nil {
		return nil, err
	}
	svc := &Service{cfg: cfg, peppers: peppers, store: st}
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
// Path changes close the old instance first, then open the new one.
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
	dbPath := filepath.Clean(cfg.DatabasePath())

	if old := current.Load(); old != nil && filepath.Clean(old.cfg.DatabasePath()) == dbPath {
		// Keep the locked SQLite writer. Opening a second handle deadlocks on *.db.lock.
		next := &Service{cfg: cfg, peppers: peppers, store: old.store}
		next.SetAuthQuotaSource(old.authQuotaSourceValue())
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
		next.RefreshModelDirectory(ctx)
		// Leave old.store attached: in-flight callers may still hold *old.
		// Ownership of Close stays with the published Service / Shutdown.
		return nil
	}

	// Different data path (or first start): release the previous exclusive lock first.
	if old := current.Swap(nil); old != nil {
		_ = old.Close()
	}
	st, err := store.OpenLocked(ctx, dbPath, store.OpenOptions{BusyTimeout: cfg.BusyTimeout}, lockfile.New())
	if err != nil {
		return err
	}
	svc := &Service{cfg: cfg, peppers: peppers, store: st}
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
