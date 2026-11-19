package soulstone

const ArchivistImage = "archivist:1.0"

func init() {
	p := systemProfile(ArchivistImage, "Archivists, class of 1.0", "Archivist", "archivist", 250, 512)
	p.PullSeconds, p.StartSeconds, p.MTBFSeconds = 4, 2, 0
	p.StartLines = []string{"opening the books of seals", "listening for seals on :8080"}
	p.CrashLine = "fatal: the quill snapped mid-seal (exit status 1)"
	index[p.Image] = withParts(p)
}
