package art

import (
	"bytes"
	"fmt"
	"image"
	"reflect"
	"sort"
	"sync"
	"testing"
)

var (
	builtOnce sync.Once
	builtSet  Set
)

func built() Set {
	builtOnce.Do(func() { builtSet = Build() })
	return builtSet
}

func atlases(s Set) map[string]Atlas {
	return map[string]Atlas{
		"terrain": s.Terrain, "buildings": s.Buildings, "walkers": s.Walkers, "effects": s.Effects,
		"icons": s.Icons, "portraits": s.Portraits, "ui": s.UI,
	}
}

func expand(prefix string, parts ...[]string) []string {
	out := []string{prefix}
	for _, p := range parts {
		var next []string
		for _, head := range out {
			for _, tail := range p {
				next = append(next, head+"/"+tail)
			}
		}
		out = next
	}
	return out
}

func seq(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprint(i)
	}
	return out
}

func list(names ...string) []string { return names }

var contractRoles = list("lamplighter", "baker", "scribe", "granary-keeper", "archivist", "courier", "tallyman",
	"oracle", "chancellor", "codex-keeper", "quartermaster", "steward", "herald", "wayfinder", "augur", "reeve",
	"pilgrim", "builder", "imp", "wanderer", "sheep", "helmbearer", "clerk", "raven", "hound")

var contractCritters = list("deer", "stag", "rabbit", "fox", "boar", "finch", "gull")

var contractKinds = list("keep-tower", "keep-stone", "keep-great", "keep-citadel", "codex", "chancery", "quartermaster", "stewards-hall",
	"wayfinder", "observatory", "gate", "treasury", "warhorn")

func stallContract() []string {
	out := []string{}
	for _, k := range StallKinds {
		for _, f := range StallFaces {
			out = append(out, StallArtKind(k, f))
		}
	}
	return out
}

func contractNames() map[string][]string {
	terrain := expand("sea", seq(SeaFrames))
	terrain = append(terrain, expand("blot", seq(Blots))...)
	terrain = append(terrain, expand("decor", list("tree-oak", "tree-pine", "tree-birch", "bush", "rock", "reeds",
		"flowers", "stump", "log", "haystack", "fence", "tree-maple", "tree-spruce", "tree-old", "tree-willow", "fern",
		"boulder", "mushrooms", "tallgrass", "lily", "ley-stone"), seq(DecorVariants))...)
	buildings := expand("building", append(append(list(), contractKinds...), stallContract()...), list("ready", "construction-0", "construction-1", "construction-2", "dark"))
	buildings = append(buildings, expand("hearth", list("1", "2", "3"), list("ready", "kindling", "guttering", "ghost", "banished", "pending"))...)
	buildings = append(buildings, expand("haven", list("pier-x", "pier-y", "quay", "crates"), seq(3))...)
	buildings = append(buildings, expand("haven", list("warehouse"), seq(2))...)
	buildings = append(buildings, expand("haven", list("bollard", "crane", "lighthouse", "sign"), seq(1))...)
	effects := expand("fx", list("fire", "smoke", "dust", "void", "sparkle", "coin", "raven", "explosion", "heal"), seq(6))
	effects = append(effects, expand("fx", list("boat"), seq(4))...)
	effects = append(effects, expand("fx", list("banner"), seq(3))...)
	effects = append(effects, expand("fx", list("reweave"), seq(6))...)
	effects = append(effects, expand("fx", list("horn"), seq(4))...)
	effects = append(effects, expand("fx", list("fish", "ley"), seq(6))...)
	effects = append(effects, expand("fx", list("charter"), seq(4))...)
	effects = append(effects, expand("fx", list("wake", "request"), seq(6))...)
	effects = append(effects, expand("ship", list("cog", "caravel", "galleon", "carrack"), seq(8), list("ne", "nw", "se", "sw"), seq(4))...)
	icons := expand("icon", list("gold", "wisdom", "renown", "pod", "node", "service", "build", "summon", "apply",
		"scale-up", "scale-down", "probe", "roll", "tech", "codex", "chronicle", "soulstone", "crier", "age", "speed",
		"pause", "alert", "describe", "logs", "yaml", "back", "close", "academy", "diploma", "thread", "loom-treasury",
		"loom-builders", "loom-fame", "siege", "archive", "warhorn"))
	return map[string][]string{
		"terrain":   terrain,
		"buildings": buildings,
		"walkers":   expand("walker", append(append(list(), contractRoles...), contractCritters...), list("idle", "walk", "work", "carry"), list("se", "sw"), seq(4)),
		"effects":   effects,
		"icons":     append(icons, "appicon"),
		"portraits": expand("portrait", append(append(append(list("academy", "loom"), contractRoles...), contractKinds...), StallKinds...)),
		"ui": expand("ui", list("panel", "panel-dark", "button", "button-hover", "button-pressed", "button-disabled",
			"tab", "tab-active", "scroll", "tooltip", "bar-frame", "bar-fill-good", "bar-fill-bad", "minimap-frame",
			"woven-strip", "chip", "bar-mini")),
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	a, b := atlases(built()), atlases(Build())
	for name, x := range a {
		y := b[name]
		if !bytes.Equal(x.Image.Pix, y.Image.Pix) || x.Image.Rect != y.Image.Rect {
			t.Errorf("%s: pixels differ between builds", name)
		}
		if !reflect.DeepEqual(x.Regions, y.Regions) {
			t.Errorf("%s: regions differ between builds", name)
		}
	}
}

func TestContractNamesExist(t *testing.T) {
	all := atlases(built())
	for atlas, names := range contractNames() {
		for _, n := range names {
			if _, ok := all[atlas].Regions[n]; !ok {
				t.Errorf("%s: missing %s", atlas, n)
			}
		}
	}
}

func TestNamesSortedAndComplete(t *testing.T) {
	for name, a := range atlases(built()) {
		names := Names(a)
		if len(names) != len(a.Regions) || !sort.StringsAreSorted(names) {
			t.Errorf("%s: Names returned %d unsorted or incomplete entries", name, len(names))
		}
	}
}

func TestRegionsInsideAtlasAndAnchorsInsideRegions(t *testing.T) {
	for name, a := range atlases(built()) {
		b := a.Image.Bounds()
		for n, r := range a.Regions {
			if r.W <= 0 || r.H <= 0 || !image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H).In(b) {
				t.Errorf("%s: %s region %+v outside %v", name, n, r, b)
			}
			if r.AnchorX < 0 || r.AnchorX >= r.W || r.AnchorY < 0 || r.AnchorY >= r.H {
				t.Errorf("%s: %s anchor outside region %+v", name, n, r)
			}
		}
	}
}

func TestRegionsDoNotOverlap(t *testing.T) {
	for name, a := range atlases(built()) {
		rects := make([]image.Rectangle, 0, len(a.Regions))
		for _, n := range Names(a) {
			r := a.Regions[n]
			rects = append(rects, image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H))
		}
		for i := range rects {
			for j := i + 1; j < len(rects); j++ {
				if rects[i].Overlaps(rects[j]) {
					t.Fatalf("%s: regions %v and %v overlap", name, rects[i], rects[j])
				}
			}
		}
	}
}

func BenchmarkBuild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Build()
	}
}
