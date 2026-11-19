package art

func squareMast(at, height, span float64, tiers int, width int) mastSpec {
	m := mastSpec{at: at, height: height, width: width, pennant: 0.35 + height/100, flutterAmp: 1.2}
	if tiers == 1 {
		m.yards = []yardSpec{{top: height - 2, foot: height * 0.28, span: span}}
		return m
	}
	m.nest = true
	m.yards = []yardSpec{
		{top: height * 0.62, foot: height * 0.2, span: span},
		{top: height - 1, foot: height * 0.66, span: span * 0.72},
	}
	return m
}

func lateenMast(at, height, fore, aft, over float64) mastSpec {
	return mastSpec{at: at, height: height, lateen: true, fore: fore, aft: aft, over: over, width: 1, pennant: 0.25, flutterAmp: 0.8}
}

var shipSpecs = map[string]shipSpec{
	"cog": {
		w: 29, h: 36, ax: 14, ay: 28,
		length: 1.2, beam: 0.54, freeboard: 5, sheer: 3, rise: 2, bulwark: 2, stripe: 1,
		bowPow: 1.7, sternPinch: 0.55,
		castles:  []castleSpec{{from: -1, to: -0.62, height: 3}},
		masts:    []mastSpec{squareMast(0.1, 16, 0.34, 1, 1)},
		bowsprit: 0.15, billow: 0.16,
	},
	"caravel": {
		w: 40, h: 45, ax: 20, ay: 34,
		length: 1.8, beam: 0.6, freeboard: 5, sheer: 3, rise: 2, bulwark: 2, stripe: 1,
		bowPow: 2, sternPinch: 0.45,
		castles: []castleSpec{{from: -1, to: -0.55, height: 3}},
		masts: []mastSpec{
			lateenMast(0.18, 15, 0.42, 0.55, 0.7),
			lateenMast(-0.5, 11, 0.3, 0.38, 0.6),
		},
		bowsprit: 0.12, billow: 0.12,
	},
	"galleon": {
		w: 56, h: 59, ax: 28, ay: 45,
		length: 2.5, beam: 0.7, freeboard: 7, sheer: 4, rise: 3, bulwark: 2, stripe: 2,
		bowPow: 2.2, sternPinch: 0.3,
		castles: []castleSpec{{from: -1, to: -0.5, height: 6}, {from: 0.55, to: 0.82, height: 4}},
		masts: []mastSpec{
			squareMast(0.45, 21, 0.36, 2, 1),
			squareMast(0.0, 26, 0.42, 2, 2),
			lateenMast(-0.62, 13, 0.3, 0.38, 0.6),
		},
		bowsprit: 0.35, billow: 0.18, gunports: true, stays: true,
	},
	"carrack": {
		w: 66, h: 70, ax: 33, ay: 54,
		length: 3.0, beam: 0.9, freeboard: 8, sheer: 5, rise: 3, bulwark: 2, stripe: 2,
		bowPow: 1.7, sternPinch: 0.35,
		castles: []castleSpec{{from: -1, to: -0.45, height: 8}, {from: 0.55, to: 1, height: 7}},
		masts: []mastSpec{
			squareMast(0.42, 22, 0.4, 2, 1),
			squareMast(0.0, 30, 0.5, 2, 2),
			lateenMast(-0.55, 16, 0.34, 0.42, 0.6),
			lateenMast(-0.85, 11, 0.24, 0.3, 0.55),
		},
		bowsprit: 0.4, billow: 0.2, stays: true,
	},
}
