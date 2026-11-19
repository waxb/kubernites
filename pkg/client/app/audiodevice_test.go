package app

import "testing"

func TestALinuxMachineWithNoSoundServerOrCardStartsSilent(t *testing.T) {
	cases := []struct {
		name  string
		goos  string
		env   map[string]string
		files map[string]bool
		pcms  []string
		want  bool
	}{
		{name: "nothing at all", goos: "linux", want: false},
		{name: "runtime dir without a pulse socket", goos: "linux", env: map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}, want: false},
		{name: "pulse or pipewire socket", goos: "linux", env: map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}, files: map[string]bool{"/run/user/1000/pulse/native": true}, want: true},
		{name: "explicit pulse server", goos: "linux", env: map[string]string{"PULSE_SERVER": "tcp:localhost"}, want: true},
		{name: "an ALSA playback card", goos: "linux", pcms: []string{"/dev/snd/pcmC0D0p"}, want: true},
		{name: "macOS always has Core Audio", goos: "darwin", want: true},
		{name: "Windows is left to the driver", goos: "windows", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := audioProbe{
				goos:   c.goos,
				getenv: func(k string) string { return c.env[k] },
				exists: func(path string) bool { return c.files[path] },
				glob:   func(string) ([]string, error) { return c.pcms, nil },
			}
			if got := p.hasDevice(); got != c.want {
				t.Fatalf("hasDevice() = %v, want %v", got, c.want)
			}
		})
	}
}
