package world

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	loosePagesLife  = 20.0
	courierImage    = "courier:"
	archivistImage  = "archivist:"
	claimNamespace  = "default/"
	phaseBound      = "Bound"
	phaseReleased   = "Released"
	phaseTerminates = "Terminating"
)

type WardenMark struct {
	Keep  string
	State string
	At    Vec
}

type StorageLink struct {
	Vault   Vec
	Keep    Vec
	Hut     Vec
	Red     bool
	Mounted bool
	Sealed  bool
	Label   string
}

type DeedMark struct {
	Pod     string
	Claim   string
	Sealed  bool
	Shelves int
}

func keepsByNode(site view.Site) map[string]view.Structure {
	out := map[string]view.Structure{}
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep && s.Object.Name != "" {
			out[s.Object.Name] = s
		}
	}
	return out
}

func courierNodes(pods []model.PodView) map[string]bool {
	out := map[string]bool{}
	for _, p := range pods {
		if strings.HasPrefix(p.Image, courierImage) && p.Node != "" && !p.Terminating {
			out[p.Node] = true
		}
	}
	return out
}

func WardenPosts(sc Scene) []WardenMark {
	return wardenPosts(sc, func() map[string]bool { return courierNodes(sc.Pods) })
}

func wardenPosts(sc Scene, couriersOf func() map[string]bool) []WardenMark {
	if len(sc.Keeps) == 0 {
		return nil
	}
	keeps, couriers := keepsByNode(sc.Site), map[string]bool(nil)
	if anyWithoutDaemons(sc.Keeps) {
		couriers = couriersOf()
	}
	out := []WardenMark{}
	for _, k := range sc.Keeps {
		s, ok := keeps[k.Name]
		state := postState(k, couriers[k.Name])
		if !ok || state == "" {
			continue
		}
		lo, hi := Footprint(s)
		out = append(out, WardenMark{Keep: k.Name, State: state, At: Vec{float64(lo.X) - .5, float64(hi.Y) + .5}})
	}
	return out
}

func anyWithoutDaemons(keeps []view.KeepView) bool {
	for _, k := range keeps {
		if len(k.Daemons) == 0 {
			return true
		}
	}
	return false
}

func postState(k view.KeepView, couriers bool) string {
	if len(k.Daemons) == 0 {
		if couriers {
			return "absent"
		}
		return ""
	}
	for _, d := range k.Daemons {
		if d.Ready {
			return "running"
		}
	}
	return "starting"
}

func StrandedHuts(sc Scene) []string {
	stranded := map[string]bool{}
	for _, k := range sc.Keeps {
		if k.Stranded > 0 && len(k.Daemons) == 0 {
			stranded[k.Name] = true
		}
	}
	if len(stranded) == 0 {
		return nil
	}
	out := []string{}
	for _, p := range sc.Pods {
		if stranded[p.Node] && strings.HasPrefix(p.Image, courierImage) && !p.Terminating {
			out = append(out, p.Name)
		}
	}
	return out
}

func VaultLook(v view.VaultView, a *view.ArchiveView) string {
	switch v.Phase {
	case phaseReleased:
		return art.VaultName("released", 0)
	case phaseTerminates:
		return art.VaultName("terminating", vaultFill(a))
	case phaseBound:
		return art.VaultName("bound", vaultFill(a))
	}
	return art.VaultName("available", 0)
}

func vaultFill(a *view.ArchiveView) int {
	if a == nil {
		return 1
	}
	left := min(max(a.SettleIn/view.ArchiveSettleSeconds, 0), 1)
	return min(1+int(3*(1-left)), art.VaultShelfLevels-1)
}

func vaultStructureLook(s view.Structure, sc Scene) string {
	for _, v := range sc.Storage.Storage.Vaults {
		if v.Name == s.Object.Name {
			return VaultLook(v, sc.Storage.Archive)
		}
	}
	if s.Variant == "bound" {
		return art.VaultName("bound", 1)
	}
	return art.VaultName("available", 0)
}

func structureAnchor(s view.Structure) Vec {
	if s.Kind == view.StructVault {
		return FootprintCentre(s)
	}
	return Anchor(s)
}

func vaultStructure(site view.Site, name string) (view.Structure, bool) {
	for _, s := range site.Structures {
		if s.Kind == view.StructVault && s.Object.Name == name {
			return s, true
		}
	}
	return view.Structure{}, false
}

func deedNamed(sc Scene, name string) (view.DeedView, bool) {
	for _, d := range sc.Storage.Storage.Deeds {
		if d.Name == name {
			return d, true
		}
	}
	return view.DeedView{}, false
}

func vaultNamed(sc Scene, name string) (view.VaultView, bool) {
	for _, v := range sc.Storage.Storage.Vaults {
		if v.Name == name {
			return v, true
		}
	}
	return view.VaultView{}, false
}

func selectedClaim(sc Scene) string {
	for _, s := range sc.Site.Structures {
		if s.Kind == view.StructVault && s.ID == sc.Selected {
			if v, ok := vaultNamed(sc, s.Object.Name); ok {
				return strings.TrimPrefix(v.Claim, claimNamespace)
			}
		}
	}
	return ""
}

func linkedPod(sc Scene, p model.PodView, claim string) bool {
	if p.Terminating || len(p.Claims) == 0 {
		return false
	}
	if selectedPod(p.Name, sc.Selected) {
		return true
	}
	for _, c := range p.Claims {
		if c == claim && claim != "" {
			return true
		}
	}
	return false
}

func StorageLinks(sc Scene, homes map[string]Vec) []StorageLink {
	if sc.Selected == "" || len(sc.Storage.Storage.Deeds) == 0 {
		return nil
	}
	claim, keeps := selectedClaim(sc), keepsByNode(sc.Site)
	out := []StorageLink{}
	for _, p := range sc.Pods {
		if !linkedPod(sc, p, claim) {
			continue
		}
		for _, c := range p.Claims {
			if l, ok := linkOf(sc, keeps, homes, p, c); ok {
				out = append(out, l)
			}
		}
	}
	return out
}

func linkOf(sc Scene, keeps map[string]view.Structure, homes map[string]Vec, p model.PodView, claim string) (StorageLink, bool) {
	d, ok := deedNamed(sc, claim)
	if !ok {
		return StorageLink{}, false
	}
	v, _ := vaultNamed(sc, d.Volume)
	vs, ok := vaultStructure(sc.Site, d.Volume)
	home, housed := homes[p.Name]
	if !ok || !housed {
		return StorageLink{}, false
	}
	hut := Vec{home.X + .5, home.Y + .5}
	l := StorageLink{Vault: FootprintCentre(vs), Keep: hut, Hut: hut, Sealed: d.Phase == phaseBound, Red: true, Label: "PVC " + d.Name + " -> PV " + d.Volume}
	if k, ok := keeps[p.Node]; ok {
		l.Keep = FootprintCentre(k)
		isle, _ := sc.Site.IslandAt(k.X, k.Y)
		l.Red = isle.ID != v.Isle
		l.Mounted = !l.Red
	}
	return l, true
}

func DeedMarks(sc Scene) []DeedMark {
	if len(sc.Storage.Storage.Deeds) == 0 {
		return nil
	}
	claim := selectedClaim(sc)
	out := []DeedMark{}
	for _, p := range sc.Pods {
		if p.Terminating || len(p.Claims) == 0 || linkedPod(sc, p, claim) {
			continue
		}
		if d, ok := deedNamed(sc, p.Claims[0]); ok {
			out = append(out, DeedMark{Pod: p.Name, Claim: d.Name, Sealed: d.Phase == phaseBound, Shelves: d.Shelves})
		}
	}
	return out
}

func DetachWaits(sc Scene) map[string]string {
	if len(sc.Storage.Storage.Deeds) == 0 {
		return nil
	}
	old := map[string]model.PodView{}
	for _, p := range sc.Pods {
		if p.Terminating && p.Node != "" {
			for _, c := range p.Claims {
				old[c] = p
			}
		}
	}
	if len(old) == 0 {
		return nil
	}
	out := map[string]string{}
	for _, p := range sc.Pods {
		if p.Terminating || p.Ready || p.Node == "" {
			continue
		}
		for _, c := range p.Claims {
			o, held := old[c]
			if d, ok := deedNamed(sc, c); held && ok && o.Node != p.Node && d.Volume != "" {
				out[d.Volume] = o.Node
			}
		}
	}
	return out
}

func LoosePageHuts(sc Scene) []string {
	a := sc.Storage.Archive
	if a == nil || a.LostCause != view.LossEmptyDir || a.LostAt <= 0 {
		return nil
	}
	if age := sc.RealmTime - a.LostAt; age < 0 || age > loosePagesLife {
		return nil
	}
	out := []string{}
	for _, p := range sc.Pods {
		if strings.HasPrefix(p.Image, archivistImage) && !p.Terminating {
			out = append(out, p.Name)
		}
	}
	return out
}
