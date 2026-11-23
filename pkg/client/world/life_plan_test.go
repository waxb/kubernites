package world

import (
	"reflect"
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func roadlessSite() view.Site {
	site := lifeSite()
	site.Structures = slices.DeleteFunc(slices.Clone(site.Structures), func(s view.Structure) bool { return s.Kind == view.StructTreasury })
	return site
}

func TestLifeIsNotPlannedAgainOnAnUnchangedSiteWithoutACartRoad(t *testing.T) {
	r := NewRenderer(nil)
	sc := Scene{Site: roadlessSite()}
	l := r.syncLife(sc)
	if l.road != nil || len(l.stalls) == 0 {
		t.Fatalf("the site has no cart road but stalls: road %v stalls %v", l.road, l.stalls)
	}
	first := &l.stalls[0]
	for range 3 {
		sc.Site = roadlessSite()
		if l = r.syncLife(sc); &l.stalls[0] != first {
			t.Fatal("the unchanged site was planned again")
		}
	}
}

func TestLifeIsPlannedAgainWhenTheSiteChangesWithoutACartRoad(t *testing.T) {
	r := NewRenderer(nil)
	sc := Scene{Site: roadlessSite()}
	l := r.syncLife(sc)
	spot := l.stalls[0]
	site := roadlessSite()
	site.Tiles = slices.Clone(site.Tiles)
	site.Tiles[int(spot.Y)*site.Width+int(spot.X)] = view.TileGrass
	sc.Site = site
	l = r.syncLife(sc)
	if want := StallSpots(site, maxStalls); !slices.Equal(l.stalls, want) || slices.Contains(l.stalls, spot) {
		t.Fatalf("stalls %v after the plaza tile at %v turned to grass, want %v", l.stalls, spot, want)
	}
}

func TestEveryFieldOfTheSiteCountsAsAChangeForTheLifePlan(t *testing.T) {
	base := roadlessSite()
	base.Islands = []view.Island{{ID: "home"}}
	base.Havens = []view.Haven{{ID: "h"}}
	base.Spots = []view.Spot{{ID: "s"}}
	base.Plots = []view.Point{{X: 1}}
	base.Region = make([]uint8, len(base.Tiles))
	base.Power = make([]uint8, len(base.Tiles))
	base.Wear = make([]uint16, len(base.Tiles))
	base.Forum = &view.Forum{Rings: 1}
	var kept plannedSite
	kept.keep(base)
	if !kept.same(base) {
		t.Fatal("the kept site differs from itself")
	}
	typ := reflect.TypeOf(base)
	for i := range typ.NumField() {
		f := typ.Field(i)
		if f.Name == "Roads" || f.Name == "Land" {
			continue
		}
		changed := cloneSiteForTest(base)
		if !bump(reflect.ValueOf(&changed).Elem().Field(i)) {
			t.Fatalf("the test cannot change Site.%s (%s)", f.Name, f.Type)
		}
		if kept.same(changed) {
			t.Errorf("a change of Site.%s is not seen", f.Name)
		}
	}
	for i := range reflect.TypeOf(view.Structure{}).NumField() {
		changed := cloneSiteForTest(base)
		bump(reflect.ValueOf(&changed.Structures[0]).Elem().Field(i))
		if kept.same(changed) {
			t.Errorf("a change of Structure.%s is not seen", reflect.TypeOf(view.Structure{}).Field(i).Name)
		}
	}
}

func cloneSiteForTest(s view.Site) view.Site {
	var k plannedSite
	k.keep(s)
	return k.site
}

func bump(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(v.Int() + 1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(v.Uint() + 1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(v.Float() + 1)
	case reflect.Bool:
		v.SetBool(!v.Bool())
	case reflect.String:
		v.SetString(v.String() + "x")
	case reflect.Struct:
		return v.NumField() > 0 && bump(v.Field(0))
	case reflect.Pointer:
		if v.IsNil() {
			return false
		}
		c := reflect.New(v.Elem().Type())
		c.Elem().Set(v.Elem())
		v.Set(c)
		return bump(c.Elem())
	case reflect.Slice:
		if v.Len() == 0 {
			return false
		}
		c := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		reflect.Copy(c, v)
		v.Set(c)
		return bump(c.Index(v.Len() - 1))
	default:
		return false
	}
	return true
}
