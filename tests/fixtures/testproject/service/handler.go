package service

import "main"

type Handler struct {
	name string
}

// Method with receiver calling TargetFunction
func (h *Handler) ProcessRequest(data int) {
	// This should NOT be detected as calling main.TargetFunction
	// because it's in a different package (would need import)
	result := processInternal(data)
	_ = result
}

func processInternal(x int) int {
	return x * 3
}

// This would be a caller if we had proper import
func WouldCallTarget() {
	// main.TargetFunction(50) // This would work with import
}