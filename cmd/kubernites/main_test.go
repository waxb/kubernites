package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/app"
)

func TestTheLiveBinaryOpensLiveOnlyAndKeepsNoSave(t *testing.T) {
	env := func(k string) string {
		if k == "HOME" || k == "KUBECONFIG" {
			return ""
		}
		return ""
	}
	o := windowOptions(options{kubeconf: "/tmp/kc"}, slog.New(slog.NewTextHandler(io.Discard, nil)), env)
	if !o.LiveOnly || o.SavePath != "" || o.DevDir != "" || o.HasSave || o.NewRealm != nil {
		t.Fatalf("the live binary must open live only and keep no save: %+v", o)
	}
	if o.ReadOnly {
		t.Fatal("READ ONLY was forced without --readonly")
	}
	if dir, err := os.UserConfigDir(); err == nil && o.SettingsDir != filepath.Join(dir, "kubernites") {
		t.Fatalf("settings, live choices and ledgers go to %q", o.SettingsDir)
	}
	capture := func(k string) string {
		if k == "KUBERNITES_CAPTURE" {
			return "/tmp/x.png"
		}
		return ""
	}
	if o := windowOptions(options{}, slog.New(slog.NewTextHandler(io.Discard, nil)), capture); o.SettingsDir != "" {
		t.Fatalf("a capture writes no file, got %q", o.SettingsDir)
	}
	if len(o.Live.Files) != 1 || o.Live.Files[0] != "/tmp/kc" {
		t.Fatalf("-kubeconfig is not the only file read: %v", o.Live.Files)
	}
}

func TestReadonlyForcesReadOnlyForTheRun(t *testing.T) {
	o, err := parseFlags([]string{"--readonly"})
	if err != nil || !o.readonly {
		t.Fatalf("--readonly gives %+v, %v", o, err)
	}
	if w := windowOptions(o, slog.New(slog.NewTextHandler(io.Discard, nil)), func(string) string { return "" }); !w.ReadOnly {
		t.Fatal("--readonly did not reach Kubernites")
	}
}

func TestVersionPrintsTheVersionAndOpensNoWindow(t *testing.T) {
	var out strings.Builder
	if err := run([]string{"-version"}, func(string) string { return "" }, &out); err != nil {
		t.Fatalf("-version: %v", err)
	}
	if got, want := out.String(), "Kubernites "+app.VersionLabel()+"\n"; got != want {
		t.Fatalf("-version printed %q, want %q", got, want)
	}
}

func TestItsOnlyFlagsAreKubeconfigReadonlyAndVersion(t *testing.T) {
	if _, err := parseFlags([]string{"-save", "x"}); err == nil || !strings.Contains(err.Error(), "save") {
		t.Fatalf("-save must be refused, got %v", err)
	}
	o, err := parseFlags([]string{"-kubeconfig", "k"})
	if err != nil || o.kubeconf != "k" {
		t.Fatalf("-kubeconfig k gives %+v, %v", o, err)
	}
}
