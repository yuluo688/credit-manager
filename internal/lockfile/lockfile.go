package lockfile

import "context"

// Locker acquires an exclusive OS file lock for the SQLite writer process.
type Locker struct{}

func New() *Locker { return &Locker{} }

func (l *Locker) Lock(ctx context.Context, path string) (unlock func() error, err error) {
	return lock(ctx, path)
}
