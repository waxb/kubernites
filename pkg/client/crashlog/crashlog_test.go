package crashlog

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

const childDirEnv = "KUBERNITES_CRASHLOG_CHILD_DIR"

var when = time.Date(2026, 11, 25, 9, 30, 0, 0, time.UTC)

func resetCrashOutput(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if err := debug.SetCrashOutput(nil, debug.CrashOptions{}); err != nil {
			t.Errorf("reset the crash output: %v", err)
		}
	})
}

func readLog(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func startLine(label string) string {
	return "2026-11-25T09:30:00Z " + label + " started on " + runtime.GOOS + "/" + runtime.GOARCH + "\n"
}

func TestStartMakesItsFolderAndNamesTheBuildThatStarted(t *testing.T) {
	resetCrashOutput(t)
	dir := filepath.Join(t.TempDir(), "kubernites")
	l, err := Start(dir, "Kubernites v9.9.9 (abc1234)", when)
	if err != nil {
		t.Fatal(err)
	}
	if l.Path() != filepath.Join(dir, FileName) {
		t.Errorf("the log is at %q, want %q", l.Path(), filepath.Join(dir, FileName))
	}
	if got, want := readLog(t, dir), startLine("Kubernites v9.9.9 (abc1234)"); got != want {
		t.Errorf("the log holds %q, want %q", got, want)
	}
}

func TestWithoutAFolderNoLogIsKept(t *testing.T) {
	l, err := Start("", "Kubernites", when)
	if l != nil || err != nil {
		t.Fatalf("Start without a folder gave %v, %v; want no log and no error", l, err)
	}
	if err := l.Note(errors.New("boom"), when); err != nil {
		t.Errorf("a note on no log failed: %v", err)
	}
	if l.Path() != "" {
		t.Errorf("no log has the path %q", l.Path())
	}
}

func TestStartKeepsAnEarlierRunAndTrimsALogPastItsLimit(t *testing.T) {
	line := strings.Repeat("x", 99) + "\n"
	cases := []struct {
		name    string
		before  string
		keepAll bool
	}{
		{"a log under the limit is kept whole", strings.Repeat(line, 100), true},
		{"a log past the limit keeps its tail from a line's start", "a cut line" + strings.Repeat(line, maxBytes/len(line)+10), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetCrashOutput(t)
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(c.before), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Start(dir, "Kubernites", when); err != nil {
				t.Fatal(err)
			}
			got := readLog(t, dir)
			if !strings.HasSuffix(got, startLine("Kubernites")) {
				t.Fatalf("the log does not end with the new run's line: %q", got[max(0, len(got)-200):])
			}
			kept := strings.TrimSuffix(got, startLine("Kubernites"))
			if c.keepAll {
				if kept != c.before {
					t.Errorf("the earlier run was not kept whole: %d bytes, want %d", len(kept), len(c.before))
				}
				return
			}
			if len(kept) > keptBytes || len(kept) < keptBytes-len(line) {
				t.Errorf("the trimmed log keeps %d bytes, want about %d", len(kept), keptBytes)
			}
			if !strings.HasPrefix(kept, line) || !strings.HasSuffix(c.before, kept) {
				t.Errorf("the trimmed log is not the earlier log's tail from a line's start")
			}
		})
	}
}

func TestWatchReturnsTheRunsErrorAndWritesItDown(t *testing.T) {
	resetCrashOutput(t)
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	want := errors.New("run the client: no window")
	if err := Watch(dir, "Kubernites", logger, func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("Watch returned %v, want %v", err, want)
	}
	got := readLog(t, dir)
	if !strings.Contains(got, " ended with an error: run the client: no window\n") {
		t.Errorf("the log does not say how the run ended: %q", got)
	}
	if err := Watch(dir, "Kubernites", logger, func() error { return nil }); err != nil {
		t.Fatalf("a clean run returned %v", err)
	}
	if strings.Count(readLog(t, dir), "ended with an error") != 1 {
		t.Errorf("a clean run wrote an error line: %q", readLog(t, dir))
	}
}

func TestALogThatCannotBeOpenedNeverStopsTheRun(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "a file")
	if err := os.WriteFile(blocked, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var warnings strings.Builder
	logger := slog.New(slog.NewTextHandler(&warnings, nil))
	ran := false
	if err := Watch(filepath.Join(blocked, "kubernites"), "Kubernites", logger, func() error { ran = true; return nil }); err != nil {
		t.Fatalf("Watch returned %v", err)
	}
	if !ran {
		t.Fatal("the run never ran")
	}
	if !strings.Contains(warnings.String(), "crashes are not written to a file") {
		t.Errorf("no warning says the crash log is off: %q", warnings.String())
	}
}

func TestACrashAfterStartLandsInTheLog(t *testing.T) {
	if dir := os.Getenv(childDirEnv); dir != "" {
		if _, err := Start(dir, "Kubernites child", when); err != nil {
			os.Exit(3)
		}
		block := make(chan struct{})
		go func() { panic("the crash the log keeps") }()
		<-block
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestACrashAfterStartLandsInTheLog$")
	cmd.Env = append(os.Environ(), childDirEnv+"="+dir)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 {
		t.Fatalf("the child did not crash with exit code 2: %v\n%s", err, out)
	}
	got := readLog(t, dir)
	for _, want := range []string{startLine("Kubernites child"), "panic: the crash the log keeps", "goroutine "} {
		if !strings.Contains(got, want) {
			t.Errorf("the log lacks %q:\n%s", want, got)
		}
	}
}
