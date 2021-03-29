package procid

// Current returns the current system thread identifier.
//
// Precondition: This should only be called with the runtime OS thread locked.
func Current() uint64
