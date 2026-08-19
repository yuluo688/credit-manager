package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	storeOpenProbeTimeout    = 100 * time.Millisecond
	storeOpenHandoverTimeout = 5 * time.Second
	storeLeasePollInterval   = 50 * time.Millisecond
)

type storeLease struct {
	path  string
	token string
	pid   int
}

func HandoverPath(databasePath string) string {
	return filepath.Clean(databasePath) + ".handover"
}

func newStoreLease(databasePath string) (*storeLease, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return nil, fmt.Errorf("create database handover token: %w", err)
	}
	pid := os.Getpid()
	return &storeLease{
		path:  HandoverPath(databasePath),
		token: fmt.Sprintf("%d-%s", pid, hex.EncodeToString(random[:])),
		pid:   pid,
	}, nil
}

func (l *storeLease) claim() error {
	if l == nil {
		return errors.New("database handover lease is nil")
	}
	if err := os.WriteFile(l.path, []byte(l.token+"\n"), 0o600); err != nil {
		return fmt.Errorf("write database handover marker: %w", err)
	}
	return nil
}

func (l *storeLease) current() bool {
	if l == nil {
		return false
	}
	token, err := readStoreLeaseToken(l.path)
	return err == nil && token == l.token
}

func (l *storeLease) release() {
	if l == nil || !l.current() {
		return
	}
	_ = os.Remove(l.path)
}

func readStoreLeaseToken(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func storeLeaseTokenPID(token string) (int, bool) {
	rawPID, _, ok := strings.Cut(token, "-")
	if !ok {
		return 0, false
	}
	pid, err := strconv.Atoi(rawPID)
	return pid, err == nil && pid > 0
}

func lockWithTimeout(ctx context.Context, locker FileLock, databasePath string, timeout time.Duration) (func() error, error) {
	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return locker.Lock(lockCtx, LockPath(databasePath))
}

func acquireWriterLock(ctx context.Context, locker FileLock, databasePath string, lease *storeLease) (func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	unlock, err := lockWithTimeout(ctx, locker, databasePath, storeOpenProbeTimeout)
	if err == nil {
		if claimErr := lease.claim(); claimErr != nil {
			_ = unlock()
			return nil, claimErr
		}
		return unlock, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("acquire database lock: %w", err)
	}
	if err := lease.claim(); err != nil {
		return nil, err
	}
	unlock, err = lockWithTimeout(ctx, locker, databasePath, storeOpenHandoverTimeout)
	if err != nil {
		lease.release()
		return nil, fmt.Errorf("hot-reload handover timed out; restart CLIProxyAPI or reinstall once when upgrading from a legacy plugin: %w", err)
	}
	return unlock, nil
}

func (s *Store) attachLease(lease *storeLease, unlock func() error) {
	s.lease = lease
	s.done = make(chan struct{})
	storeUnlockers.Store(s, unlock)
	go lease.monitor(s)
}

func (l *storeLease) monitor(store *Store) {
	if l == nil || store == nil || store.done == nil {
		return
	}
	ticker := time.NewTicker(storeLeasePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-store.done:
			return
		case <-ticker.C:
			token, err := readStoreLeaseToken(l.path)
			if err != nil || token == "" || token == l.token {
				continue
			}
			claimPID, ok := storeLeaseTokenPID(token)
			if !ok || claimPID != l.pid {
				continue
			}
			_ = store.Close()
			return
		}
	}
}
