package kvm

// userMemoryRegion is a region of physical memory.
//
// This mirrors kvm_memory_region.
type userMemoryRegion struct {
	slot          uint32
	flags         uint32
	guestPhysAddr uint64
	memorySize    uint64
	userspaceAddr uint64
}

// runData is the run structure. This may be mapped for synchronous register
// access (although that doesn't appear to be supported by my kernel at least).
//
// This mirrors kvm_run.
type runData struct {
	requestInterruptWindow uint8
	_                      [7]uint8

	exitReason                 uint32
	readyForInterruptInjection uint8
	ifFlag                     uint8
	_                          [2]uint8

	cr8      uint64
	apicBase uint64

	// This is the union data for exits. Interpretation depends entirely on
	// the exitReason above (see vCPU code for more information).
	data [32]uint64
}
