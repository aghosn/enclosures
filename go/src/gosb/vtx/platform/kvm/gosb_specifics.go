package kvm

import (
	"gosb/commons"
	mv "gosb/vtx/platform/memview"
	"syscall"
)

//go:nosplit
func GetFs(addr *uint64)

//go:nosplit
func GetFs2() uint64

// SetAllEPTSlots registers the different regions with KVM for HVA -> GPA mappings.
func (m *Machine) SetAllEPTSlots() {
	// First, we register the pages used for page tables.
	m.MemView.PTEAllocator.All.Foreach(func(e *commons.ListElem) {
		arena := mv.ToArena(e)
		commons.Check(arena.Dirty)
		var err syscall.Errno
		arena.Slot, err = m.setEPTRegion(&m.MemView.NextSlot, arena.GPA, uint64(mv.ARENA_TOTAL_SIZE), arena.HVA, 1)
		if err != 0 {
			panic("Error mapping slot")
		}
		arena.Dirty = false
		m.MemView.PTEAllocator.Dirties -= 1
		commons.Check(m.MemView.PTEAllocator.Dirties >= 0)
	})
	commons.Check(m.MemView.PTEAllocator.Dirties == 0)

	// Second, map the memory regions.
	m.MemView.Regions.Foreach(func(e *commons.ListElem) {
		mem := mv.ToMemoryRegion(e)
		span := mem.Span
		var err syscall.Errno
		span.Slot, err = m.setEPTRegion(&m.MemView.NextSlot, span.GPA, span.Size, span.Start, 1)
		if err != 0 {
			panic("Error mapping slot")
		}
	})
}

// UpdateEPTSlots registers new EPT slots.
func (m *Machine) UpdateEPTSlots(f func(start, size, gpa uintptr)) {
	ptea := m.MemView.PTEAllocator
	commons.Check(ptea.Dirties >= 0)
	var err syscall.Errno
	for a := mv.ToArena(ptea.All.First); a != nil; a = mv.ToArena(a.Next) {
		if !a.Dirty {
			continue
		}
		a.Slot, err = m.setEPTRegion(&m.MemView.NextSlot, a.GPA, uint64(mv.ARENA_TOTAL_SIZE), a.HVA, 1)
		if err != 0 {
			panic("Error mapping slot")
		}
		if f != nil {
			// This callback should map for PTEs in all the views
			f(uintptr(a.HVA), mv.ARENA_TOTAL_SIZE, uintptr(a.GPA))
		}
		a.Dirty = false
		ptea.Dirties -= 1
	}
	commons.Check(ptea.Dirties == 0)
}
