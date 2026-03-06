package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogAndEntries(t *testing.T) {
	l := New(Options{MaxEntries: 50})
	defer l.Close()

	fixedTime := time.Date(2026, 3, 1, 14, 30, 25, 0, time.Local)
	l.now = func() time.Time { return fixedTime }

	l.Log(LevelInfo, CatStartup, "snappy started")
	l.Log(LevelInfo, CatRefresh, "refresh complete")

	entries := l.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries() len = %d, want 2", len(entries))
	}

	if entries[0].Level != LevelInfo {
		t.Errorf("entries[0].Level = %q, want %q", entries[0].Level, LevelInfo)
	}
	if entries[0].Category != CatStartup {
		t.Errorf("entries[0].Category = %q, want %q", entries[0].Category, CatStartup)
	}
	if entries[0].Message != "snappy started" {
		t.Errorf("entries[0].Message = %q, want %q", entries[0].Message, "snappy started")
	}
	if !strings.Contains(entries[0].Formatted, "[14:30:25]") {
		t.Errorf("entries[0].Formatted = %q, want to contain timestamp", entries[0].Formatted)
	}
	if !strings.Contains(entries[0].Formatted, "INFO") {
		t.Errorf("entries[0].Formatted = %q, want to contain INFO", entries[0].Formatted)
	}
	if !strings.Contains(entries[0].Formatted, "STARTUP") {
		t.Errorf("entries[0].Formatted = %q, want to contain STARTUP", entries[0].Formatted)
	}
}

func TestRingBufferLimit(t *testing.T) {
	l := New(Options{MaxEntries: 3})
	defer l.Close()

	for i := range 5 {
		l.Log(LevelInfo, CatRefresh, strings.Repeat("x", i+1))
	}

	entries := l.Entries()
	if len(entries) != 3 {
		t.Fatalf("Entries() len = %d, want 3", len(entries))
	}

	// Should have the last 3 entries
	if entries[0].Message != "xxx" {
		t.Errorf("entries[0].Message = %q, want %q", entries[0].Message, "xxx")
	}
}

func TestEntriesReturnsCopy(t *testing.T) {
	l := New(Options{MaxEntries: 50})
	defer l.Close()

	l.Log(LevelInfo, CatRefresh, "test")
	entries := l.Entries()
	entries[0].Message = "modified"

	original := l.Entries()
	if original[0].Message != "test" {
		t.Error("Entries() did not return a copy")
	}
}

func TestFileLogging(t *testing.T) {
	dir := t.TempDir()
	l := New(Options{LogDir: dir, MaxEntries: 50})
	defer l.Close()

	l.Log(LevelInfo, CatRefresh, "test message")
	l.Close()

	content, err := os.ReadFile(filepath.Join(dir, "snappy.log"))
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(content), "test message") {
		t.Errorf("log file content = %q, want to contain %q", content, "test message")
	}
}

func TestRingBufferBackingArrayDoesNotGrow(t *testing.T) {
	l := New(Options{MaxEntries: 3})
	defer l.Close()

	// Fill the buffer.
	for i := range 3 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("msg-%d", i))
	}

	// Record the capacity after filling.
	l.mu.Lock()
	capAfterFill := cap(l.entries)
	l.mu.Unlock()

	// Write many more entries, cycling through the buffer.
	for i := range 100 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("overflow-%d", i))
	}

	l.mu.Lock()
	capAfterOverflow := cap(l.entries)
	l.mu.Unlock()

	if capAfterOverflow != capAfterFill {
		t.Errorf("backing array capacity grew from %d to %d, want no growth", capAfterFill, capAfterOverflow)
	}

	entries := l.Entries()
	if len(entries) != 3 {
		t.Fatalf("Entries() len = %d, want 3", len(entries))
	}
	// Should have the last 3 entries.
	if entries[0].Message != "overflow-97" {
		t.Errorf("entries[0].Message = %q, want %q", entries[0].Message, "overflow-97")
	}
}

func TestNoFileLogging(t *testing.T) {
	l := New(Options{MaxEntries: 50})
	defer l.Close()

	// Should not panic when file is nil
	l.Log(LevelInfo, CatRefresh, "test")
}

func TestRotationCreatesBackups(t *testing.T) {
	dir := t.TempDir()
	// 100 byte max to trigger rotation quickly.
	l := New(Options{LogDir: dir, MaxEntries: 50, MaxSize: 100, MaxFiles: 3})
	defer l.Close()

	for i := range 20 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("message number %03d with padding to fill space", i))
	}
	l.Close()

	if _, err := os.Stat(filepath.Join(dir, "snappy.log")); err != nil {
		t.Error("snappy.log should exist after rotation")
	}
	if _, err := os.Stat(filepath.Join(dir, "snappy.log.1")); err != nil {
		t.Error("snappy.log.1 should exist after rotation")
	}
}

func TestRotationDeletesOldestBackup(t *testing.T) {
	dir := t.TempDir()
	l := New(Options{LogDir: dir, MaxEntries: 50, MaxSize: 50, MaxFiles: 2})
	defer l.Close()

	for i := range 50 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("entry-%03d-padding-to-exceed-fifty-bytes-easily", i))
	}
	l.Close()

	if _, err := os.Stat(filepath.Join(dir, "snappy.log.2")); err != nil {
		t.Error("snappy.log.2 should exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "snappy.log.3")); !os.IsNotExist(err) {
		t.Error("snappy.log.3 should NOT exist with MaxFiles=2")
	}
}

func TestNoRotationWhenMaxSizeZero(t *testing.T) {
	dir := t.TempDir()
	l := New(Options{LogDir: dir, MaxEntries: 50, MaxSize: 0, MaxFiles: 3})
	defer l.Close()

	for i := range 20 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("message %d", i))
	}
	l.Close()

	if _, err := os.Stat(filepath.Join(dir, "snappy.log.1")); !os.IsNotExist(err) {
		t.Error("no rotation should occur when MaxSize is 0")
	}
}

func TestMaxEntriesZeroDoesNotPanic(t *testing.T) {
	l := New(Options{MaxEntries: 0})
	defer l.Close()

	for i := range 5 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("message %d", i))
	}

	if len(l.Entries()) != 0 {
		t.Error("Entries() should remain empty when MaxEntries is 0")
	}
}

func TestRotationPreservesDataOnRenameFail(t *testing.T) {
	dir := t.TempDir()
	l := New(Options{LogDir: dir, MaxEntries: 50, MaxSize: 100, MaxFiles: 2})
	defer l.Close()

	// Write enough to trigger rotation.
	for i := range 5 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("message-%03d-padding-to-fill-space-easily", i))
	}

	// Make the backup destination directory to block the active rename.
	// os.Rename fails when dst is a non-empty directory.
	backup1 := filepath.Join(dir, "snappy.log.1")
	_ = os.MkdirAll(filepath.Join(backup1, "blocker"), 0o755)

	// Capture pre-rotation content.
	preContent, err := os.ReadFile(filepath.Join(dir, "snappy.log"))
	if err != nil {
		t.Fatalf("reading log before blocked rotation: %v", err)
	}

	// Write more to trigger another rotation (rename will fail).
	for i := range 10 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("after-block-%03d-padding-to-fill-space", i))
	}
	l.Close()

	postContent, err := os.ReadFile(filepath.Join(dir, "snappy.log"))
	if err != nil {
		t.Fatalf("reading log after blocked rotation: %v", err)
	}

	// The file must still contain the pre-rotation data (not truncated).
	if !strings.Contains(string(postContent), string(preContent[:20])) {
		t.Error("log file was truncated despite failed rename; data lost")
	}
}

func TestRotationWithMaxFilesZeroClampsToOneBackup(t *testing.T) {
	dir := t.TempDir()
	l := New(Options{LogDir: dir, MaxEntries: 10, MaxSize: 50, MaxFiles: 0})
	defer l.Close()

	for i := range 30 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("entry-%03d-padding-to-exceed-fifty-bytes-easily", i))
	}
	l.Close()

	if _, err := os.Stat(filepath.Join(dir, "snappy.log.1")); err != nil {
		t.Error("snappy.log.1 should exist when MaxSize is set and MaxFiles is 0")
	}
	if _, err := os.Stat(filepath.Join(dir, "snappy.log.2")); !os.IsNotExist(err) {
		t.Error("snappy.log.2 should NOT exist when MaxFiles is clamped to 1")
	}
}

func TestLoadTailSeedsRingBuffer(t *testing.T) {
	dir := t.TempDir()
	l := New(Options{LogDir: dir, MaxEntries: 50})
	l.Log(LevelInfo, CatStartup, "previous session entry")
	l.Log(LevelWarn, CatThinned, "previous thinning warning")
	l.Close()

	l2 := New(Options{LogDir: dir, MaxEntries: 50})
	defer l2.Close()
	l2.LoadTail()

	entries := l2.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries() len = %d, want 2", len(entries))
	}
	if entries[0].Category != CatStartup {
		t.Errorf("entries[0].Category = %q, want %q", entries[0].Category, CatStartup)
	}
	if entries[0].Level != LevelInfo {
		t.Errorf("entries[0].Level = %q, want %q", entries[0].Level, LevelInfo)
	}
	if entries[1].Level != LevelWarn {
		t.Errorf("entries[1].Level = %q, want %q", entries[1].Level, LevelWarn)
	}
	if entries[1].Message != "previous thinning warning" {
		t.Errorf("entries[1].Message = %q, want %q", entries[1].Message, "previous thinning warning")
	}
}

func TestLoadTailNoFileIsNoop(t *testing.T) {
	l := New(Options{MaxEntries: 50})
	defer l.Close()
	l.LoadTail()
	if len(l.Entries()) != 0 {
		t.Error("LoadTail with no file should leave entries empty")
	}
}

func TestLoadTailTruncatesToMaxEntries(t *testing.T) {
	dir := t.TempDir()
	l := New(Options{LogDir: dir, MaxEntries: 100})
	for i := range 20 {
		l.Log(LevelInfo, CatRefresh, fmt.Sprintf("line-%d", i))
	}
	l.Close()

	l2 := New(Options{LogDir: dir, MaxEntries: 5})
	defer l2.Close()
	l2.LoadTail()

	entries := l2.Entries()
	if len(entries) != 5 {
		t.Fatalf("Entries() len = %d, want 5", len(entries))
	}
	if entries[0].Message != "line-15" {
		t.Errorf("entries[0].Message = %q, want %q", entries[0].Message, "line-15")
	}
}
