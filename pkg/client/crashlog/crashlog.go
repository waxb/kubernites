package crashlog

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"
)

const (
	FileName  = "crash.log"
	maxBytes  = 1 << 20
	keptBytes = 256 << 10
)

type Log struct {
	path string
}

func Watch(dir, label string, logger *slog.Logger, run func() error) error {
	l, err := Start(dir, label, time.Now())
	if err != nil {
		logger.Warn("crashes are not written to a file", "err", err)
	}
	runErr := run()
	if err := l.Note(runErr, time.Now()); err != nil {
		logger.Warn("the error was not written to the crash log", "err", err)
	}
	return runErr
}

func Start(dir, label string, now time.Time) (*Log, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("make the crash log's folder: %w", err)
	}
	path := filepath.Join(dir, FileName)
	if err := trim(path); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open the crash log: %w", err)
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s %s started on %s/%s\n", stamp(now), label, runtime.GOOS, runtime.GOARCH); err != nil {
		return nil, fmt.Errorf("write to the crash log: %w", err)
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		return nil, fmt.Errorf("send crashes to the crash log: %w", err)
	}
	return &Log{path: path}, nil
}

func (l *Log) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *Log) Note(runErr error, now time.Time) error {
	if l == nil || runErr == nil {
		return nil
	}
	f, err := os.OpenFile(l.path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open the crash log: %w", err)
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s ended with an error: %v\n", stamp(now), runErr); err != nil {
		return fmt.Errorf("write to the crash log: %w", err)
	}
	return nil
}

func stamp(now time.Time) string {
	return now.UTC().Format(time.RFC3339)
}

func trim(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("look at the crash log: %w", err)
	}
	if info.Size() <= maxBytes {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read the crash log: %w", err)
	}
	if len(data) <= maxBytes {
		return nil
	}
	tail := data[len(data)-keptBytes:]
	if i := bytes.IndexByte(tail, '\n'); i >= 0 {
		tail = tail[i+1:]
	}
	if err := os.WriteFile(path, tail, 0o644); err != nil {
		return fmt.Errorf("trim the crash log: %w", err)
	}
	return nil
}
