//go:build windows

package lockfile

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func lock(ctx context.Context, path string) (func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	handle := windows.Handle(file.Fd())
	var overlapped windows.Overlapped
	deadline := time.Now().Add(30 * time.Second)
	for {
		err = windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlapped)
		if err == nil {
			return func() error {
				_ = windows.UnlockFileEx(handle, 0, 1, 0, &overlapped)
				return file.Close()
			}, nil
		}
		if err := ctx.Err(); err != nil {
			_ = file.Close()
			return nil, err
		}
		if time.Now().After(deadline) {
			_ = file.Close()
			return nil, fmt.Errorf("timeout acquiring lock %s: %w", path, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
