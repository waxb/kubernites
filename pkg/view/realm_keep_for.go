package view

func KeepAllocatable(capacity ResourcePair) ResourcePair {
	return ResourcePair{CPU: max(0, capacity.CPU-ReeveShare.CpuMilli), Memory: max(0, capacity.Memory-ReeveShare.MemBytes)}
}
