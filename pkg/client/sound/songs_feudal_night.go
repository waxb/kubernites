package sound

func feudalNightSong() song {
	harp := seq(arp4("G3", "D4", "B4", "D4"), arp4("E3", "B3", "G4", "B3"), arp4("C3", "G3", "E4", "G3"), arp4("D3", "A3", "F#4", "A3"))
	tune := seq(rest(32), realmA, rest(32), realmA2, rest(64), realmB, rest(32))
	horns := seq(hold("B3", 32), hold("C4", 32), hold("B3", 32), hold("A3", 32))
	bass := seq(halves("G2", "D2"), halves("E2", "B1"), halves("C2", "G1"), halves("D2", "A1"))
	return song{
		bpm: 84, steps: 256, level: 0.62, room: 0.55, echo: 3, feedback: 0.36, seed: 0x46454e49,
		parts: append([]part{
			{inst: instHarp, line: harp, amp: 0.36, pan: -0.25, gate: 1, room: 0.5, echo: 0.3},
			{inst: instRecorder, line: tune, amp: 0.12, pan: 0.15, gate: 0.95, room: 0.55, echo: 0.14},
			{inst: instHorn, line: horns, amp: 0.07, pan: 0.3, gate: 1, room: 0.55},
			{inst: instBass, line: bass, amp: 0.15, gate: 0.95, room: 0.15},
			{inst: instDrone, line: seq(hold("G2", 64), hold("D2", 64)), amp: 0.04, gate: 1, room: 0.5},
		}, nightPads()...),
		beats: []beat{{line: nightKit, amp: 0.24, room: 0.3}},
	}
}
