// Package logger provides a ring-buffer logger with optional file output.
package logger

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level is the severity of a log entry.
type Level string

// Log levels.
const (
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// Category identifies what kind of event occurred.
type Category string

// Event categories.
const (
	CatStartup  Category = "STARTUP"
	CatRefresh  Category = "REFRESH"
	CatSnapshot Category = "SNAPSHOT"
	CatCreated  Category = "CREATED"
	CatAdded    Category = "ADDED"
	CatRemoved  Category = "REMOVED"
	CatAuto     Category = "AUTO"
	CatThinned  Category = "THINNED"
	CatFound    Category = "FOUND"
	CatShutdown Category = "SHUTDOWN"
	CatOpen     Category = "OPEN"
)

// Entry is a single log entry with timestamp, level, category, and message.
type Entry struct {
	Timestamp time.Time
	Level     Level
	Category  Category
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
func (l *Logger) Log(level Level, cat Category, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	formatted := fmt.Sprintf("[%s] %-5s %-8s %s", now.Format("15:04:05"), level, cat, message)

	entry := Entry{
		Timestamp: now,
		Level:     level,
		Category:  cat,
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

// LoadTail reads the last maxSize lines from the log file and seeds the ring
// buffer. Call this after New and before the first Log to give the TUI
// continuity across restarts. Unparseable lines are silently skipped.
func (l *Logger) LoadTail() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.filePath == "" || l.maxSize <= 0 {
		return
	}

	f, err := os.Open(l.filePath)
	if err != nil {
		return
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Keep only the last maxSize lines.
	if len(lines) > l.maxSize {
		lines = lines[len(lines)-l.maxSize:]
	}

	for _, line := range lines {
		if entry, ok := parseLogLine(line, l.now()); ok {
			if len(l.entries) < l.maxSize {
				l.entries = append(l.entries, entry)
			} else {
				copy(l.entries, l.entries[1:])
				l.entries[l.maxSize-1] = entry
			}
		}
	}
}

// parseLogLine parses a formatted log line into an Entry.
// Expected format: "[HH:MM:SS] LEVEL CATEGORY message"
// The date component is taken from refTime since the file only stores time.
func parseLogLine(line string, refTime time.Time) (Entry, bool) {
	// Minimum: "[HH:MM:SS] X Y z" = at least 16 chars
	if len(line) < 16 || line[0] != '[' {
		return Entry{}, false
	}

	closeBracket := strings.IndexByte(line, ']')
	if closeBracket < 0 {
		return Entry{}, false
	}

	timeStr := line[1:closeBracket]
	t, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		return Entry{}, false
	}
	// Combine parsed time-of-day with refTime's date.
	ts := time.Date(refTime.Year(), refTime.Month(), refTime.Day(),
		t.Hour(), t.Minute(), t.Second(), 0, refTime.Location())

	rest := line[closeBracket+1:]
	rest = strings.TrimLeft(rest, " ")

	// Split into level, category, and message.
	fields := strings.SplitN(rest, " ", 3)

	var level Level
	var cat Category
	var message string

	switch len(fields) {
	case 3:
		level = Level(strings.TrimSpace(fields[0]))
		remaining := strings.TrimLeft(fields[1]+" "+fields[2], " ")
		catAndMsg := strings.SplitN(remaining, " ", 2)
		cat = Category(strings.TrimSpace(catAndMsg[0]))
		if len(catAndMsg) > 1 {
			message = strings.TrimLeft(catAndMsg[1], " ")
		}
	case 2:
		// Old format: "[HH:MM:SS] TYPE     message" - treat TYPE as category, default to INFO.
		level = LevelInfo
		cat = Category(strings.TrimSpace(fields[0]))
		message = strings.TrimLeft(fields[1], " ")
	default:
		return Entry{}, false
	}

	return Entry{
		Timestamp: ts,
		Level:     level,
		Category:  cat,
		Message:   message,
		Formatted: line,
	}, true
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
