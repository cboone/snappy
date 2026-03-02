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

// Event types for log categorization.
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

// Options configures a Logger.
type Options struct {
	LogDir     string // directory for the log file; empty disables file logging
	MaxEntries int    // ring buffer capacity for TUI display
	MaxSize    int64  // max log file size in bytes before rotation; 0 disables rotation
	MaxFiles   int    // number of rotated backup files to keep
}

// Logger maintains a capped ring buffer of log entries and optionally
// writes them to a file with size-based rotation.
type Logger struct {
	entries  []Entry
	maxSize  int
	file     *os.File
	filePath string // full path to snappy.log
	maxBytes int64  // max file size before rotation; 0 = no rotation
	maxFiles int    // number of backup files to keep
	mu       sync.Mutex
	now      func() time.Time
}

// New creates a Logger. If LogDir is non-empty, it creates the directory
// and opens a log file. File logging failures are non-fatal.
func New(opts Options) *Logger {
	maxEntries := max(opts.MaxEntries, 0)
	maxBytes := max(opts.MaxSize, 0)

	maxFiles := opts.MaxFiles
	if maxBytes > 0 && maxFiles <= 0 {
		maxFiles = 1
	}

	l := &Logger{
		entries:  make([]Entry, 0, maxEntries),
		maxSize:  maxEntries,
		maxBytes: maxBytes,
		maxFiles: maxFiles,
		now:      time.Now,
	}

	if opts.LogDir == "" {
		return l
	}

	if err := os.MkdirAll(opts.LogDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot create log directory %s: %v\n", opts.LogDir, err)
		return l
	}

	logPath := filepath.Join(opts.LogDir, "snappy.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot open log file: %v\n", err)
		return l
	}
	l.file = f
	l.filePath = logPath

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

	if l.maxSize > 0 {
		if len(l.entries) < l.maxSize {
			l.entries = append(l.entries, entry)
		} else {
			copy(l.entries, l.entries[1:])
			l.entries[l.maxSize-1] = entry
		}
	}

	if l.file != nil {
		l.maybeRotate()
		if l.file != nil {
			_, _ = fmt.Fprintln(l.file, formatted)
		}
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

// maybeRotate checks the current log file size and rotates if it reaches or
// exceeds maxBytes. Must be called with l.mu held.
func (l *Logger) maybeRotate() {
	if l.maxBytes <= 0 {
		return
	}

	info, err := l.file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot stat log file: %v\n", err)
		return
	}

	if info.Size() < l.maxBytes {
		return
	}

	l.rotateFiles()
}

// rotateFiles closes the current file, shifts backups, and opens a fresh log.
// If the active log file cannot be renamed aside, it reopens in append mode
// to avoid truncating data that was never backed up.
// Must be called with l.mu held.
func (l *Logger) rotateFiles() {
	_ = l.file.Close()
	l.file = nil

	// Shift existing backups from highest to lowest.
	// Example with maxFiles=3: delete .3, rename .2->.3, .1->.2, .log->.1
	activeRenamed := true
	for i := l.maxFiles; i >= 1; i-- {
		src := l.backupPath(i - 1)
		dst := l.backupPath(i)
		if i == l.maxFiles {
			_ = os.Remove(dst)
		}
		if err := os.Rename(src, dst); err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: cannot rename %s to %s: %v\n", src, dst, err)
				if i == 1 {
					activeRenamed = false
				}
			}
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if activeRenamed {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}

	f, err := os.OpenFile(l.filePath, flags, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot open new log file after rotation: %v\n", err)
		return
	}
	l.file = f
}

// backupPath returns the file path for backup number n.
// n=0 is the active log file, n>=1 produces .log.1, .log.2, etc.
func (l *Logger) backupPath(n int) string {
	if n == 0 {
		return l.filePath
	}
	return fmt.Sprintf("%s.%d", l.filePath, n)
}
