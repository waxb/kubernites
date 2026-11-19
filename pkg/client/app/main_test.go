package app

import (
	"os"
	"testing"

	"github.com/waxb/kubernites/pkg/client/gfx/gfxtest"
)

func TestMain(m *testing.M) {
	os.Exit(gfxtest.Main(m))
}
