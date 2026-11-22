package kubectltree

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live/write"
)

var protectedWalk = 50000

type guarded struct {
	mu    sync.Mutex
	gen   int
	at    int
	paths []string
	files []os.FileInfo
}

func resolvedPath(p string) string {
	if r, ok := physicalPath(p); ok {
		return r
	}
	return filepath.Clean(p)
}

const maxLinks = 255

func components(p string) []string {
	var out []string
	for _, c := range strings.FieldsFunc(p, func(r rune) bool { return r < 0x80 && os.IsPathSeparator(uint8(r)) }) {
		if c != "." {
			out = append(out, c)
		}
	}
	return out
}

func physicalPath(raw string) (string, bool) {
	vol := filepath.VolumeName(raw)
	todo := components(raw[len(vol):])
	cur := vol + string(filepath.Separator)
	links := 0
	for len(todo) > 0 {
		name := todo[0]
		todo = todo[1:]
		if name == ".." {
			cur = filepath.Dir(cur)
			continue
		}
		next := filepath.Join(cur, name)
		st, err := os.Lstat(next)
		if err != nil {
			return filepath.Join(append([]string{next}, todo...)...), true
		}
		if st.Mode()&os.ModeSymlink == 0 {
			cur = next
			continue
		}
		if links++; links > maxLinks {
			return "", false
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", false
		}
		if v := filepath.VolumeName(target); filepath.IsAbs(target) {
			cur, target = v+string(filepath.Separator), target[len(v):]
		}
		todo = append(components(target), todo...)
	}
	return cur, true
}

func rawAbs(p string) (string, bool) {
	if filepath.IsAbs(p) {
		return p, true
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	return wd + string(filepath.Separator) + p, true
}

func (t *Tree) freshGuard() {
	t.guard.mu.Lock()
	t.guard.gen++
	t.guard.mu.Unlock()
}

func (t *Tree) guardedFiles() ([]string, []os.FileInfo) {
	t.guard.mu.Lock()
	defer t.guard.mu.Unlock()
	if t.guard.paths != nil && t.guard.at == t.guard.gen {
		return t.guard.paths, t.guard.files
	}
	var paths []string
	add := func(p string) {
		if p == "" {
			return
		}
		if abs, err := filepath.Abs(p); err == nil {
			paths = append(paths, filepath.Clean(abs))
		}
	}
	for _, k := range t.paths {
		add(k)
	}
	if cfg, err := (&clientcmd.ClientConfigLoadingRules{Precedence: t.paths}).Load(); err == nil {
		for _, u := range cfg.AuthInfos {
			if u != nil {
				add(u.TokenFile)
				add(u.ClientKey)
			}
		}
	}
	var files []os.FileInfo
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil {
			files = append(files, st)
		}
	}
	files = append(files, walkDotKube(filepath.Join(t.local, ".kube"), protectedWalk)...)
	t.guard.paths, t.guard.files, t.guard.at = paths, files, t.guard.gen
	return paths, files
}

var kubectlCaches = []string{"cache", "http-cache"}

func walkDotKube(kube string, bound int) []os.FileInfo {
	root, ok := physicalPath(kube)
	if !ok {
		return nil
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil
	}
	files := []os.FileInfo{st}
	visited := map[string]bool{root: true}
	first := func(dir string) bool {
		real, ok := physicalPath(dir)
		if !ok || visited[real] {
			return false
		}
		visited[real] = true
		return true
	}
	seen := 0
	var later []string
	walk := func(queue []string) bool {
		for len(queue) > 0 {
			dir := queue[0]
			queue = queue[1:]
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				p := filepath.Join(dir, e.Name())
				if dir == root && slices.Contains(kubectlCaches, e.Name()) {
					later = append(later, p)
					continue
				}
				if seen++; seen > bound {
					return false
				}
				st, err := os.Stat(p)
				if err != nil {
					continue
				}
				files = append(files, st)
				if st.IsDir() && first(p) {
					queue = append(queue, p)
				}
			}
		}
		return true
	}
	if walk([]string{root}) {
		for _, c := range later {
			if seen++; seen > bound {
				break
			}
			st, err := os.Stat(c)
			if err != nil {
				continue
			}
			files = append(files, st)
			if !st.IsDir() || !first(c) {
				continue
			}
			if !walk([]string{c}) {
				break
			}
		}
	}
	return files
}

func (t *Tree) protected(p string) bool {
	physical, ok := physicalPath(p)
	if !ok {
		return true
	}
	cleaned := filepath.Clean(p)
	return t.protectedAs(cleaned, cleaned) || t.protectedAs(physical, p)
}

func (t *Tree) protectedAs(p, opened string) bool {
	kube := filepath.Join(t.local, ".kube")
	paths, files := t.guardedFiles()
	if folded(kube, p) || folded(resolvedPath(kube), p) {
		return true
	}
	for _, g := range paths {
		if fold(g) == fold(p) || fold(resolvedPath(g)) == fold(p) {
			return true
		}
	}
	same := func(st os.FileInfo) bool {
		for _, f := range files {
			if os.SameFile(st, f) {
				return true
			}
		}
		return false
	}
	if st, err := os.Stat(opened); err == nil && same(st) {
		return true
	}
	for cur := p; ; cur = filepath.Dir(cur) {
		if st, err := os.Stat(cur); err == nil && same(st) {
			return true
		}
		if filepath.Dir(cur) == cur {
			return false
		}
	}
}

func (t *Tree) serverUserinfos() []string {
	var out []string
	add := func(server string) {
		if u := write.ServerUserinfo(server); u != "" && !slices.Contains(out, u) {
			out = append(out, u)
		}
	}
	add(t.server)
	if cfg, err := (&clientcmd.ClientConfigLoadingRules{Precedence: t.paths}).Load(); err == nil {
		for _, c := range cfg.Clusters {
			if c != nil {
				add(c.Server)
			}
		}
	}
	slices.SortFunc(out, func(a, b string) int { return len(b) - len(a) })
	return out
}
