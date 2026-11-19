package write

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

const MemoryFile = "live-choices.json"

func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func (l *Level) UnmarshalText(data []byte) error {
	v, ok := ParseLevel(string(data))
	if !ok {
		return fmt.Errorf("unknown level %q", data)
	}
	*l = v
	return nil
}

type Remembered struct {
	Context Context `json:"context"`
	Level   Level   `json:"level"`
	Warned  Level   `json:"warned"`
	Classes []Class `json:"classes,omitempty"`
}

func (r Remembered) Choice() Choice {
	return Choice{Context: r.Context, Level: r.Level, Classes: slices.Clone(r.Classes)}
}

type Memory struct {
	Contexts []Remembered `json:"contexts"`
}

type Changed struct {
	Was    Context
	Server bool
	User   bool
}

func (c Changed) Any() bool {
	return c.Server || c.User
}

func MemoryPath(dir string) string {
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, MemoryFile)
}

func LoadMemory(dir string) (Memory, error) {
	path := MemoryPath(dir)
	if path == "" {
		return Memory{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Memory{}, nil
	}
	if err != nil {
		return Memory{}, err
	}
	var m Memory
	if err := json.Unmarshal(data, &m); err != nil {
		return Memory{}, fmt.Errorf("read %s: %w", MemoryFile, err)
	}
	out := Memory{}
	for _, r := range m.Contexts {
		if r.Context.Name == "" || (r.Level != Writes && r.Level != Seamless) {
			continue
		}
		var classes []Class
		for _, c := range r.Classes {
			if KnownClass(c) && !slices.Contains(classes, c) {
				classes = append(classes, c)
			}
		}
		r.Classes = classes
		if r.Warned < r.Level {
			r.Warned = r.Level
		}
		out.Keep(r)
	}
	return out, nil
}

func SaveMemory(dir string, m Memory) error {
	path := MemoryPath(dir)
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (m Memory) Recall(c Context) (Remembered, Changed, bool) {
	for _, r := range m.Contexts {
		if r.Context.Name != c.Name {
			continue
		}
		ch := Changed{Was: r.Context, Server: r.Context.Server != c.Server, User: r.Context.User != c.User}
		if ch.Any() {
			return Remembered{}, ch, false
		}
		return r, Changed{}, true
	}
	return Remembered{}, Changed{}, false
}

func (m *Memory) Keep(r Remembered) {
	m.Forget(r.Context.Name)
	slices.Sort(r.Classes)
	m.Contexts = append(m.Contexts, r)
	sort.Slice(m.Contexts, func(i, j int) bool { return m.Contexts[i].Context.Name < m.Contexts[j].Context.Name })
}

func (m *Memory) Forget(name string) {
	m.Contexts = slices.DeleteFunc(m.Contexts, func(r Remembered) bool { return r.Context.Name == name })
}

func (m *Memory) ForgetAll() {
	m.Contexts = nil
}
