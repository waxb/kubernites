package write

import (
	"os"
	"path/filepath"
	"testing"
)

func TestATokenWarningRememberedBeforeTheSplitNeverPassesTheNoPreviewWarning(t *testing.T) {
	dir := t.TempDir()
	written := `{
  "contexts": [
    {
      "context": {
        "Name": "prod",
        "Server": "https://prod.example:6443",
        "User": "admin",
        "Identity": "id-prod"
      },
      "level": "WRITES",
      "warned": "WRITES",
      "classes": [
        "unpreviewed"
      ]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, MemoryFile), []byte(written), 0o600); err != nil {
		t.Fatal(err)
	}
	mem, err := LoadMemory(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin", Identity: "id-prod"}
	r, _, ok := mem.Recall(ctx)
	if !ok {
		t.Fatal("the remembered choice was not recalled")
	}
	g, err := New(r.Choice())
	if err != nil {
		t.Fatal(err)
	}
	if g.Passed(Unpreviewed) {
		t.Errorf("a choices file written while a token request asked the shared no-preview warning still passes that class for a card order an admission webhook cannot preview, whose own warning was never shown")
	}
}
