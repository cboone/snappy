// Package service provides single-instance locking and launchd service
// management for snappy.
package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ErrLocked is returned when the lock is already held by another process.
var ErrLocked = errors.New("another snappy auto-snapshot process is already running")

// LockFile represents an exclusive file-based lock using flock.
// The lock is automatically released by the OS when the process exits.
type LockFile struct {
	path string
	file *os.File
}

// DefaultLockPath returns the lock file path within the given log directory.
// If logDir is empty, it falls back to the OS temporary directory.
func DefaultLockPath(logDir string) string {
	if logDir == "" {
		logDir = os.TempDir()
	}
	return filepath.Join(logDir, "snappy.lock")
}

// Acquire attempts to take an exclusive non-blocking lock on the file at path.
// The parent directory is created if it does not exist.
// Returns a LockFile on success, or ErrLocked if another process holds the lock.
func Acquire(path string) (*LockFile, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating lock directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}

	return &LockFile{path: path, file: f}, nil
}

// Release releases the lock and closes the underlying file.
func (l *LockFile) Release() error {
	if l.file == nil {
		return nil
	}
	// Unlock then close. Closing also releases the lock, but being explicit
	// avoids relying on close-time side effects.
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err := l.file.Close()
	l.file = nil
	return err
}

// IsHeld checks whether the lock at path is currently held by another process.
// It probes by attempting a shared (LOCK_SH) lock; if the attempt is blocked
// by an exclusive lock, the lock is held. Using LOCK_SH instead of LOCK_EX
// prevents concurrent IsHeld probes from interfering with each other.
// Note: Acquire uses an exclusive flock and may briefly see ErrLocked
// if it races with an IsHeld probe holding a shared lock.
// Returns false if the file does not exist.
func IsHeld(path string) bool {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	if err != nil {
		return errors.Is(err, syscall.EWOULDBLOCK)
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false
}
