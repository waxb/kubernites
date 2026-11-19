package soulstone

import "testing"

func TestThePostWardenIsASystemImageNobodyBuys(t *testing.T) {
	p, ok := Lookup("warden:1.0")
	if !ok || !p.System || p.Sprite != "warden" || p.Repository != "warden" || p.Tag != "1.0" || len(p.Kinds) != 0 {
		t.Fatalf("warden = %+v, %v", p, ok)
	}
}
