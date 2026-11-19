package livescenes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAFixtureIsFoundInsideAModuleThatEmbedsThisOne(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "lib")
	rel := filepath.Join("pkg", "live", "testdata", "x.json")
	for path, body := range map[string]string{
		filepath.Join(root, "go.mod"):  "module example.com/game\n",
		filepath.Join(inner, "go.mod"): "module github.com/waxb/kubernites\n",
		filepath.Join(inner, rel):      "{}",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(filepath.Join(root))
	want, _ := filepath.EvalSymlinks(filepath.Join(inner, rel))
	got, _ := filepath.EvalSymlinks(FixturePath(rel))
	if got != want {
		t.Errorf("FixturePath(%s) = %s, want %s", rel, got, want)
	}
}
