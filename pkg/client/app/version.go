package app

const Version = "0.14.5"

var Commit = ""

func VersionLabel() string {
	if Commit == "" {
		return "v" + Version
	}
	return "v" + Version + " (" + Commit + ")"
}
