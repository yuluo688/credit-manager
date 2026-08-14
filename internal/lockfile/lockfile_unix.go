//go:build !windows

package lockfile

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"
)

func lock(ctx context.Context, path string) (func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return func() error {
				_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
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
