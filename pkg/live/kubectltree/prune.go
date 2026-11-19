package kubectltree

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/version"
	apimachineryversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/cmd/auth"
	cmdconfig "k8s.io/kubectl/pkg/cmd/config"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/yaml"

	"github.com/waxb/kubernites/pkg/live/write"
)

const (
	ReasonPlugins    = "plugins are off in the Stone: a plugin is another program"
	ReasonCompletion = "shell completion is not part of the Stone: the Stone completes with Tab"
	ReasonDiff       = "kubectl diff runs an outside diff program, which Kubernites never starts"
	ReasonKustomize  = "kustomize is not part of the Stone: it can run git and other programs"
	ReasonProxy      = "kubectl proxy is not part of the Stone: it would open a door to the API server outside Kubernites' gate"
	ReasonEditor     = "this opens an outside editor; kubectl edit opens the Scriptorium instead"
	ReasonKubeconfig = "the kubeconfig is read, never written"
	ReasonCredential = "the kubeconfig's credentials stay out of the Stone"
	ReasonConnection = "the Stone speaks only to the connected cluster, through the kubeconfig and context chosen at connect"
	ReasonProfile    = "profiles would be written to files"
	ReasonDumpFiles  = "cluster-info dump writes to the Stone, not to a directory"
	ReasonManager    = "the Scriptorium sends its changes with server-side apply as kubernites"
	ReasonLeave      = "the Stone's terminal is always left with ctrl-p ctrl-q"
	ReasonDebugFile  = "kubectl debug takes Pod or Node names in the Stone: with -f, kubectl's own debug ends the whole program on a Pod file"
)

var removed = map[string]string{
	"plugin":     ReasonPlugins,
	"completion": ReasonCompletion,
	"diff":       ReasonDiff,
	"kustomize":  ReasonKustomize,
	"proxy":      ReasonProxy,
	"kuberc":     "kuberc is off in the Stone",
}

var connectionFlags = []string{
	"kubeconfig", "context", "cluster", "user", "server", "token", "as", "as-uid", "as-group", "as-user-extra",
	"username", "password", "certificate-authority", "client-certificate", "client-key", "insecure-skip-tls-verify",
	"tls-server-name", "proxy-url", "cache-dir", "kuberc",
}

var configWrites = []string{"set-cluster", "set-credentials", "set-context", "set", "unset", "use-context", "delete-cluster", "delete-context", "delete-user", "rename-context"}

func (t *Tree) prune(root *cobra.Command, cf *genericclioptions.ConfigFlags, ios genericiooptions.IOStreams, r *run) {
	for name, reason := range removed {
		if c := child(root, name); c != nil {
			stub(c, refuser(r, reason))
		}
	}
	for _, name := range []string{cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd} {
		c := &cobra.Command{Use: name}
		stub(c, refuser(r, ReasonCompletion))
		root.AddCommand(c)
	}
	if c := child(root, "config"); c != nil {
		root.RemoveCommand(c)
		root.AddCommand(t.config(cf, ios, r))
	}
	if c := child(child(root, "apply"), "edit-last-applied"); c != nil {
		stub(c, refuser(r, ReasonEditor))
	}
	if c := child(root, "edit"); c != nil {
		routeEdit(c, r)
	}
	for _, name := range []string{"delete", "apply"} {
		if c := child(root, name); c != nil {
			destroying(c, r)
		}
	}
	if c := child(child(root, "auth"), "can-i"); c != nil {
		canI(c, cf, ios)
	}
	if c := child(root, "version"); c != nil {
		versionRun(c, cf, ios)
	}
	for _, name := range connectionFlags {
		lockFlag(root.PersistentFlags().Lookup(name), ReasonConnection, r)
	}
	for _, name := range []string{"profile", "profile-output"} {
		lockFlag(root.PersistentFlags().Lookup(name), ReasonProfile, r)
	}
	if c := child(root, "create"); c != nil {
		lockFlag(c.Flags().Lookup("edit"), ReasonEditor, r)
	}
	for _, name := range []string{"attach", "run"} {
		if c := child(root, name); c != nil {
			lockFlag(c.Flags().Lookup("detach-keys"), ReasonLeave, r)
		}
	}
	if c := child(child(root, "cluster-info"), "dump"); c != nil {
		lockFlag(c.Flags().Lookup("output-directory"), ReasonDumpFiles, r)
	}
}

func guardFiles(root *cobra.Command, r *run) {
	if c := child(root, "debug"); c != nil {
		lockFlag(c.Flags().Lookup("filename"), ReasonDebugFile, r)
	}
	walk(root, func(c *cobra.Command) {
		for _, fs := range []*pflag.FlagSet{c.LocalNonPersistentFlags(), c.PersistentFlags()} {
			if f := fs.Lookup("kustomize"); f != nil {
				lockFlag(f, ReasonKustomize, r)
			}
			if f := fs.Lookup("filename"); f != nil {
				if _, ok := f.Value.(guardedFiles); !ok {
					f.Value = guardedFiles{Value: f.Value, run: r}
				}
			}
			for _, name := range []string{"from-file", "from-env-file", "custom", "cert", "key"} {
				if f := fs.Lookup(name); f != nil {
					if _, ok := f.Value.(guardedFiles); !ok {
						f.Value = guardedFiles{Value: f.Value, run: r, keyed: name == "from-file"}
					}
				}
			}
		}
	})
}

func destroying(c *cobra.Command, r *run) {
	declare := func(cmd *cobra.Command, args []string) {
		if severalAtOnce(cmd, args) {
			r.declare(write.Destroy)
		}
	}
	if run := c.Run; run != nil {
		c.Run = func(cmd *cobra.Command, args []string) {
			declare(cmd, args)
			run(cmd, args)
		}
	}
	if run := c.RunE; run != nil {
		c.RunE = func(cmd *cobra.Command, args []string) error {
			declare(cmd, args)
			return run(cmd, args)
		}
	}
}

func severalAtOnce(cmd *cobra.Command, args []string) bool {
	changed := func(name string) bool {
		f := cmd.Flags().Lookup(name)
		return f != nil && f.Changed && f.Value.String() != "" && f.Value.String() != "false"
	}
	if cmd.Name() == "apply" {
		return changed("prune")
	}
	if changed("selector") || changed("all") || changed("field-selector") {
		return true
	}
	switch {
	case len(args) == 0:
		return false
	case !strings.Contains(args[0], "/"):
		return strings.Contains(args[0], ",") || len(args) > 2
	}
	return len(args) > 1
}

func child(c *cobra.Command, name string) *cobra.Command {
	if c == nil {
		return nil
	}
	for _, sub := range c.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}

func walk(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, sub := range c.Commands() {
		walk(sub, fn)
	}
}

func stub(c *cobra.Command, run func(*cobra.Command, []string) error) {
	c.RemoveCommand(c.Commands()...)
	c.Hidden = true
	c.DisableFlagParsing = true
	c.Args = cobra.ArbitraryArgs
	c.ValidArgsFunction = nil
	c.PreRun, c.PreRunE, c.Run, c.PostRun, c.PostRunE = nil, nil, nil, nil, nil
	c.RunE = run
}

func refuser(r *run, reason string) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		r.refuse(reason)
		return errors.New(reason)
	}
}

func (t *Tree) config(cf *genericclioptions.ConfigFlags, ios genericiooptions.IOStreams, r *run) *cobra.Command {
	rules := &clientcmd.ClientConfigLoadingRules{Precedence: append([]string{}, t.paths...)}
	po := &clientcmd.PathOptions{GlobalFile: t.paths[0], ExplicitFileFlag: clientcmd.RecommendedConfigPathFlag, LoadingRules: rules}
	if len(t.paths) == 1 {
		rules.ExplicitPath = t.paths[0]
	} else {
		po.EnvVar = clientcmd.RecommendedConfigPathEnvVar
	}
	c := cmdconfig.NewCmdConfig(cf, po, ios)
	for _, name := range configWrites {
		if sub := child(c, name); sub != nil {
			stub(sub, refuser(r, ReasonKubeconfig))
		}
	}
	lockFlag(c.PersistentFlags().Lookup(clientcmd.RecommendedConfigPathFlag), ReasonConnection, r)
	if v := child(c, "view"); v != nil {
		c.RemoveCommand(v)
		v = cmdconfig.NewCmdConfigView(ios, redacted{po})
		for _, name := range []string{"raw", "flatten", "merge"} {
			lockFlag(v.Flags().Lookup(name), ReasonCredential, r)
		}
		c.AddCommand(v)
	}
	return c
}

type redacted struct {
	*clientcmd.PathOptions
}

const Redacted = "REDACTED"

func (r redacted) GetStartingConfig() (*clientcmdapi.Config, error) {
	cfg, err := r.PathOptions.GetStartingConfig()
	if err != nil {
		return nil, err
	}
	cfg = cfg.DeepCopy()
	for _, user := range cfg.AuthInfos {
		if user == nil {
			continue
		}
		if p := user.AuthProvider; p != nil {
			for k := range p.Config {
				p.Config[k] = Redacted
			}
		}
		if e := user.Exec; e != nil {
			for i := range e.Args {
				e.Args[i] = Redacted
			}
			for i := range e.Env {
				e.Env[i].Value = Redacted
			}
		}
		if user.Token != "" {
			user.Token = Redacted
		}
		if user.Password != "" {
			user.Password = Redacted
		}
		if len(user.ClientKeyData) > 0 {
			user.ClientKeyData = []byte(Redacted)
		}
	}
	return cfg, nil
}

func canI(c *cobra.Command, cf *genericclioptions.ConfigFlags, ios genericiooptions.IOStreams) {
	c.Run = func(cmd *cobra.Command, args []string) {
		f := cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(cf))
		o := &auth.CanIOptions{IOStreams: ios}
		o.AllNamespaces, _ = cmd.Flags().GetBool("all-namespaces")
		o.Quiet, _ = cmd.Flags().GetBool("quiet")
		o.Subresource, _ = cmd.Flags().GetString("subresource")
		o.List, _ = cmd.Flags().GetBool("list")
		o.NoHeaders, _ = cmd.Flags().GetBool("no-headers")
		cmdutil.CheckErr(o.Complete(f, args))
		cmdutil.CheckErr(o.Validate())
		if o.List {
			cmdutil.CheckErr(o.RunAccessList())
			return
		}
		allowed, err := o.RunAccessCheck()
		cmdutil.CheckErr(err)
		if !allowed {
			panic(fatalExit{code: 1})
		}
	}
}

func ClientVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range info.Deps {
			if dep.Path == "k8s.io/kubectl" && strings.HasPrefix(dep.Version, "v0.") {
				return "v1." + strings.TrimPrefix(dep.Version, "v0.")
			}
		}
	}
	return "v1.37.0"
}

type versions struct {
	ClientVersion *apimachineryversion.Info `json:"clientVersion,omitempty"`
	ServerVersion *apimachineryversion.Info `json:"serverVersion,omitempty"`
}

func versionRun(c *cobra.Command, cf *genericclioptions.ConfigFlags, ios genericiooptions.IOStreams) {
	c.Run = func(cmd *cobra.Command, args []string) {
		clientOnly, _ := cmd.Flags().GetBool("client")
		output, _ := cmd.Flags().GetString("output")
		if output != "" && output != "yaml" && output != "json" {
			cmdutil.CheckErr(errors.New(`--output must be 'yaml' or 'json'`))
		}
		v := versions{ClientVersion: &apimachineryversion.Info{GitVersion: ClientVersion(), Major: "1", Minor: strings.Split(strings.TrimPrefix(ClientVersion(), "v1."), ".")[0]}}
		var serverErr error
		if !clientOnly {
			if dc, err := cf.ToDiscoveryClient(); err == nil {
				dc.Invalidate()
				v.ServerVersion, serverErr = dc.ServerVersion()
			} else {
				serverErr = err
			}
		}
		switch output {
		case "":
			fmt.Fprintf(ios.Out, "Client Version: %s\n", v.ClientVersion.GitVersion)
			if v.ServerVersion != nil {
				fmt.Fprintf(ios.Out, "Server Version: %s\n", v.ServerVersion.GitVersion)
			}
		case "yaml":
			b, err := yaml.Marshal(&v)
			cmdutil.CheckErr(err)
			fmt.Fprintln(ios.Out, string(b))
		case "json":
			b, err := json.MarshalIndent(&v, "", "  ")
			cmdutil.CheckErr(err)
			fmt.Fprintln(ios.Out, string(b))
		}
		if v.ServerVersion != nil {
			if w := skew(v.ClientVersion.GitVersion, v.ServerVersion.GitVersion); w != "" {
				fmt.Fprintf(ios.ErrOut, "Warning: %s\n", w)
			}
		}
		cmdutil.CheckErr(serverErr)
	}
}

func skew(client, server string) string {
	c, err := version.ParseSemantic(client)
	if err != nil {
		return ""
	}
	s, err := version.ParseSemantic(server)
	if err != nil {
		return ""
	}
	if c.Major() != s.Major() || math.Abs(float64(c.Minor())-float64(s.Minor())) > 1 {
		return fmt.Sprintf("version difference between client (%d.%d) and server (%d.%d) exceeds the supported minor version skew of +/-1", c.Major(), c.Minor(), s.Major(), s.Minor())
	}
	return ""
}
