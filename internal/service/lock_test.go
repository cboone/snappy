package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := Acquire(path)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer func() { _ = lock.Release() }()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock file not created: %v", err)
	}
}

func TestAcquireReturnsErrLockedWhenHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	first, err := Acquire(path)
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	defer func() { _ = first.Release() }()

	_, err = Acquire(path)
	if err == nil {
		t.Fatal("second Acquire() should have failed")
	}
	if err != ErrLocked {
		t.Fatalf("second Acquire() error = %v, want ErrLocked", err)
	}
}

func TestAcquireCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "test.lock")

	lock, err := Acquire(path)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer func() { _ = lock.Release() }()

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("directory not created: %v", err)
	}
}

func TestReleaseAllowsReacquire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	first, err := Acquire(path)
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	second, err := Acquire(path)
	if err != nil {
		t.Fatalf("re-Acquire() error = %v", err)
	}
	_ = second.Release()
}

func TestReleaseIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := Acquire(path)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("first Release() error = %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("second Release() error = %v (should be nil)", err)
	}
}

func TestIsHeldReturnsTrueWhenLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := Acquire(path)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer func() { _ = lock.Release() }()

	if !IsHeld(path) {
		t.Error("IsHeld() = false, want true")
	}
}

func TestIsHeldReturnsFalseWhenReleased(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := Acquire(path)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	_ = lock.Release()

	if IsHeld(path) {
		t.Error("IsHeld() = true, want false")
	}
}

func TestIsHeldReturnsFalseWhenFileDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.lock")

	if IsHeld(path) {
		t.Error("IsHeld() = true for nonexistent file, want false")
	}
}

func TestIsHeldDoesNotBlockAcquire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	// Create the lock file so IsHeld can open it.
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating lock file: %v", err)
	}
	_ = f.Close()

	// IsHeld on an unheld lock should not prevent a subsequent Acquire.
	if IsHeld(path) {
		t.Fatal("IsHeld() = true on unheld lock")
	}

	lock, err := Acquire(path)
	if err != nil {
		t.Fatalf("Acquire() after IsHeld() error = %v", err)
	}
	_ = lock.Release()
}

func TestDefaultLockPath(t *testing.T) {
	got := DefaultLockPath("/some/dir")
	want := "/some/dir/snappy.lock"
	if got != want {
		t.Errorf("DefaultLockPath() = %q, want %q", got, want)
	}
}
