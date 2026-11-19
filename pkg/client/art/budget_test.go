package art

import "testing"

const atlasBudgetBytes = 52 << 20

func atlasBytes(s Set) (int, map[string]int) {
	total, per := 0, map[string]int{}
	for name, a := range atlases(s) {
		per[name] = len(a.Image.Pix)
		total += per[name]
	}
	return total, per
}

func TestAtlasesStayLean(t *testing.T) {
	total, per := atlasBytes(built())
	t.Logf("atlases hold %.1f MiB: %v", float64(total)/(1<<20), per)
	if total > atlasBudgetBytes {
		t.Fatalf("atlases hold %d bytes, budget %d", total, atlasBudgetBytes)
	}
}
