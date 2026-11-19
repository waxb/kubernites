package app

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	AutosaveSlots = 3
	autoSuffix    = ".auto"
	ageSuffix     = ".age-"
	runningSuffix = ".running"
	CorruptSuffix = ".corrupt-"
)

type SaveSet struct{ Path string }

func (s SaveSet) Slot(i int) string {
	return fmt.Sprintf("%s%s%d", s.Path, autoSuffix, i%AutosaveSlots+1)
}

func (s SaveSet) Checkpoint(age string) string {
	return s.Path + ageSuffix + strings.ToLower(strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, age))
}

func (s SaveSet) Marker() string { return s.Path + runningSuffix }

type saveHead struct {
	SavedAt   time.Time `json:"savedAt"`
	RealmTime float64   `json:"realmTime"`
	Game      struct {
		Age string `json:"age"`
	} `json:"game"`
}

type Restored struct {
	From  string
	Note  string
	Crash bool
}

func (s SaveSet) Write(data []byte) error {
	if s.Path == "" {
		return errors.New("no save path configured")
	}
	return SaveAtomic(s.Path, data)
}

func (s SaveSet) WriteSlot(data []byte, slot int) error {
	if s.Path == "" {
		return errors.New("no save path configured")
	}
	return SaveAtomic(s.Slot(slot), data)
}

func (s SaveSet) WriteCheckpoint(age string, data []byte) error {
	if s.Path == "" || age == "" {
		return nil
	}
	return SaveAtomic(s.Checkpoint(age), data)
}

func (s SaveSet) MarkRunning(now time.Time) error {
	if s.Path == "" {
		return nil
	}
	return SaveAtomic(s.Marker(), []byte(now.UTC().Format(time.RFC3339)))
}

func (s SaveSet) MarkClosed() {
	if s.Path == "" {
		return
	}
	os.Remove(s.Marker())
}

func (s SaveSet) Reset() {
	if s.Path == "" {
		return
	}
	for _, path := range s.candidates() {
		os.Remove(path)
	}
}

func (s SaveSet) crashed() bool {
	if s.Path == "" {
		return false
	}
	_, err := os.Stat(s.Marker())
	return err == nil
}

func (s SaveSet) candidates() []string {
	dir, base := filepath.Dir(s.Path), filepath.Base(s.Path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type found struct {
		path string
		at   time.Time
	}
	var out []found
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, base+autoSuffix) && !strings.HasPrefix(name, base+ageSuffix) || strings.Contains(name, ".tmp-") {
			continue
		}
		if info, err := e.Info(); err == nil {
			out = append(out, found{filepath.Join(dir, name), info.ModTime()})
		}
	}
	slices.SortFunc(out, func(a, b found) int { return cmp.Or(b.at.Compare(a.at), strings.Compare(a.path, b.path)) })
	paths := make([]string, 0, len(out))
	for _, f := range out {
		paths = append(paths, f.path)
	}
	return paths
}

func describe(data []byte, now time.Time) string {
	var h saveHead
	if err := json.Unmarshal(data, &h); err != nil || h.SavedAt.IsZero() {
		return "an earlier save"
	}
	ago := now.Sub(h.SavedAt)
	when := AgoText(ago)
	if h.Game.Age == "" {
		return when
	}
	return fmt.Sprintf("%s, in the %s", when, view.Content.AgeTitle(h.Game.Age))
}

func AgoText(d time.Duration) string {
	s := int(d.Seconds())
	switch {
	case s < 0:
		return "just now"
	case s < 60:
		return fmt.Sprintf("%d s ago", s)
	case s < 3600:
		return fmt.Sprintf("%d min ago", s/60)
	case s < 86400:
		return fmt.Sprintf("%d h ago", s/3600)
	}
	return fmt.Sprintf("%d days ago", s/86400)
}

func (s SaveSet) Load(now time.Time, log *slog.Logger) (client.Realm, bool, Restored) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return s.fallback(now, log, "The latest save was missing", nil)
	}
	if err != nil {
		log.Warn("could not read save", "err", err)
		return s.fallback(now, log, "The latest save could not be read", nil)
	}
	r, err := campaignHook.LoadRealm(data, now)
	if err != nil {
		backup := s.Path + CorruptSuffix + now.Format("20060102-150405")
		if renameErr := os.Rename(s.Path, backup); renameErr != nil {
			log.Warn("could not keep a copy of the unreadable save", "err", renameErr)
		}
		log.Warn("save could not be loaded, kept a copy", "err", err, "backup", backup)
		return s.fallback(now, log, "The latest save could not be read", nil)
	}
	log.Info("realm restored", "save", s.Path)
	if s.crashed() {
		return r, true, Restored{From: s.Path, Crash: true, Note: "The last session did not close cleanly. The realm is restored to its last autosave, " + describe(data, now) + "."}
	}
	return r, true, Restored{}
}

func (s SaveSet) fallback(now time.Time, log *slog.Logger, why string, _ []byte) (client.Realm, bool, Restored) {
	for _, path := range s.candidates() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		r, err := campaignHook.LoadRealm(data, now)
		if err != nil {
			log.Warn("autosave could not be loaded", "err", err, "path", path)
			continue
		}
		log.Info("realm restored from an autosave", "path", path)
		return r, true, Restored{From: path, Note: why + ", so the realm is restored to the autosave from " + describe(data, now) + "."}
	}
	return nil, false, Restored{}
}
