package view

func SeatsOnKeep(capacity, asks, takes ResourcePair) int {
	if asks.CPU <= 0 && asks.Memory <= 0 {
		return 0
	}
	alloc := KeepAllocatable(capacity)
	return min(RoomFor(alloc, asks), RoomFor(alloc, takes))
}

func KeepTakes(asks, takes ResourcePair) ResourcePair {
	return ResourcePair{CPU: asks.CPU, Memory: max(asks.Memory, takes.Memory)}
}
