package sound

var (
	glassPartials = []partial{{1, 1, 1}, {2.0, 0.25, 0.5}, {3.0, 0.08, 0.3}, {4.1, 0.05, 0.2}}
	coinPartials  = []partial{{1, 1, 1}, {2.0, 0.3, 0.6}, {2.76, 0.15, 0.4}}
	tollPartials  = []partial{
		{0.5, 0.55, 1.0}, {1.0, 1.0, 0.85}, {1.003, 0.4, 0.8}, {1.19, 0.5, 0.6}, {1.5, 0.3, 0.45},
		{2.0, 0.65, 0.5}, {2.51, 0.22, 0.3}, {3.0, 0.25, 0.25}, {4.07, 0.14, 0.18}, {5.36, 0.08, 0.12},
	}
	softPulse = instrument{kind: kindPulse, attack: 0.004, release: 0.06, decay: 0.5, duty: 0.5, tone: 900}
	softHorn  = instrument{kind: kindHorn, attack: 0.25, release: 0.45, tone: 2.2, vibrato: 0.004}
	warHorn   = instrument{kind: kindHorn, attack: 0.045, release: 0.3, tone: 5, vibrato: 0.003}
	drumRoll  = []float64{0, 0.2, 0.36, 0.49, 0.59, 0.67, 0.73, 0.78}
)

var sfxSpecs = map[SFX]sfxSpec{
	SFXClick:     {length: 0.07, level: 0.4, priority: 0, seed: 0xc11c, draw: drawClick},
	SFXCoin:      {length: 0.45, level: 0.55, priority: 1, seed: 0xc014, draw: drawCoin},
	SFXBuild:     {length: 0.5, level: 0.6, room: 0.12, priority: 1, seed: 0xb017d, draw: drawBuild},
	SFXBuilt:     {length: 1.05, level: 0.62, room: 0.25, priority: 2, seed: 0xb0117, draw: drawBuilt},
	SFXPodStart:  {length: 0.85, level: 0.55, room: 0.15, priority: 1, seed: 0x9057a, draw: drawPodStart},
	SFXError:     {length: 0.45, level: 0.38, priority: 2, seed: 0xe770, draw: drawError},
	SFXUnlock:    {length: 1.25, level: 0.6, room: 0.3, priority: 3, seed: 0x1ec4, draw: drawUnlock},
	SFXRollout:   {length: 1.15, level: 0.48, room: 0.2, priority: 1, seed: 0x7011, draw: drawRollout},
	SFXVictory:   {length: 2.2, level: 0.75, room: 0.3, priority: 4, seed: 0x71c7, draw: drawVictory},
	SFXTremor:    {length: 1.5, level: 0.72, priority: 3, seed: 0x7e30, draw: drawTremor},
	SFXCalamity:  {length: 2.45, level: 0.78, room: 0.35, priority: 4, seed: 0xca1a, draw: drawCalamity},
	SFXBoat:      {length: 1.6, level: 0.55, room: 0.3, priority: 1, seed: 0xb0a7, draw: drawBoat},
	SFXChronicle: {length: 1.1, level: 0.55, room: 0.25, priority: 2, seed: 0xc4a0, draw: drawChronicle},
	SFXReveal:    {length: 0.95, level: 0.45, room: 0.35, priority: 2, seed: 0x4e7e, draw: drawReveal},
	SFXDrums:     {length: 1.45, level: 0.72, room: 0.22, priority: 3, seed: 0xd4d5, draw: drawDrums},
	SFXHorn:      {length: 1.6, level: 0.56, room: 0.35, priority: 2, seed: 0x4042, draw: drawHorn},
	SFXReady:     {length: 0.85, level: 0.42, room: 0.3, priority: 2, seed: 0x4ead, draw: drawReady},
	SFXGleamLast: {length: 1.6, level: 0.7, room: 0.3, priority: 3, seed: 0x91e3, draw: drawGleamLast},
	SFXPage:      {length: 0.45, level: 0.3, room: 0.15, priority: 1, seed: 0x9a6e, draw: drawPage},
}

func drawClick(c *canvas) {
	c.tone(0, glide{1500, 1100, 0.05, 0.03}, 1, 0)
	c.tone(0, glide{3100, 2600, 0.03, 0.015}, 0.3, 0)
	c.noise(0, hiss{0.015, 2000, 6000, 0.0005, 0.01}, 0.6, 0)
}

func drawCoin(c *canvas) {
	c.bell(0, strike{987.77, 0.15, 0.1}, 0.8, -0.15, coinPartials)
	c.bell(0.08, strike{1318.51, 0.35, 0.37}, 0.85, 0.15, coinPartials)
}

func drawBuild(c *canvas) {
	for i, t := range []float64{0, 0.2} {
		pitch := 1 + 0.1*float64(i)
		pan := -0.12 + 0.24*float64(i)
		c.noise(t, hiss{0.05, 900, 4200, 0.0008, 0.035}, 1.4, pan)
		c.tone(t, glide{190 * pitch, 170 * pitch, 0.12, 0.07}, 0.8, pan)
		c.tone(t, glide{420 * pitch, 400 * pitch, 0.08, 0.04}, 0.4, pan)
	}
	c.tone(0.2, glide{2900, 2880, 0.12, 0.09}, 0.08, 0.1)
}

func drawBuilt(c *canvas) {
	for i, f := range []float64{392, 493.88, 587.33} {
		c.note(instHorn, 0.13*float64(i), 0.12, f, 0.55, -0.2+0.2*float64(i))
	}
	c.note(instHorn, 0.39, 0.45, 783.99, 0.6, 0.15)
	for i, f := range []float64{196, 246.94, 293.66} {
		c.note(instHarp, 0.39+0.015*float64(i), 0.5, f, 0.5, -0.1+0.1*float64(i))
	}
	c.hit('J', 0.39, 0.35, 0.2)
}

func drawPodStart(c *canvas) {
	c.noise(0, hiss{0.5, 300, 2200, 0.12, 0.35}, 0.5, 0)
	for i := 0; i < 7; i++ {
		t := 0.05 + 0.5*c.rng.unit()
		c.noise(t, hiss{0.012, 1500, 8000, 0.0005, 0.008}, 0.4+0.3*c.rng.unit(), 0.5*c.rng.signed())
	}
	for i, f := range []float64{261.63, 329.63, 392, 523.25} {
		c.note(instHarp, 0.06+0.08*float64(i), 0.3, f, 0.8, -0.15+0.1*float64(i))
	}
}

func drawError(c *canvas) {
	c.note(softPulse, 0, 0.13, 220, 0.8, -0.05)
	c.note(softPulse, 0.16, 0.18, 146.83, 0.8, 0.05)
	c.note(instBass, 0, 0.13, 110, 0.5, 0)
	c.note(instBass, 0.16, 0.18, 73.42, 0.5, 0)
}

func drawUnlock(c *canvas) {
	c.noise(0, hiss{0.14, 1500, 7000, 0.04, 0.1}, 0.6, -0.35)
	c.noise(0.05, hiss{0.12, 2000, 8000, 0.03, 0.08}, 0.45, 0.35)
	for i, f := range []float64{392, 493.88, 587.33, 783.99, 987.77} {
		c.bell(0.12+0.09*float64(i), strike{f, 0.7, 0.75}, 0.5, -0.3+0.15*float64(i), glassPartials)
	}
}

func drawReady(c *canvas) {
	c.noise(0, hiss{0.08, 2000, 8000, 0.02, 0.06}, 0.35, 0)
	for i, f := range []float64{587.33, 783.99} {
		c.bell(0.05+0.09*float64(i), strike{f, 0.55, 0.6}, 0.45, -0.15+0.3*float64(i), glassPartials)
	}
}

func drawPage(c *canvas) {
	c.scratch(0, hiss{0.16, 1800, 7000, 0.03, 0.12}, 0.5, -0.2)
	c.scratch(0.11, hiss{0.2, 900, 4000, 0.04, 0.15}, 0.45, 0.2)
	c.noise(0.24, hiss{0.08, 300, 1200, 0.005, 0.05}, 0.25, 0)
}

func drawRollout(c *canvas) {
	for i, f := range []float64{196, 220, 246.94, 293.66, 329.63, 392, 440, 493.88} {
		pan := 0.45
		if i%2 == 0 {
			pan = -0.45
		}
		c.note(instLute, 0.1*float64(i), 0.18, f, 0.74, pan)
		c.note(instChip, 0.1*float64(i), 0.05, f/2, 0.12, 0)
	}
	c.note(instLute, 0.8, 0.3, 587.33, 0.63, 0)
}

func drawVictory(c *canvas) {
	lead := []float64{392, 493.88, 587.33, 783.99, 587.33, 783.99, 987.77}
	starts := []float64{0, 0.11, 0.22, 0.33, 0.52, 0.63, 0.74}
	for i, f := range lead {
		c.note(instHorn, starts[i], 0.1, f, 0.6, -0.1)
	}
	c.note(instHorn, 0.9, 0.9, 1174.66, 0.55, 0)
	c.note(instHorn, 0.9, 0.9, 783.99, 0.4, -0.3)
	c.note(instHorn, 0.9, 0.9, 587.33, 0.35, 0.3)
	c.note(instBass, 0.9, 0.9, 98, 0.6, 0)
	for i := 0; i < 6; i++ {
		c.hit('K', 0.5+0.07*float64(i), 0.4+0.08*float64(i), 0)
	}
	c.hit('K', 0.9, 1, 0)
	c.hit('J', 0.9, 0.6, 0.25)
	c.hit('J', 1.15, 0.4, -0.25)
}

func gleamNote(c *canvas, hz float64) {
	c.note(instHorn, 0, 0.3, hz, 0.65, 0)
	c.bell(0, strike{hz * 2, 0.3, 0.3}, 0.25, 0.1, glassPartials)
}

func drawGleamLast(c *canvas) {
	gleamNote(c, 587.33)
	c.note(instHorn, 0.05, 1.2, 783.99, 0.5, 0.2)
	c.note(instHorn, 0.05, 1.2, 987.77, 0.35, -0.2)
	c.note(instBass, 0.05, 1.0, 98, 0.5, 0)
	c.hit('J', 0.05, 0.5, 0.2)
}

func drawTremor(c *canvas) {
	c.put(0, rumble(c, 1.45), 1, 0)
	for i := 0; i < 14; i++ {
		t := 0.1 + 1.1*c.rng.unit()
		c.noise(t, hiss{0.03, 200, 1200, 0.001, 0.02}, 0.3+0.4*c.rng.unit(), 0.6*c.rng.signed())
	}
}

func rumble(c *canvas, dur float64) []float32 {
	n := c.at(dur)
	out := make([]float32, n)
	lp1, lp2 := newLowpass(240, c.rate), newLowpass(240, c.rate)
	attack, fall := c.at(0.15), c.at(0.35)
	var ph1, ph2, wob float64
	for i := range out {
		env := envelope(i, attack, n, 1)
		if i > fall {
			rel := 1 - float64(i-fall)/float64(n-fall)
			env *= rel * rel
		}
		f := 58 - 12*float64(i)/float64(n)
		ph1, ph2 = advance(ph1, f/float64(c.rate)), advance(ph2, 3*f/float64(c.rate))
		wob = advance(wob, 7/float64(c.rate))
		v := 4*lp2.low(lp1.low(c.rng.signed())) + 0.45*sine(ph1) + 0.35*sine(2*ph1) + 0.2*sine(ph2)
		out[i] = float32(v * env * (0.7 + 0.3*sine(wob)))
	}
	return out
}

func drawCalamity(c *canvas) {
	c.noise(0, hiss{0.02, 1000, 5000, 0.0005, 0.015}, 0.5, 0)
	c.bell(0, strike{146.83, 2.4, 2.45}, 1, -0.1, tollPartials)
	c.bell(0.004, strike{146.83 * 1.0015, 2.2, 2.4}, 0.5, 0.25, tollPartials)
}

func drawBoat(c *canvas) {
	c.note(softHorn, 0, 0.8, 146.83, 0.7, -0.15)
	c.note(softHorn, 0.05, 0.75, 220, 0.45, 0.15)
	c.noise(0.1, hiss{1.3, 200, 900, 0.35, 1.1}, 0.35, -0.3)
	c.noise(0.5, hiss{1.0, 250, 1000, 0.3, 0.8}, 0.3, 0.3)
}

func drawChronicle(c *canvas) {
	for i, t := range []float64{0, 0.08, 0.17, 0.25} {
		c.scratch(t, hiss{0.06, 2500, 7000, 0.005, 0.05}, 0.8, -0.2+0.12*float64(i))
	}
	c.bell(0.38, strike{880, 0.6, 0.65}, 0.55, -0.15, glassPartials)
	c.bell(0.46, strike{1318.51, 0.6, 0.6}, 0.45, 0.2, glassPartials)
}

func drawReveal(c *canvas) {
	c.bell(0, strike{783.99, 0.8, 0.85}, 0.6, -0.2, glassPartials)
	c.bell(0.004, strike{786.2, 0.8, 0.85}, 0.25, 0.2, glassPartials)
	c.bell(0.1, strike{1174.66, 0.7, 0.8}, 0.5, 0.2, glassPartials)
}

func drawDrums(c *canvas) {
	for i, t := range drumRoll {
		swell := float64(i) / float64(len(drumRoll))
		pan := 0.3
		if i%2 == 0 {
			pan = -0.3
		}
		c.hit('K', t, 0.35+0.5*swell, pan)
		c.hit('S', t+0.09-0.07*swell, 0.12+0.22*swell, -pan)
		c.hit('T', t, 0.25+0.3*swell, pan)
	}
	c.hit('B', 0.86, 0.6, 0)
	c.hit('K', 0.86, 1, -0.1)
	c.hit('S', 0.86, 0.5, 0.2)
	c.hit('J', 0.86, 0.45, 0.25)
}

func drawHorn(c *canvas) {
	c.note(warHorn, 0, 0.13, 146.83, 0.7, -0.1)
	c.note(warHorn, 0.2, 0.13, 146.83, 0.7, -0.1)
	c.note(warHorn, 0.42, 0.75, 220, 0.8, 0.05)
	c.note(warHorn, 0.44, 0.73, 146.83, 0.35, -0.25)
	c.note(instBass, 0.42, 0.75, 73.42, 0.25, 0)
}
