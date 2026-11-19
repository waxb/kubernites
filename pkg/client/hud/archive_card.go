package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	archiveCardID  = "muniment"
	archiveRaiseID = "muniment-raise"
	sealwrightOne  = "sealwright:1.0"
	verbPrefix     = "verb:"
	verbDig        = "dig"
	verbDeed       = "deed"
	verbStateful   = "stateful"
	verbGuild      = "guild"
	booksDeed      = "books"
	booksShelves   = 2
	archivistImage = "archivist:"
	verbTextPad    = 10
	verbGap        = 4
)

type storageVerb struct {
	key   string
	label string
	tip   string
	act   view.Action
}

func pageKnown(f Frame, id string) bool {
	for _, p := range f.State.Grimoire {
		if p.ID == id {
			return p.Rank > 0 || p.State == "unlocked"
		}
	}
	return false
}

func sealwrightsCame(f Frame) bool {
	for _, s := range f.State.Soulstones {
		if s.Image == sealwrightOne && s.Owned {
			return true
		}
	}
	return false
}

func archiveIsStateful(f Frame) bool {
	_, ok := statefulSetNamed(f, view.ArchiveName)
	return ok
}

func archiveDeed(f Frame) string {
	for _, p := range f.Pods {
		if strings.HasPrefix(p.Image, archivistImage) && len(p.Claims) > 0 {
			return p.Claims[0]
		}
	}
	return ""
}

func guildStands(f Frame) bool {
	for _, c := range f.State.Storage.Classes {
		if c.Name == view.GuildClass {
			return true
		}
	}
	return false
}

func freeVault(f Frame) bool {
	for _, v := range f.State.Storage.Vaults {
		if v.Phase == "Available" {
			return true
		}
	}
	return false
}

func archiveObject(f Frame) string {
	if archiveIsStateful(f) {
		return "statefulset"
	}
	return "deployment"
}

func archiveKind(f Frame) string {
	if archiveIsStateful(f) {
		return "StatefulSet"
	}
	return "Deployment"
}

func archiveCard(f Frame) Card {
	a := *f.State.Archive
	c := Card{Title: ui.Bridge("THE MUNIMENT ROOM", "ARCHIVE"), K8s: objectRef(archiveKind(f), archiveObject(f)+".apps", view.ArchiveName), progress: -1}
	c.Lines = []Line{
		{Text: ui.Bridge(fmt.Sprintf("UNSETTLED %s GOLD IN %s SEALS", Compact(a.Unsettled), Compact(a.Seals)), fmt.Sprintf("unsettled $%s in %s TLS requests", Compact(a.Unsettled), Compact(a.Seals))), Clr: ui.Palette.GoldBright},
		{Text: ui.Bridge("THE BOOKS SETTLE IN "+Duration(a.SettleIn), "paid out as money in "+Duration(a.SettleIn)), Clr: ui.Palette.Text},
		paceLine(a),
		booksLine(f),
	}
	if a.LostAt > 0 {
		c.Lines = append(c.Lines, lossLine(a, f.State.RealmTime))
	}
	if l, ok := guildLine(f); ok {
		c.Lines = append(c.Lines, l)
	}
	return c
}

func guildLine(f Frame) (Line, bool) {
	for _, sc := range f.State.Storage.Classes {
		if sc.Name != view.GuildClass {
			continue
		}
		dflt := ""
		if sc.Default {
			dflt = " (default)"
		}
		return Line{Text: ui.Bridge("THE VAULT-WRIGHTS' GUILD DIGS A VAULT FOR EACH DEED WHEN ITS HUT FIRST NEEDS ONE",
			fmt.Sprintf("StorageClass %s%s: %s, reclaimPolicy %s", sc.Name, dflt, sc.BindingMode, sc.Reclaim)), Clr: ui.Palette.Blue}, true
	}
	return Line{}, false
}

func paceLine(a view.ArchiveView) Line {
	if a.Writers == 0 {
		return Line{Text: ui.Bridge("NO ARCHIVIST WRITES: SEALS GO UNRECORDED", "no ready archive Pod: requests are not recorded"), Clr: ui.Palette.RedText}
	}
	return Line{Text: ui.Bridge(fmt.Sprintf("%d WRITING, %s SEALS A SECOND AT MOST", a.Writers, Compact(a.Pace)), fmt.Sprintf("%d writer Pods, %s requests/s at most", a.Writers, Compact(a.Pace))), Clr: ui.Palette.Text}
}

func booksLine(f Frame) Line {
	if archiveIsStateful(f) {
		return Line{Text: ui.Bridge("EACH ARCHIVIST KEEPS ITS OWN DEED, WHATEVER KEEP IT LANDS ON", "volumeClaimTemplates: one PersistentVolumeClaim per Pod, kept across rescheduling"), Clr: ui.Palette.Green}
	}
	if claim := archiveDeed(f); claim != "" {
		return Line{Text: ui.Bridge("BOOKS IN DEED "+claim+": A REPLACED HUT FINDS THEM AGAIN", "books volume: PersistentVolumeClaim "+claim), Clr: ui.Palette.Green}
	}
	return Line{Text: ui.Bridge("LOOSE PAGES IN THE HUT: A REPLACED HUT LOSES THEM", "books volume: emptyDir, deleted with its Pod"), Clr: ui.Palette.RedText}
}

func lossLine(a view.ArchiveView, now float64) Line {
	story := fmt.Sprintf("LAST LOSS %s SEALS, -%s GOLD, %s AGO", Compact(a.LostSeals), Compact(a.LastLoss), Duration(now-a.LostAt))
	cause := "emptyDir deleted with its Pod"
	if a.LostCause == "persistentVolume" {
		cause = "PersistentVolume deleted"
	}
	names := fmt.Sprintf("last loss %s requests, -%s gold, %s ago: %s", Compact(a.LostSeals), Compact(a.LastLoss), Duration(now-a.LostAt), cause)
	return Line{Text: ui.Bridge(story, names), Clr: ui.Palette.RedText}
}

func storageVerbs(f Frame) []storageVerb {
	out := []storageVerb{}
	vaults := pageKnown(f, view.VaultsPage)
	if vaults && !guildStands(f) && !freeVault(f) {
		out = append(out, digVerb())
	}
	if vaults && !archiveIsStateful(f) && archiveDeed(f) == "" {
		if _, written := deedOf(f, booksDeed); !written {
			out = append(out, deedVerb())
		}
	}
	if pageKnown(f, view.StatefulCrewsPage) && !archiveIsStateful(f) {
		out = append(out, statefulVerb())
	}
	if pageKnown(f, view.GuildPage) && !guildStands(f) {
		out = append(out, guildVerb())
	}
	return out
}

func digVerb() storageVerb {
	price := Compact(view.VaultShelfPrice * booksShelves)
	return storageVerb{key: verbDig, label: ui.Bridge("DIG VAULT", "CREATE PV"),
		tip: ui.Bridge(fmt.Sprintf("Dig a Vault of %s on the home isle for %s gold. A Deed of that size binds to it.", view.Barrels(booksShelves*roomsPerGiB), price),
			fmt.Sprintf("Create a %dGi PersistentVolume (ReadWriteOnce, the home isle's zone) for %s gold. A claim of that size binds to it.", booksShelves, price)),
		act: view.Action{Action: view.ActDigVault, Shelves: booksShelves}}
}

func deedVerb() storageVerb {
	return storageVerb{key: verbDeed, label: ui.Bridge("WRITE DEED", "CREATE PVC"),
		tip: ui.Bridge(fmt.Sprintf("Write a Deed for %s and hand it to the Muniment Room. The books move into a Vault; the move is a rollout, so the loose pages in the old hut are lost once.", view.Barrels(booksShelves*roomsPerGiB)),
			fmt.Sprintf("Create PersistentVolumeClaim %s (%dGi) and mount it in Deployment archive as volume books. The rollout replaces the Pod, so its emptyDir is deleted once.", booksDeed, booksShelves)),
		act: view.Action{Action: view.ActWriteDeed, Name: booksDeed, Shelves: booksShelves}}
}

func statefulVerb() storageVerb {
	return storageVerb{key: verbStateful, label: ui.Bridge("RAISE STATEFUL CREW", "TO STATEFULSET"),
		tip: ui.Bridge("Make the archive a Stateful Crew: each archivist gets a number, a name and a Deed of its own, and a replaced archivist finds its books again.",
			"Convert Deployment archive into a StatefulSet with volumeClaimTemplates: stable names, one PersistentVolumeClaim per Pod, kept across rescheduling."),
		act: view.Action{Action: view.ActRaiseStatefulSet, From: view.ArchiveName}}
}

func guildVerb() storageVerb {
	return storageVerb{key: verbGuild, label: ui.Bridge("FOUND GUILD", "CREATE STORAGECLASS"),
		tip: ui.Bridge("Found the Vault-wrights' guild: every new Deed gets a Vault dug for it when its hut first needs one.",
			"Create StorageClass "+view.GuildClass+" (default, WaitForFirstConsumer): new claims are provisioned when their Pod is scheduled."),
		act: view.Action{Action: view.ActFoundGuild}}
}

func cardVerbs(f Frame, id string) []storageVerb {
	if id != archiveCardID || f.State.Archive == nil {
		return nil
	}
	return storageVerbs(f)
}

func storageVerbRows(verbs []storageVerb) [][]storageVerb {
	rows := [][]storageVerb{}
	width := KindW - 2*KindPad
	used := width
	for _, v := range verbs {
		w := verbWidth(v)
		if used+verbGap+w > width || len(rows) == 0 {
			rows, used = append(rows, nil), -verbGap
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], v)
		used += verbGap + w
	}
	return rows
}

func verbWidth(v storageVerb) int {
	return ui.Font{}.Width(v.label, 1) + verbTextPad
}

func verbBlock(f Frame, id string) int {
	return len(storageVerbRows(cardVerbs(f, id))) * (KindBtnH + verbGap)
}

func verbButtons(box ui.Rect, f Frame, id string) []Button {
	rows := storageVerbRows(cardVerbs(f, id))
	out := []Button{}
	for i, row := range rows {
		x := box.X + KindPad
		y := box.Y + box.H - KindPad - KindBtnH - (len(rows)-i)*(KindBtnH+verbGap)
		for _, v := range row {
			w := verbWidth(v)
			bid := KindKey + ":" + verbPrefix + v.key
			out = append(out, Button{ID: bid, Label: v.label, Tip: v.tip, Rect: ui.Rect{X: x, Y: y, W: w, H: KindBtnH}, Selected: bid == f.Guide.Button})
			x += w + verbGap
		}
	}
	return out
}

func (h *HUD) tryButtonGoal(f Frame) string {
	verb := strings.HasPrefix(f.Guide.Button, KindKey+":"+verbPrefix)
	switch {
	case verb && h.KindCard == archiveCardID:
		return f.Guide.Button
	case verb && h.crewsShown(f):
		return CrewsKey + ":open:" + archiveCardID
	case strings.HasPrefix(f.Guide.Button, CrewsKey+":") && h.crewsShown(f):
		if _, ok := HitID(h.crewsButtons(f), f.Guide.Button); ok {
			return f.Guide.Button
		}
	}
	return ""
}

func (h *HUD) verbAwaits(f Frame) bool {
	return strings.HasPrefix(f.Guide.Button, KindKey+":"+verbPrefix) && h.KindCard != archiveCardID
}

func (h *HUD) pressVerb(key string, f Frame, click Intent) []Intent {
	for _, v := range cardVerbs(f, h.KindCard) {
		if v.key == key {
			return []Intent{ActFrom(v.act, FromButton), click}
		}
	}
	return []Intent{click}
}

func archiveRowNote(f Frame) string {
	if f.State.Archive == nil {
		return ""
	}
	return ui.Bridge(Compact(f.State.Archive.Unsettled)+" GOLD UNSETTLED", "$"+Compact(f.State.Archive.Unsettled)+" unsettled")
}

func unregisteredNote(f Frame) string {
	if lost := f.State.SealsUnpaid; lost > 0 {
		return ui.Bridge("seals unregistered: -"+Compact(lost)+" gold/s", "no archive: -$"+Compact(lost)+"/s")
	}
	return ui.Bridge("no Muniment Room yet", "no archive yet")
}
