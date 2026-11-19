package app

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/waxb/kubernites/pkg/client/hud"
)

type marshaler interface {
	Marshal(savedAt time.Time) ([]byte, error)
}

func SaveAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create save directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary save: %w", err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return fmt.Errorf("write temporary save: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(name)
		return fmt.Errorf("sync temporary save: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return fmt.Errorf("close temporary save: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return fmt.Errorf("replace save: %w", err)
	}
	return nil
}

func SaveRealm(path string, r marshaler, now time.Time) error {
	if path == "" {
		return errors.New("no save path configured")
	}
	data, err := r.Marshal(now)
	if err != nil {
		return fmt.Errorf("marshal realm: %w", err)
	}
	return SaveAtomic(path, data)
}

func BackupSave(path string, log *slog.Logger) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err != nil {
		return
	}
	backup := path + ".previous"
	if err := os.Rename(path, backup); err != nil {
		log.Warn("could not keep the previous realm", "err", err)
		return
	}
	log.Info("kept the previous realm", "backup", backup)
}

func (g *Game) saveSet() SaveSet { return SaveSet{Path: g.Opts.SavePath} }

func (g *Game) Save() error {
	s := g.realmSession()
	if g.Capture != nil || s == nil || s.Live || g.Opts.SavePath == "" {
		return nil
	}
	data, err := s.Realm.Marshal(time.Now())
	if err == nil {
		err = g.saveSet().Write(data)
	}
	if err != nil {
		g.Log.Error("autosave failed", "err", err, "save", g.Opts.SavePath)
		s.SaveErr = err.Error()
		g.Sess.Hud.Toast("Could not save the realm: "+err.Error(), true)
		return err
	}
	if err := g.saveSet().WriteSlot(data, g.slot); err != nil {
		g.Log.Warn("could not write an autosave slot", "err", err, "slot", g.slot)
	}
	g.slot = (g.slot + 1) % AutosaveSlots
	s.Saved, s.SavedAt, s.SaveErr = true, s.Clock, ""
	g.checkpoint(s.State.Age, data)
	g.Log.Debug("realm saved", "save", g.Opts.SavePath)
	return nil
}

func (g *Game) checkpoint(age string, data []byte) {
	if age == "" || age == g.checkedAge {
		return
	}
	if err := g.saveSet().WriteCheckpoint(age, data); err != nil {
		g.Log.Warn("could not write the age checkpoint", "err", err, "age", age)
		return
	}
	g.checkedAge = age
}

func (g *Game) AutosaveDue(in hud.Input, dt float64) bool {
	g.SaveTick += dt
	if g.SaveTick < SaveInterval {
		return false
	}
	if in.Held || g.Pointer.Dragging {
		return false
	}
	g.SaveTick = 0
	return true
}

func (g *Game) markRunning() {
	if g.Capture != nil || g.Opts.SavePath == "" {
		return
	}
	if err := g.saveSet().MarkRunning(time.Now()); err != nil {
		g.Log.Warn("could not mark the session as running", "err", err)
	}
}

func (g *Game) MarkClosed() {
	if g.Capture != nil {
		return
	}
	g.saveSet().MarkClosed()
}

func (g *Game) opened(s *Session) {
	g.markRunning()
	g.checkedAge = ""
	if _, err := os.Stat(g.saveSet().Checkpoint(s.State.Age)); err == nil {
		g.checkedAge = s.State.Age
	}
	if note := g.Opts.RestoredNote; note != "" {
		g.Opts.RestoredNote = ""
		s.Hud.Toast(note, false)
		s.Hud.BannerCause("RESTORED", note)
	}
}
