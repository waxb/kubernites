package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const archivistPrefix = "archivist:"

type bookWalk struct {
	pod   string
	vault string
}

type storageBeats struct {
	primed bool
	pods   map[string]bool
	vaults map[string]bool
}

func boundVault(deeds []view.DeedView, claim string) string {
	for _, d := range deeds {
		if d.Name == claim && d.Phase == "Bound" {
			return d.Volume
		}
	}
	return ""
}

func ArchivistReturns(before map[string]bool, pods []model.PodView, deeds []view.DeedView) []bookWalk {
	out := []bookWalk{}
	for _, p := range pods {
		if before[p.Name] || !strings.HasPrefix(p.Image, archivistPrefix) || len(p.Claims) == 0 {
			continue
		}
		if vault := boundVault(deeds, p.Claims[0]); vault != "" {
			out = append(out, bookWalk{pod: p.Name, vault: vault})
		}
	}
	return out
}

func GuildDug(before map[string]bool, vaults []view.VaultView) []string {
	out := []string{}
	for _, v := range vaults {
		if v.Dynamic && !before[v.Name] {
			out = append(out, v.Name)
		}
	}
	return out
}

func archivists(pods []model.PodView) map[string]bool {
	out := map[string]bool{}
	for _, p := range pods {
		if strings.HasPrefix(p.Image, archivistPrefix) {
			out[p.Name] = true
		}
	}
	return out
}

func vaultNames(vaults []view.VaultView) map[string]bool {
	out := map[string]bool{}
	for _, v := range vaults {
		out[v.Name] = true
	}
	return out
}

func (s *Session) storageNotices() {
	b := s.beats
	s.beats = storageBeats{primed: true, pods: archivists(s.Pods), vaults: vaultNames(s.State.Storage.Vaults)}
	if !b.primed {
		return
	}
	for _, w := range ArchivistReturns(b.pods, s.Pods, s.State.Storage.Deeds) {
		home, okHome := s.Renderer.Home(w.pod)
		vault, okVault := vaultSpot(s.Site, w.vault)
		if okHome && okVault {
			s.Town.send(world.Errand{Kind: world.ErrandBooks, Role: "archivist", From: home, To: vault}, s.Clocks.World)
		}
	}
	for _, name := range GuildDug(b.vaults, s.State.Storage.Vaults) {
		if at, ok := vaultSpot(s.Site, name); ok {
			s.Juice("puff", at, "")
			s.Juice("sparkle", at, "")
		}
	}
}
