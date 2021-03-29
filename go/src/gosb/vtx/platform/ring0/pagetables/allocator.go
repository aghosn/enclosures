package pagetables

// Allocator is used to allocate and map PTEs.
//
// Note that allocators may be called concurrently.
type Allocator interface {
	// NewPTEs returns a new set of PTEs and their physical address.
	NewPTEs() *PTEs

	// PhysicalFor gives the physical address for a set of PTEs.
	PhysicalFor(ptes *PTEs) uintptr

	// LookupPTEs looks up PTEs by physical address.
	LookupPTEs(physical uintptr) *PTEs

	// FreePTEs marks a set of PTEs a freed, although they may not be available
	// for use again until Recycle is called, below.
	FreePTEs(ptes *PTEs)
}
