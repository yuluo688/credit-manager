package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrInvalidArgument           = errors.New("invalid argument")
	ErrCallerNotFound            = errors.New("caller not found")
	ErrCallerDisabled            = errors.New("caller is disabled")
	ErrInsufficientQuota         = errors.New("insufficient quota")
	ErrReservationNotFound       = errors.New("reservation not found")
	ErrReservationFinalized      = errors.New("reservation is already finalized")
	ErrIdempotencyConflict       = errors.New("idempotency key is already used for a different request")
	ErrPricingRuleNotFound       = errors.New("pricing rule not found")
	ErrModelDisabled             = errors.New("model is disabled")
	ErrPluginKeyNotFound         = errors.New("plugin key not found")
	ErrPluginKeyDisabled         = errors.New("plugin key is disabled")
	ErrPluginKeyRevoked          = errors.New("plugin key is revoked")
	ErrPluginKeyExpired          = errors.New("plugin key is expired")
	ErrModelNotAllowed           = errors.New("model is not allowed for this key")
	ErrDailyQuotaExceeded        = errors.New("daily quota exceeded")
	ErrWeeklyQuotaExceeded       = errors.New("weekly quota exceeded")
	ErrMonthlyQuotaExceeded      = errors.New("monthly quota exceeded")
	ErrDailyTokenLimitExceeded   = errors.New("daily model token limit exceeded")
	ErrWeeklyTokenLimitExceeded  = errors.New("weekly model token limit exceeded")
	ErrMonthlyTokenLimitExceeded = errors.New("monthly model token limit exceeded")
	ErrTotalTokenLimitExceeded   = errors.New("total model token limit exceeded")
	ErrConcurrentLimit           = errors.New("maximum concurrent requests reached")
)

// Store owns the SQLite connection pool. MaxOpenConns is intentionally one.
type Store struct {
	db        *sql.DB
	lease     *storeLease
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

type OpenOptions struct {
	BusyTimeout time.Duration
}

// FileLock describes the required host-owned cross-process writer lock.
type FileLock interface {
	Lock(ctx context.Context, path string) (unlock func() error, err error)
}

func LockPath(databasePath string) string {
	return filepath.Clean(databasePath) + ".lock"
}

func Open(ctx context.Context, databasePath string, options OpenOptions) (*Store, error) {
	if strings.TrimSpace(databasePath) == "" {
		return nil, fmt.Errorf("%w: database path is required", ErrInvalidArgument)
	}
	if options.BusyTimeout <= 0 || options.BusyTimeout > 5*time.Minute {
		return nil, fmt.Errorf("%w: busy timeout must be greater than zero and at most 5 minutes", ErrInvalidArgument)
	}

	dsn := sqliteDSN(databasePath, options.BusyTimeout)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{db: db}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := Migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func OpenLocked(ctx context.Context, databasePath string, options OpenOptions, locker FileLock) (*Store, error) {
	if locker == nil {
		return nil, fmt.Errorf("%w: file locker is required", ErrInvalidArgument)
	}
	lease, err := newStoreLease(databasePath)
	if err != nil {
		return nil, err
	}
	unlock, err := acquireWriterLock(ctx, locker, databasePath, lease)
	if err != nil {
		return nil, err
	}
	store, err := Open(ctx, databasePath, options)
	if err != nil {
		_ = unlock()
		lease.release()
		return nil, err
	}
	if !lease.current() {
		_ = store.Close()
		_ = unlock()
		lease.release()
		return nil, errors.New("database handover was superseded by another plugin instance")
	}
	store.attachLease(lease, unlock)
	return store, nil
}

var storeUnlockers = newUnlockRegistry()

type unlockRegistry struct {
	items map[*Store]func() error
	gate  chan struct{}
}

func newUnlockRegistry() *unlockRegistry {
	r := &unlockRegistry{items: make(map[*Store]func() error), gate: make(chan struct{}, 1)}
	r.gate <- struct{}{}
	return r
}

func (r *unlockRegistry) Store(store *Store, unlock func() error) {
	<-r.gate
	r.items[store] = unlock
	r.gate <- struct{}{}
}

func (r *unlockRegistry) LoadAndDelete(store *Store) func() error {
	<-r.gate
	unlock := r.items[store]
	delete(r.items, store)
	r.gate <- struct{}{}
	return unlock
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		if s.done != nil {
			close(s.done)
		}
		var dbErr error
		if s.db != nil {
			dbErr = s.db.Close()
		}
		var lockErr error
		if unlock := storeUnlockers.LoadAndDelete(s); unlock != nil {
			lockErr = unlock()
		}
		if s.lease != nil {
			s.lease.release()
		}
		s.closeErr = errors.Join(dbErr, lockErr)
	})
	return s.closeErr
}

func (s *Store) DB() *sql.DB { return s.db }

func sqliteDSN(path string, busyTimeout time.Duration) string {
	milliseconds := busyTimeout.Milliseconds()
	query := url.Values{}
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", milliseconds))
	query.Add("_pragma", "foreign_keys(ON)")
	return "file:" + filepath.ToSlash(filepath.Clean(path)) + "?" + query.Encode()
}
