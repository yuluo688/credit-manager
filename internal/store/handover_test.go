package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yuluo688/credit-manager/internal/lockfile"
)

func TestOpenLockedHandsDatabaseToNewPluginInstance(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "credit-manager.db")
	opts := OpenOptions{BusyTimeout: time.Second}

	first, err := OpenLocked(ctx, path, opts, lockfile.New())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close() })
	if _, err := first.CreateCaller(ctx, CallerSpec{ID: "handover", QuotaMicroUSD: 0, Enabled: true}); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	second, err := OpenLocked(ctx, path, opts, lockfile.New())
	if err != nil {
		t.Fatalf("open replacement store: %v", err)
	}
	defer second.Close()
	if elapsed := time.Since(started); elapsed >= storeOpenHandoverTimeout {
		t.Fatalf("database handover took %v", elapsed)
	}

	if _, err := first.GetCaller(ctx, "handover"); err == nil {
		t.Fatal("retired store still serves queries")
	}
	got, err := second.GetCaller(ctx, "handover")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "handover" {
		t.Fatalf("replacement store lost persisted caller: %+v", got)
	}
	if second.lease == nil || !second.lease.current() {
		t.Fatal("replacement store does not own the handover lease")
	}
}

func TestOpenLockedAcquiresImmediatelyWhenUnlocked(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "credit-manager.db")
	started := time.Now()
	st, err := OpenLocked(ctx, path, OpenOptions{BusyTimeout: time.Second}, lockfile.New())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if elapsed := time.Since(started); elapsed >= storeOpenHandoverTimeout/2 {
		t.Fatalf("uncontended open took %v", elapsed)
	}
}

func TestOpenLockedIgnoresForeignProcessHandover(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "credit-manager.db")
	st, err := OpenLocked(ctx, path, OpenOptions{BusyTimeout: time.Second}, lockfile.New())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.CreateCaller(ctx, CallerSpec{ID: "keep", QuotaMicroUSD: 0, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(HandoverPath(path), []byte("99999-deadbeef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * storeLeasePollInterval)
	if _, err := st.GetCaller(ctx, "keep"); err != nil {
		t.Fatalf("foreign handover closed the store: %v", err)
	}
}

func TestOpenLockedHandoverTimesOutAgainstLegacyHolder(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "credit-manager.db")
	unlock, err := lockfile.New().Lock(ctx, LockPath(path))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	started := time.Now()
	_, err = OpenLocked(ctx, path, OpenOptions{BusyTimeout: time.Second}, lockfile.New())
	if err == nil {
		t.Fatal("expected handover timeout against a legacy lock holder")
	}
	if elapsed := time.Since(started); elapsed < storeOpenHandoverTimeout {
		t.Fatalf("handover returned too quickly: %v", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "handover timed out") {
		t.Fatalf("error = %v, want handover timeout", err)
	}
}
