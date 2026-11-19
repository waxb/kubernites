package soulstone

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	LatestTag      = "latest"
	CursedTag      = "2.1"
	baseMTBF       = 900.0
	reliableMTBF   = 4500.0
	cursedSeconds  = 1.5
	cursedTemplate = "The newest class, whispered to be brilliant. Nobody examined them."
)

var classTags = []string{"1.0", "1.1", "2.0", "3.0", LatestTag}

var classPace = []float64{1, 1, 2, 4, 8}

var classPrice = []float64{0, 1, 3, 5, 27}

type tradeLine struct {
	trade, profession, kind, boxes, care string
	firstCost, base, teach               float64
	description, startLine, crashLine    string
	boxMilli, boxMem                     [SizeClasses]float64
	thread, idle, memBase, memPerRps     float64
	pull, start, size                    float64
	pledgeCPU, pledgeMemory              string
}

var tradeLines = []tradeLine{
	{trade: "porter", profession: "Porter", kind: KindParcel, boxes: "parcels", care: "Taught to lift with their legs",
		firstCost: 0, base: 1000, teach: 150,
		description: "The founding trade. Nimble hands for parcels, who drop one now and then.",
		startLine:   "shouldering the first parcel", crashLine: "fatal: a parcel slipped and burst (exit status 1)",
		boxMilli: [SizeClasses]float64{1200, 2400, 4800, 9600}, boxMem: [SizeClasses]float64{4, 12, 40, 128},
		thread: 1000, idle: 4, memBase: 40, memPerRps: 3, pull: 4, start: 2, size: 38},
	{trade: "clerk", profession: "Clerk", kind: KindLedger, boxes: "ledgers", care: "Taught to blot before they turn the page",
		firstCost: 200, base: 9000,
		description: "Patient hands that write every ledger into the books, like a database. They need much housing.",
		startLine:   "unrolling fresh vellum", crashLine: "fatal: the ink ran dry (exit status 1)",
		boxMilli: [SizeClasses]float64{1650, 2860, 4950, 8800}, boxMem: [SizeClasses]float64{24, 64, 160, 384},
		thread: 1000, idle: 6, memBase: 256, memPerRps: 24, pull: 6, start: 3, size: 60},
	{trade: "hauler", profession: "Hauler", kind: KindCrate, boxes: "crates", care: "Taught to chock the cart wheels",
		firstCost: 250, base: 30000,
		description: "Broad backs for crates and chests. Big loads cost them little extra food, but every load takes rooms in their housing.",
		startLine:   "testing the cart axle", crashLine: "fatal: the cart axle cracked (exit status 1)",
		boxMilli: [SizeClasses]float64{4000, 5000, 7000, 10000}, boxMem: [SizeClasses]float64{8, 32, 96, 256},
		thread: 1000, idle: 6, memBase: 96, memPerRps: 8, pull: 5, start: 2, size: 52, pledgeCPU: "1", pledgeMemory: "1Gi"},
	{trade: "courier", profession: "Courier", kind: KindDispatch, boxes: "dispatches", care: "Taught to double-knot the satchel",
		firstCost: 350, base: 330000,
		description: "Batch riders for dispatches. They wait for a full satchel, never mind a delay, and travel light on housing.",
		startLine:   "packing the satchel", crashLine: "fatal: the satchel strap tore (exit status 1)",
		boxMilli: [SizeClasses]float64{5000, 8900, 15600, 27800}, boxMem: [SizeClasses]float64{6, 24, 72, 200},
		thread: 1000, idle: 4, memBase: 48, memPerRps: 4, pull: 3, start: 2, size: 34, pledgeCPU: "1", pledgeMemory: "256Mi"},
	{trade: "sealwright", profession: "Sealwright", kind: KindSealed, boxes: "sealed boxes", care: "Taught to temper the wax",
		firstCost: 500, base: 3300000,
		description: "Wax and signet for sealed boxes, which pass the Gateway unopened, like TLS passthrough. Every seal costs them dear in food.",
		startLine:   "warming the sealing wax", crashLine: "fatal: the signet cracked (exit status 1)",
		boxMilli: [SizeClasses]float64{16700, 29200, 50100, 83500}, boxMem: [SizeClasses]float64{6, 20, 64, 192},
		thread: 2000, idle: 10, memBase: 64, memPerRps: 6, pull: 5, start: 3, size: 48, pledgeCPU: "2", pledgeMemory: "512Mi"},
}

func ClassTags() []string {
	return append([]string(nil), classTags...)
}

func ClassIndex(tag string) int {
	for i, t := range classTags {
		if t == tag {
			return i
		}
	}
	return -1
}

func TradeNames() []string {
	out := make([]string, 0, len(tradeLines))
	for _, l := range tradeLines {
		out = append(out, l.trade)
	}
	return out
}

func ClassOf(trade string, class int) (Profile, bool) {
	if class < 0 || class >= len(classTags) {
		return Profile{}, false
	}
	return Lookup(trade + ":" + classTags[class])
}

func ladderProfiles() []Profile {
	out := []Profile{}
	for _, l := range tradeLines {
		for i := range classTags {
			out = append(out, l.class(i))
		}
		out = append(out, l.cursed())
	}
	return out
}

func (l tradeLine) profile(tag string, pace float64) Profile {
	p := Profile{
		Image: l.trade + ":" + tag, Name: className(l.profession, tag), Profession: l.profession, Sprite: l.trade,
		Trade: l.trade, Kinds: []string{l.kind}, Port: TradePort, ThreadMilli: l.thread, IdleMilli: l.idle,
		MemBaseMi: l.memBase, MemPerRpsMi: l.memPerRps, PullSeconds: l.pull, StartSeconds: l.start, SizeMi: l.size, BoxMemMi: l.boxMem,
		StartLines: []string{"listening for " + l.boxes + " on :8080", l.startLine}, CrashLine: l.crashLine,
		PledgeCPU: l.pledgeCPU, PledgeMemory: l.pledgeMemory, MTBFSeconds: reliableMTBF,
	}
	for s := range p.BoxMilliS {
		p.BoxMilliS[s] = l.boxMilli[s] / pace
	}
	p.CPUPerPilgrimMilliS = p.BoxMilliS[SizeParcel]
	return p
}

func (l tradeLine) class(i int) Profile {
	p := l.profile(classTags[i], classPace[i])
	p.Cost, p.Description = l.price(i), l.classWords(i)
	if i == 0 {
		p.MTBFSeconds = baseMTBF
	}
	return p
}

func (l tradeLine) cursed() Profile {
	p := l.profile(CursedTag, classPace[ClassIndex("2.0")])
	p.Cursed, p.CurseSeconds, p.MTBFSeconds, p.Description = true, cursedSeconds, 0, cursedTemplate
	p.StartLines = []string{"reading scroll " + l.trade + "-config"}
	p.CrashLine = fmt.Sprintf("panic: scroll key %s_ROUTE missing", strings.ToUpper(l.trade))
	return p
}

func (l tradeLine) price(i int) float64 {
	switch {
	case i == 0:
		return l.firstCost
	case i == 1 && l.teach > 0:
		return l.teach
	}
	return TwoFigures(l.base * classPrice[i])
}

func (l tradeLine) classWords(i int) string {
	switch classTags[i] {
	case "1.0":
		return l.description
	case "1.1":
		return l.care + ": the same pace, and they stop five times less often."
	case "2.0":
		return "Twice the " + l.boxes + " of 1.0 for the same food."
	case "3.0":
		return "Four times the " + l.boxes + " of 1.0 for the same food."
	}
	return "The Academy's newest class: eight times the " + l.boxes + " of 1.0 for the same food. latest is a name, not a number: the Academy may send another class under it, and a crew that asks for latest cannot tell which one it wears."
}

func TwoFigures(v float64) float64 {
	if v <= 0 {
		return 0
	}
	mag := math.Pow(10, math.Floor(math.Log10(v))-1)
	return math.Round(v/mag) * mag
}

func className(profession, tag string) string {
	if tag == LatestTag {
		return profession + "s, the latest class"
	}
	return fmt.Sprintf("%ss, class of %s", profession, tag)
}

func ClassWords(tag string) string {
	if tag == LatestTag {
		return "the latest class"
	}
	return "class " + tag
}

func TradeClass(trade, tag string) string {
	if tag == LatestTag {
		return "the latest " + trade + " class"
	}
	return trade + " class " + tag
}

func isLatestTag(tag string) bool {
	return tag == "" || tag == LatestTag
}

func TagNewer(a, b string) bool {
	if al, bl := isLatestTag(a), isLatestTag(b); al || bl {
		return al && !bl
	}
	am, an := tagParts(a)
	bm, bn := tagParts(b)
	return am > bm || am == bm && an > bn
}

func tagParts(tag string) (int, int) {
	major, minor, _ := strings.Cut(tag, ".")
	m, _ := strconv.Atoi(major)
	n, _ := strconv.Atoi(minor)
	return m, n
}

func ladderAliases() map[string]string {
	out := map[string]string{}
	for _, l := range tradeLines {
		out[l.trade] = l.trade + ":" + LatestTag
	}
	return out
}
