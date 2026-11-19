package app

import (
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client"
)

type foundingCampaign struct {
	noCampaign
	seeds  []uint64
	epochs []time.Time
	fresh  int
}

func (c *foundingCampaign) FoundRealm(seed uint64, epoch time.Time) client.Realm {
	c.seeds, c.epochs = append(c.seeds, seed), append(c.epochs, epoch)
	return nil
}

func (c *foundingCampaign) NewRealm() client.Realm {
	c.fresh++
	return nil
}

func TestACaptureFoundsItsRealmFromTheCaptureSeedNotTheClock(t *testing.T) {
	c := &foundingCampaign{}
	SetCampaign(c)
	defer SetCampaign(nil)
	opts := Options{Capture: "shot.png", NewRealm: func() client.Realm {
		t.Fatal("a capture founded its realm from the clock")
		return nil
	}}
	opts.RealmOrNew()
	opts.RealmOrNew()
	if c.fresh != 0 || len(c.seeds) != 2 {
		t.Fatalf("a capture founded %d realms from the seed and %d from the clock", len(c.seeds), c.fresh)
	}
	for i := range c.seeds {
		if c.seeds[i] != CaptureSeed || !c.epochs[i].Equal(CaptureEpoch) {
			t.Fatalf("capture realm %d founded from seed %d at %v", i, c.seeds[i], c.epochs[i])
		}
	}
}

func TestAPlayedGameStillFoundsAFreshRealm(t *testing.T) {
	c := &foundingCampaign{}
	SetCampaign(c)
	defer SetCampaign(nil)
	Options{}.RealmOrNew()
	if c.fresh != 1 || len(c.seeds) != 0 {
		t.Fatalf("a played game founded %d fresh realms and %d seeded ones", c.fresh, len(c.seeds))
	}
}
