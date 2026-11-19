package soulstone

import "testing"

func TestTheArchivistIsASystemClass(t *testing.T) {
	p, ok := Lookup(ArchivistImage)
	if !ok || !p.System || p.Profession != "Archivist" || len(p.Kinds) != 0 {
		t.Fatalf("archivist = %+v, %v", p, ok)
	}
	for _, played := range Playable() {
		if played.Image == ArchivistImage {
			t.Fatal("the archivist is not a trade on the registry row")
		}
	}
}
