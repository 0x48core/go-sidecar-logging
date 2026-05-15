package logger

import (
	"fmt"
	"os"
	"sync"
)

// FileLogger defines the contract for thread-safe append writes.
// Defined at the consumer side so background/log_writer.go owns the interface.
type FileLogger interface {
	WriteLine(line string) error
}

type fileLogger struct {
	mu   sync.Mutex
	path string
}

// NewFileLogger returns a FileLogger that appends to the file at path.
// The file and any missing parent directories are created on first write.
func NewFileLogger(path string) FileLogger {
	return &fileLogger{
		path: path,
	}
}

// WriteLine acquires the mutex, appends line + newline to the file, then releases.
// Open/close on every call is intentional: avoids stale file handles across
// container restarts and keeps the critical section as short as possible.
func (l *fileLogger) WriteLine(line string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, line)
	return err
}
