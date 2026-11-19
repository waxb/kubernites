package sound

const (
	titleA      = "D5 -- F5 A5  D5 -- C6 A5  G5 -- A5 F5  E5 D5 C5 A4  D5 -- F5 A5  D6 C6 A5 F5  G5 A5 C6 A5  F5 E5 D5 --"
	titleB      = "D5 -- -- F5  -- -- D5 --  E5 -- -- G5  -- -- E5 --  F5 -- E5 D5  -- -- A4 --  C#5 -- -- --  E5 -- -- --"
	titleArp    = "D4 A4 D5 A4  F4 C5 F5 C5  G4 D5 G5 D5  A4 E5 A5 E5"
	titleArpB   = "Bb3 F4 Bb4 F4  D5 F4 Bb4 F4  C4 G4 C5 G4  E5 G4 C5 G4  D4 A4 D5 A4  F5 A4 D5 A4  A3 E4 A4 E4  C#5 E4 A4 E4"
	titleBass   = "D2 -- D2 A2  F2 -- F2 C3  G2 -- G2 D3  A2 -- A2 A1"
	titleBassB  = "Bb1 -- -- --  F2 -- -- --  C2 -- -- --  G2 -- -- --  D2 -- -- --  A2 -- -- --  A1 -- -- --  E2 -- -- --"
	titleDrums  = "K H -- H  S H -- H  K H K H  S H -- H"
	titleDrumsB = "K -- -- H  -- -- H --  K -- -- H  S -- H --"
	crisisLead  = "D5 Eb5 D5 A4  D5 F5 Eb5 D5  C5 D5 C5 G4  C5 Eb5 D5 C5  Bb4 C5 D5 F5  Eb5 D5 C5 A4"
	crisisOst   = "D3 D3 A2 D3  Eb3 Eb3 Bb2 Eb3  C3 C3 G2 C3  Bb2 Bb2 F2 A2"
	crisisBass  = "D2 D2 D2 A1  D2 D2 F2 Eb2  C2 C2 C2 G1  Bb1 Bb1 C2 A1"
	crisisDrums = "K H K H  S H K H  K H K H  S H S H"
	crisisBooms = "B -- -- --  -- -- -- --  B -- -- --  -- -- -- --"
)

func titleBridgePads() []part {
	top := seq(rest(64), hold("D4", 8), hold("C4", 8), hold("D4", 8), hold("C#4", 8), rest(32))
	mid := seq(rest(64), hold("Bb3", 8), hold("G3", 8), hold("A3", 16), rest(32))
	low := seq(rest(64), hold("F3", 8), hold("E3", 8), hold("F3", 8), hold("E3", 8), rest(32))
	return []part{
		{inst: instPad, line: top, amp: 0.08, pan: 0.4, gate: 1, room: 0.5},
		{inst: instPad, line: mid, amp: 0.08, pan: -0.35, gate: 1, room: 0.5},
		{inst: instPad, line: low, amp: 0.08, gate: 1, room: 0.5},
	}
}

func titleSong() song {
	parts := []part{
		{inst: instLute, line: seq(titleA, titleA, rest(32), titleA), amp: 0.84, pan: -0.15, gate: 0.9, room: 0.3, echo: 0.1},
		{inst: instRecorder, line: seq(rest(32), titleA, rest(32), titleA), shift: -12, amp: 0.07, pan: 0.22, gate: 0.85, room: 0.4},
		{inst: instRecorder, line: seq(rest(64), titleB, rest(32)), amp: 0.16, pan: 0.15, gate: 0.92, room: 0.45},
		{inst: instChip, line: seq(times(titleArp, 4), titleArpB, times(titleArp, 2)), amp: 0.1, pan: 0.3, gate: 0.55, room: 0.25},
		{inst: instBass, line: seq(times(titleBass, 4), titleBassB, times(titleBass, 2)), amp: 0.21, gate: 0.8, room: 0.1},
	}
	drums := seq(times(titleDrums, 4), times(titleDrumsB, 2), times(titleDrums, 2))
	return song{
		bpm: 112, steps: 128, level: 0.66, room: 0.45, echo: 3, feedback: 0.28, seed: 0x4b554245,
		parts: append(parts, titleBridgePads()...),
		beats: []beat{{line: drums, amp: 0.4, room: 0.15}},
	}
}

func crisisSong() song {
	toll := seq("D4", hold(tokenHold, 31))
	return song{
		bpm: 148, steps: 192, level: 0.84, room: 0.4, seed: 0x43524953,
		parts: []part{
			{inst: instReed, line: seq(times(crisisLead, 4), rest(96)), amp: 0.27, pan: 0.15, gate: 0.72, room: 0.25},
			{inst: instBrass, line: seq(rest(96), times(crisisLead, 4)), shift: -12, amp: 0.22, pan: 0.1, gate: 0.8, room: 0.3},
			{inst: instLute, line: times(crisisOst, 12), amp: 0.24, pan: -0.25, gate: 0.65, room: 0.2},
			{inst: instBass, line: times(crisisBass, 12), amp: 0.17, gate: 0.85, room: 0.1},
			{inst: instDrone, line: seq(rest(96), hold("D2", 96)), amp: 0.07, pan: -0.2, gate: 1, room: 0.4},
			{inst: instDrone, line: seq(rest(96), hold("A2", 96)), amp: 0.06, pan: 0.2, gate: 1, room: 0.4},
			{inst: instToll, line: seq(rest(96), times(toll, 3)), amp: 0.13, pan: 0.3, room: 0.6},
		},
		beats: []beat{
			{line: times(crisisDrums, 12), amp: 0.3, room: 0.15},
			{line: seq(rest(96), times(crisisBooms, 6)), amp: 0.3, room: 0.3},
		},
	}
}
