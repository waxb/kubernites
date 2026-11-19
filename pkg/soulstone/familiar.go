package soulstone

func Familiars() []Profile {
	out := []Profile{}
	for _, p := range Playable() {
		if p.Familiar {
			out = append(out, p)
		}
	}
	return out
}
