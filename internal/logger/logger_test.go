package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogAndEntries(t *testing.T) {
	l := New("", 50)
	defer l.Close()

	fixedTime := time.Date(2026, 3, 1, 14, 30, 25, 0, time.Local)
	l.now = func() time.Time { return fixedTime }

	l.Log(Startup, "snappy started")
	l.Log(Info, "refresh complete")

	entries := l.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries() len = %d, want 2", len(entries))
	}

	if entries[0].Type != Startup {
		t.Errorf("entries[0].Type = %q, want %q", entries[0].Type, Startup)
	}
	if entries[0].Message != "snappy started" {
		t.Errorf("entries[0].Message = %q, want %q", entries[0].Message, "snappy started")
	}
	if !strings.Contains(entries[0].Formatted, "[14:30:25]") {
		t.Errorf("entries[0].Formatted = %q, want to contain timestamp", entries[0].Formatted)
	}
	if !strings.Contains(entries[0].Formatted, "STARTUP") {
		t.Errorf("entries[0].Formatted = %q, want to contain STARTUP", entries[0].Formatted)
	}
}

func TestRingBufferLimit(t *testing.T) {
	l := New("", 3)
	defer l.Close()

	for i := range 5 {
		l.Log(Info, strings.Repeat("x", i+1))
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
	l := New("", 50)
	defer l.Close()

	l.Log(Info, "test")
	entries := l.Entries()
	entries[0].Message = "modified"

	original := l.Entries()
	if original[0].Message != "test" {
		t.Error("Entries() did not return a copy")
	}
}

func TestFileLogging(t *testing.T) {
	dir := t.TempDir()
	l := New(dir, 50)
	defer l.Close()

	l.Log(Info, "test message")
	l.Close()

	content, err := os.ReadFile(filepath.Join(dir, "snappy.log"))
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(content), "test message") {
		t.Errorf("log file content = %q, want to contain %q", content, "test message")
	}
}

func TestNoFileLogging(t *testing.T) {
	l := New("", 50)
	defer l.Close()

	// Should not panic when file is nil
	l.Log(Info, "test")
}
