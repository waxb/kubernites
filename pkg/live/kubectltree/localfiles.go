package kubectltree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	webFiles         = "the Stone reads manifests from files: a web address would be fetched outside Kubernites' gate"
	noStdin          = "the Stone has no standard input: write the manifest in the Scriptorium"
	ReasonKubeReads  = "the Stone never reads ~/.kube, the kubeconfig chosen at connect or a token file or client key it names, through a link or not"
	maxWalk          = 10000
	globCharacters   = `*?[\`
	keyedFileSources = "from-file"
)

var fileFlags = []string{"filename", "from-file", "from-env-file", "patch-file", "custom", "cert", "key"}

var templateFiles = map[string]bool{"go-template-file": true, "templatefile": true, "jsonpath-file": true, "custom-columns-file": true}

type localRead struct {
	flag string
	path string
}

func localReads(cmd *cobra.Command) []localRead {
	var out []localRead
	flags := cmd.Flags()
	for _, name := range fileFlags {
		f := flags.Lookup(name)
		if f == nil || !f.Changed {
			continue
		}
		for _, v := range flagValues(f) {
			if name == keyedFileSources {
				if _, p, ok := strings.Cut(v, "="); ok {
					v = p
				}
			}
			out = append(out, localRead{flag: name, path: v})
		}
	}
	if p, ok := templateFile(flags); ok {
		out = append(out, localRead{flag: "template", path: p})
	}
	return out
}

func flagValues(f *pflag.Flag) []string {
	if s, ok := f.Value.(pflag.SliceValue); ok {
		return s.GetSlice()
	}
	return []string{f.Value.String()}
}

func templateFile(flags *pflag.FlagSet) (string, bool) {
	o := flags.Lookup("output")
	if o == nil {
		return "", false
	}
	format, inline, _ := strings.Cut(o.Value.String(), "=")
	if !templateFiles[format] {
		return "", false
	}
	if t := flags.Lookup("template"); t != nil && t.Value.String() != "" {
		return t.Value.String(), true
	}
	return inline, true
}

func (t *Tree) checkReads(cmd *cobra.Command, stdin bool) error {
	for _, read := range localReads(cmd) {
		p := read.path
		lower := strings.ToLower(strings.TrimSpace(p))
		switch {
		case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
			return errors.New(webFiles)
		case p == "-" && read.flag == "filename":
			if !stdin {
				return errors.New(noStdin)
			}
			continue
		case p == "":
			continue
		}
		matches := []string{p}
		if read.flag == "filename" {
			globbed, err := expandAsKubectl(p)
			if err != nil {
				return fmt.Errorf("the Stone cannot read the pattern %s: %v", shown(p), err)
			}
			matches = append(matches, globbed...)
		} else if _, err := os.Lstat(p); err != nil && strings.ContainsAny(p, globCharacters) {
			globbed, err := filepath.Glob(p)
			if err != nil {
				return fmt.Errorf("the Stone cannot read the pattern %s: %v", shown(p), err)
			}
			matches = append(matches, globbed...)
		}
		for _, m := range matches {
			if err := t.plainFiles(m); err != nil {
				return err
			}
		}
	}
	return nil
}

func expandAsKubectl(pattern string) ([]string, error) {
	if _, err := os.Stat(pattern); os.IsNotExist(err) {
		return filepath.Glob(pattern)
	}
	return []string{pattern}, nil
}

func (t *Tree) plainFiles(path string) error {
	if t.protectedRead(path) {
		return fmt.Errorf("%s: %s", ReasonKubeReads, shown(path))
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}
	refuse := func(p string) error {
		return fmt.Errorf("the Stone reads only regular files and directories: %s is not one, and reading it could wait forever", shown(p))
	}
	if st.Mode().IsRegular() {
		return nil
	}
	if !st.IsDir() {
		return refuse(path)
	}
	seen := 0
	return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || p == path {
			return nil
		}
		if seen++; seen > maxWalk {
			return fmt.Errorf("the Stone reads at most %d entries of a directory: %s holds more", maxWalk, shown(path))
		}
		if t.protectedRead(p) {
			return fmt.Errorf("%s: %s is inside %s", ReasonKubeReads, shown(p), shown(path))
		}
		st, err := os.Stat(p)
		if err != nil {
			return nil
		}
		if !st.Mode().IsRegular() && !st.IsDir() {
			return refuse(p)
		}
		return nil
	})
}

func (t *Tree) protectedRead(p string) bool {
	abs, ok := rawAbs(p)
	if !ok {
		return true
	}
	return t.protected(abs)
}
