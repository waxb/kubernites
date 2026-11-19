package sound

const (
	darkC1    = "B4 -- -- E5  D5 -- B4 --  G4 -- A4 B4  A4 -- -- --  B4 -- -- E5  D5 -- B4 D5  E5 -- D5 B4  A4 -- -- --"
	darkC2    = "G4 -- -- B4  C5 -- E5 --  D5 -- B4 G4  A4 -- F#4 --  E4 -- G4 B4  C5 -- B4 A4  G4 -- A4 B4  D5 -- -- --"
	darkA3    = "G4 A4 B4 D5  -- B4 A4 G4  E4 F#4 G4 B4  -- A4 G4 --  C5 D5 B4 G4  A4 G4 E4 --  D4 G4 A4 B4  G4 -- -- --"
	darkTurn  = "D4 -- G4 --  B4 -- D5 --  -- C5 B4 --  A4 -- -- --  G4 -- B4 --  A4 -- F#4 --  D4 -- -- --  D4 -- -- --"
	darkHarmC = "E3 B3 E4 B3  C3 G3 C4 G3  G3 D4 G4 D4  D3 A3 D4 A3"
	darkBassC = "E2 -- B2 --  C2 -- G2 --  G2 -- D3 --  D2 -- A2 --"
)

func realmBass128() string {
	return seq(times(realmBass, 4), realmBassB, times(realmBass, 2))
}

func realmDrums128() string {
	return seq(times(realmDrums, 2), times(realmDrums2, 2), times(realmDrumsB, 2), times(realmDrums, 2))
}

func darkBreakdownChords() string {
	return seq(arp4("E3", "B3", "G4", "B3"), arp4("C3", "G3", "E4", "G3"), arp4("G3", "D4", "B4", "D4"), arp4("D3", "A3", "F#4", "A3"))
}

func darkLuteRise() string {
	return seq(darkC1, rest(32), rest(32), rest(64), darkTurn, realmA2, realmA)
}

func darkBreath() string {
	return seq(hold("B4", 8), hold("G4", 8), hold("C5", 8), hold("E4", 8), hold("D5", 8), hold("B4", 8), hold("A4", 12), rest(4))
}

func darkRecorderRise() string {
	return seq(rest(32), darkC2, rest(32), darkBreath(), rest(32), rest(32), realmA)
}

func darkHarpRise() string {
	return seq(rest(64), darkA3, darkBreakdownChords(), rest(32), rest(64))
}

func darkChipRise() string {
	return seq(times(darkHarmC, 4), times(realmHarm, 2), rest(64), times(realmHarm, 2), times(realmHarm, 4))
}

func darkBassRise() string {
	return seq(times(darkBassC, 4), times(realmBass, 2), hold("E2", 16), hold("C2", 16), hold("G2", 16), hold("D2", 16), times(realmBass, 2), times(realmBass, 4))
}

func darkDrumsRise() string {
	return seq(times(realmDrums, 4), times(realmDrums2, 2), rest(32), times(realmDrumsB, 2), times(realmDrums2, 2), times(realmDrums, 4))
}

func darkPads() []part {
	top := seq(rest(64), hold("B3", 8), hold("C4", 8), hold("D4", 8), hold("A3", 8), rest(32), rest(64), hold("B3", 8), hold("C4", 8), hold("D4", 8), hold("A3", 8), rest(32),
		rest(96), hold("B3", 16), hold("C4", 16), hold("B3", 16), hold("A3", 16), rest(96))
	mid := seq(rest(64), hold("G3", 16), hold("B3", 8), hold("F#3", 8), rest(32), rest(64), hold("G3", 16), hold("B3", 8), hold("F#3", 8), rest(32),
		rest(96), hold("G3", 16), hold("G3", 16), hold("G3", 16), hold("F#3", 16), rest(96))
	low := seq(rest(64), hold("E3", 16), hold("G3", 8), hold("D3", 8), rest(32), rest(64), hold("E3", 16), hold("G3", 8), hold("D3", 8), rest(32),
		rest(96), hold("E3", 16), hold("C3", 16), hold("G3", 16), hold("D3", 16), rest(96))
	return []part{
		{inst: instPad, line: top, amp: 0.08, pan: 0.4, gate: 1, room: 0.5},
		{inst: instPad, line: mid, amp: 0.08, pan: -0.35, gate: 1, room: 0.5},
		{inst: instPad, line: low, amp: 0.08, gate: 1, room: 0.5},
	}
}
