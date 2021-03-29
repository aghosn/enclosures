package kvm

// handleBluepillFault handles a physical fault.
//
// The corresponding virtual address is returned. This may throw on error.
//
//go:nosplit
/*
func handleBluepillFault(m *Machine, physical uintptr, phyRegions []physicalRegion, flags uint32) (uintptr, bool) {
	// Paging fault: we need to map the underlying physical pages for this
	// fault. This all has to be done in this function because we're in a
	// signal handler context. (We can't call any functions that might
	// split the stack.)
	virtualStart, physicalStart, length, ok := calculateBluepillFault(physical, phyRegions)
	if !ok {
		return 0, false
	}

	// Set the KVM slot.
	//
	// First, we need to acquire the exclusive right to set a slot.  See
	// machine.nextSlot for information about the protocol.
	slot := atomic.SwapUint32(&m.nextSlot, ^uint32(0))
	for slot == ^uint32(0) {
		yield() // Race with another call.
		slot = atomic.SwapUint32(&m.nextSlot, ^uint32(0))
	}
	errno := m.setMemoryRegion(int(slot), physicalStart, length, virtualStart, flags)
	if errno == 0 {
		// Successfully added region; we can increment nextSlot and
		// allow another set to proceed here.
		atomic.StoreUint32(&m.nextSlot, slot+1)
		return virtualStart + (physical - physicalStart), true
	}

	// Release our slot (still available).
	atomic.StoreUint32(&m.nextSlot, slot)

	switch errno {
	case syscall.EEXIST:
		// The region already exists. It's possible that we raced with
		// another vCPU here. We just revert nextSlot and return true,
		// because this must have been satisfied by some other vCPU.
		return virtualStart + (physical - physicalStart), true
	case syscall.EINVAL:
		throw("set memory region failed; out of slots")
	case syscall.ENOMEM:
		throw("set memory region failed: out of memory")
	case syscall.EFAULT:
		throw("set memory region failed: invalid physical range")
	default:
		throw("set memory region failed: unknown reason")
	}

	panic("unreachable")
}*/
