package main

import (
	"flag"
	"fmt"
	_ "github.com/waxb/kubernites/pkg/client/livescenes"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/crashlog"
	"github.com/waxb/kubernites/pkg/live"
)

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "kubernites:", err)
		os.Exit(1)
	}
}

type options struct {
	kubeconf string
	readonly bool
	version  bool
}

func parseFlags(args []string) (options, error) {
	var o options
	fs := flag.NewFlagSet("kubernites", flag.ContinueOnError)
	fs.StringVar(&o.kubeconf, "kubeconfig", "", "kubeconfig file for LIVE CLUSTER; read only after you confirm the warning. Default: $KUBECONFIG, else ~/.kube/config")
	fs.BoolVar(&o.readonly, "readonly", false, "LIVE CLUSTER only reads in this run: the READ ONLY box stays ticked, whatever was chosen before")
	fs.BoolVar(&o.version, "version", false, "print Kubernites' version and exit")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	return o, nil
}

func run(args []string, env func(string) string, out io.Writer) error {
	o, err := parseFlags(args)
	if err != nil {
		return err
	}
	if o.version {
		_, err := fmt.Fprintln(out, "Kubernites", app.VersionLabel())
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	opts := windowOptions(o, logger, env)
	return crashlog.Watch(opts.SettingsDir, "Kubernites "+app.VersionLabel(), logger, func() error { return app.Run(nil, opts) })
}

func windowOptions(o options, logger *slog.Logger, env func(string) string) app.Options {
	return app.Options{
		Logger:       logger,
		Capture:      env("KUBERNITES_CAPTURE"),
		CaptureScene: env("KUBERNITES_CAPTURE_SCENE"),
		LiveOnly:     true,
		Live:         app.LiveSource{Files: live.KubeconfigPaths(o.kubeconf, env, homeDir())},
		SettingsDir:  settingsDir(env),
		ReadOnly:     o.readonly,
	}
}

func settingsDir(env func(string) string) string {
	if env("KUBERNITES_CAPTURE") != "" {
		return ""
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "kubernites")
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
