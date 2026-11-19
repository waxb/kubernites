package sound

const feudalBassC = "E2 -- B2 G2  C2 -- G2 E2  G2 -- D3 B2  D2 -- A2 F#2"

func feudalBass128() string {
	return seq(times(feudalBass, 4), realmBassB, times(feudalBass, 2))
}

func feudalDrums128() string {
	return seq(times(feudalDrums, 4), times(realmDrums, 2), times(feudalDrums, 2))
}

func feudalLuteRise() string {
	return seq(rest(32), darkC2, rest(32), rest(64), darkTurn, realmA2, realmA)
}

func feudalBrassRise() string {
	return seq(darkC1, rest(32), rest(32), rest(64), rest(32), rest(32), realmA)
}

func feudalHornRise() string {
	return seq(rest(96), hold("B3", 16), hold("C4", 16), hold("D4", 16), hold("A3", 16), rest(96))
}

func feudalHarpRise() string {
	return seq(times(darkHarmC, 4), darkA3, darkBreakdownChords(), times(realmHarm, 2), times(realmHarm, 4))
}

func feudalBassRise() string {
	return seq(times(feudalBassC, 4), times(feudalBass, 2), hold("E2", 16), hold("C2", 16), hold("G2", 16), hold("D2", 16), times(feudalBass, 2), times(feudalBass, 4))
}

func feudalDrumsRise() string {
	return seq(times(feudalDrums, 4), times(realmDrums2, 2), rest(32), times(realmDrumsB, 2), times(feudalDrums, 2), times(feudalDrums, 4))
}
