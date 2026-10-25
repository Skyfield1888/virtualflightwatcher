package vatsim

// PilotCount returns the number of pilots currently connected (from the last refresh).
func PilotCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(pilotByCallsign)
}
