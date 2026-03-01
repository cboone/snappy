// Package logger provides a ring-buffer logger with optional file output.
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType identifies the kind of log event.
type EventType string

const (
	Startup EventType = "STARTUP"
	Info    EventType = "INFO"
	Created EventType = "CREATED"
	Added   EventType = "ADDED"
	Removed EventType = "REMOVED"
	Auto    EventType = "AUTO"
	Error   EventType = "ERROR"
	Thinned EventType = "THINNED"
)

// Entry is a single log entry with timestamp, type, and message.
type Entry struct {
	Timestamp time.Time
	Type      EventType
	Message   string
	Formatted string
}

// Logger maintains a capped ring buffer of log entries and optionally
// writes them to a file.
type Logger struct {
	entries []Entry
	maxSize int
	file    *os.File
	mu      sync.Mutex
	now     func() time.Time
}

// New creates a Logger. If logDir is non-empty, it creates the directory
// and opens a log file. File logging failures are non-fatal.
func New(logDir string, maxEntries int) *Logger {
	l := &Logger{
		maxSize: maxEntries,
		now:     time.Now,
	}

	if logDir == "" {
		return l
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot create log directory %s: %v\n", logDir, err)
		return l
	}

	f, err := os.OpenFile(
		filepath.Join(logDir, "snappy.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot open log file: %v\n", err)
		return l
	}
	l.file = f

	return l
}

// Log records an event in the ring buffer and writes to the log file.
func (l *Logger) Log(eventType EventType, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	formatted := fmt.Sprintf("[%s] %-8s %s", now.Format("15:04:05"), eventType, message)

	entry := Entry{
		Timestamp: now,
		Type:      eventType,
		Message:   message,
		Formatted: formatted,
	}

	l.entries = append(l.entries, entry)
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[1:]
	}

	if l.file != nil {
		_, _ = fmt.Fprintln(l.file, formatted)
	}
}

// Entries returns a copy of the current log entries.
func (l *Logger) Entries() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := make([]Entry, len(l.entries))
	copy(result, l.entries)
	return result
}

// Close closes the log file if one is open.
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}
