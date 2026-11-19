package sound

const (
	realmA      = "G4 -- B4 D5  -- B4 A4 --  E4 -- G4 B4  -- A4 G4 --  C5 -- B4 G4  A4 -- E4 --  D4 G4 A4 B4  G4 -- -- --"
	realmA2     = "G4 -- B4 D5  -- B4 A4 --  E4 -- G4 B4  -- A4 G4 --  D5 -- B4 D5  E5 -- D5 B4  C5 -- G4 C5  A4 -- F#4 A4"
	realmB      = "B4 -- -- A4  G4 -- -- E4  G4 -- -- A4  C5 -- -- B4  D5 -- -- B4  G4 -- B4 --  A4 -- -- --  F#4 -- -- --"
	realmHarm   = "G3 D4 G4 D4  E3 B3 E4 B3  C3 G3 C4 G3  D3 A3 D4 A3"
	realmHarmB  = "E3 B3 E4 B3  G4 B3 E4 B3  C3 G3 C4 G3  E4 G3 C4 G3  G3 D4 G4 D4  B4 D4 G4 D4  D3 A3 D4 A3  F#4 A3 D4 A3"
	realmBass   = "G2 -- D3 --  E2 -- B2 --  C2 -- G2 --  D2 -- D2 --"
	realmBassB  = "E2 -- -- --  B2 -- -- --  C2 -- -- --  G2 -- -- --  G2 -- -- --  D3 -- -- --  D2 -- -- --  A2 -- -- --"
	feudalBass  = "G2 -- D3 B2  E2 -- B2 G2  C2 -- G2 E2  D2 -- A2 F#2"
	realmDrums  = "K -- H --  -- -- H --  K -- H --  S -- H --"
	realmDrums2 = "K -- H T  -- -- H --  K -- H --  S -- H T"
	realmDrumsB = "K -- -- --  -- -- H --  -- -- H --  S -- -- --"
	feudalDrums = "K -- H --  J -- H T  K -- H K  S -- H T"
	nightKit    = "K -- -- --  -- -- -- --  -- -- -- --  -- -- -- --  K -- -- --  -- -- -- --  -- -- -- --  -- -- H --"
)

func realmLead() string {
	return seq(realmA, realmA2, rest(32), realmA)
}

func realmHarmony() string {
	return seq(times(realmHarm, 4), realmHarmB, times(realmHarm, 2))
}

func realmBridgePads() []part {
	top := seq(rest(64), hold("B3", 8), hold("C4", 8), hold("D4", 8), hold("A3", 8), rest(32))
	mid := seq(rest(64), hold("G3", 16), hold("B3", 8), hold("F#3", 8), rest(32))
	low := seq(rest(64), hold("E3", 16), hold("G3", 8), hold("D3", 8), rest(32))
	return []part{
		{inst: instPad, line: top, amp: 0.08, pan: 0.4, gate: 1, room: 0.5},
		{inst: instPad, line: mid, amp: 0.08, pan: -0.35, gate: 1, room: 0.5},
		{inst: instPad, line: low, amp: 0.08, gate: 1, room: 0.5},
	}
}

func darkSong() song {
	lute := seq(realmLead(), rest(32), realmA2, realmB, realmA, darkLuteRise())
	recorder := seq(rest(64), realmB, rest(32), realmA, rest(96), darkRecorderRise())
	chip := seq(realmHarmony(), rest(32), times(realmHarm, 2), realmHarmB, times(realmHarm, 2), darkChipRise())
	bass := seq(realmBass128(), realmBass128(), darkBassRise())
	parts := []part{
		{inst: instLute, line: lute, amp: 0.89, pan: -0.12, gate: 0.95, room: 0.35, echo: 0.08},
		{inst: instLute, line: seq(rest(128), times(realmHarm, 2), rest(96), rest(192), times(realmHarm, 4)), amp: 0.34, pan: 0.3, gate: 0.7, room: 0.3},
		{inst: instRecorder, line: recorder, amp: 0.17, pan: 0.18, gate: 0.92, room: 0.45},
		{inst: instRecorder, line: seq(rest(160), realmA2, rest(64), rest(256)), shift: 12, amp: 0.05, pan: 0.3, gate: 0.85, room: 0.45},
		{inst: instHarp, line: seq(rest(256), darkHarpRise()), amp: 0.24, pan: 0.28, gate: 0.9, room: 0.45, echo: 0.2},
		{inst: instChip, line: chip, amp: 0.1, pan: 0.32, gate: 0.55, room: 0.25},
		{inst: instBass, line: bass, amp: 0.23, gate: 0.7, room: 0.1},
	}
	return song{
		bpm: 88, steps: 512, level: 0.82, room: 0.45, echo: 3, feedback: 0.3, seed: 0x5245414c,
		parts: append(parts, darkPads()...),
		beats: []beat{{line: seq(realmDrums128(), realmDrums128(), darkDrumsRise()), amp: 0.5, room: 0.15}},
	}
}

func nightSong() song {
	harp := seq(arp4("G3", "D4", "B4", "D4"), arp4("E3", "B3", "G4", "B3"), arp4("C3", "G3", "E4", "G3"), arp4("D3", "A3", "F#4", "A3"),
		arp4("G3", "D4", "B4", "D4"), arp4("E3", "B3", "G4", "B3"), arp4("A2", "E3", "C4", "E3"), arp4("D3", "A3", "F#4", "A3"))
	verse := seq(rest(12), "D4 -- G4 --", hold("B4", 8), hold("A4", 4), hold("G4", 4), hold("E4", 8), hold("G4", 6), rest(2),
		hold("F#4", 4), hold("A4", 8), rest(4))
	answer := seq(rest(12), "B4 -- D5 --", hold("E5", 8), hold("D5", 4), hold("B4", 4),
		hold("C5", 6), hold("B4", 2), hold("A4", 8), hold("F#4", 8), hold("A4", 4), rest(4))
	stars := seq(rest(28), hold("B5", 32), hold("A5", 22), hold("D6", 30), hold("F#5", 16),
		rest(20), hold("B5", 40), hold("A5", 36), hold("E6", 20), hold("F#5", 12))
	bass := seq(halves("G2", "D2"), halves("E2", "B1"), halves("C2", "G1"), halves("D2", "A1"),
		halves("G2", "D2"), halves("E2", "B1"), halves("A1", "E2"), halves("D2", "A1"))
	return song{
		bpm: 88, steps: 256, level: 0.6, room: 0.55, echo: 3, feedback: 0.38, seed: 0x4e494748,
		parts: append([]part{
			{inst: instHarp, line: harp, amp: 0.4, pan: -0.25, gate: 1, room: 0.5, echo: 0.35},
			{inst: instRecorder, line: seq(verse, answer, rest(64), answer), amp: 0.14, pan: 0.15, gate: 0.95, room: 0.55, echo: 0.12},
			{inst: instBass, line: bass, amp: 0.15, gate: 0.95, room: 0.15},
			{inst: instChime, line: stars, amp: 0.1, pan: 0.4, room: 0.7, echo: 0.3},
		}, nightPads()...),
		beats: []beat{{line: nightKit, amp: 0.28, room: 0.3}},
	}
}

func nightPads() []part {
	low := seq(hold(tokenHold, 16), hold("E3", 32), hold("D3", 32), hold("E3", 32), hold("D3", 16))
	mid := seq(hold("G3", 48), hold("F#3", 16), hold("G3", 32), hold("A3", 16), hold("F#3", 16))
	top := seq(hold("B3", 32), hold("C4", 16), hold("A3", 16), hold("B3", 32), hold("C4", 16), hold("A3", 16))
	return []part{
		{inst: instPad, line: low, amp: 0.05, gate: 1, room: 0.6},
		{inst: instPad, line: mid, amp: 0.05, pan: -0.4, gate: 1, room: 0.6},
		{inst: instPad, line: top, amp: 0.05, pan: 0.35, gate: 1, room: 0.6},
	}
}

func feudalSong() song {
	hornA := seq(hold("B3", 8), hold("A3", 8), hold("D4", 8), hold("A3", 4), hold("F#3", 4))
	hornB := seq(hold("G3", 16), hold("B3", 8), hold("A3", 8))
	double := seq(realmA, rest(64), realmA, rest(32), realmA2, rest(32), realmA)
	parts := []part{
		{inst: instLute, line: seq(realmLead(), rest(32), realmA2, rest(32), realmA, feudalLuteRise()), amp: 0.84, pan: -0.15, gate: 0.95, room: 0.3, echo: 0.06},
		{inst: instBrass, line: seq(rest(128), realmA, rest(96), feudalBrassRise()), amp: 0.2, pan: 0.1, gate: 0.85, room: 0.35},
		{inst: instRecorder, line: seq(double, rest(224), realmA), shift: 12, amp: 0.056, pan: 0.28, gate: 0.85, room: 0.4},
		{inst: instRecorder, line: seq(rest(64), realmB, rest(32), rest(64), realmB, rest(32), rest(96), darkBreath(), rest(96)), amp: 0.17, pan: 0.2, gate: 0.92, room: 0.4},
		{inst: instHorn, line: seq(times(seq(rest(32), hornA, hornB, hornA), 2), feudalHornRise()), amp: 0.12, pan: -0.3, gate: 0.97, room: 0.45},
		{inst: instHarp, line: seq(realmHarmony(), realmHarmony(), feudalHarpRise()), amp: 0.18, pan: 0.35, gate: 0.8, room: 0.3},
		{inst: instChip, line: seq(realmHarmony(), realmHarmony(), darkChipRise()), shift: 12, amp: 0.035, pan: 0.45, gate: 0.4, room: 0.2},
		{inst: instBass, line: seq(feudalBass128(), feudalBass128(), feudalBassRise()), amp: 0.23, gate: 0.75, room: 0.1},
		{inst: instDrone, line: seq(times(seq(hold(tokenHold, 64), rest(32), hold("G2", 32)), 2), rest(96), hold("E2", 32), hold("D2", 32), rest(96)), amp: 0.06, pan: -0.1, gate: 1, room: 0.4},
		{inst: instDrone, line: seq(times(seq(hold(tokenHold, 64), rest(32), hold("D3", 32)), 2), rest(96), hold("B2", 32), hold("A2", 32), rest(96)), amp: 0.05, pan: 0.1, gate: 1, room: 0.4},
	}
	return song{
		bpm: 96, steps: 512, level: 0.85, room: 0.45, echo: 3, feedback: 0.25, seed: 0x46455544,
		parts: append(parts, darkPads()...),
		beats: []beat{{line: seq(feudalDrums128(), feudalDrums128(), feudalDrumsRise()), amp: 0.46, room: 0.15}},
	}
}
