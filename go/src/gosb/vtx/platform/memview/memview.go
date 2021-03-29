package memview

import (
	"fmt"
	"gosb/commons"
	pg "gosb/vtx/platform/ring0/pagetables"
	"runtime"
	"unsafe"
)

type RegType = int

const (
	IMMUTABLE_REG  RegType = iota // Cannot be changed during the sandbox execution.
	HEAP_REG       RegType = iota // Can map/unmap, e.g., the heap
	EXTENSIBLE_REG RegType = iota // Can grow, add new parts.
)

// TODO replace with runtime information.
const (
	HEAP_START    = uint64(0xc000000000)
	HEAP_REG_SIZE = uint64(0x4000000)
	HEAP_BITMAP   = 256
)

// MemorySpan represents a contiguous memory region and the corresponding GPA.
type MemorySpan struct {
	commons.ListElem        // for extra Regions
	Start            uint64 // Start address of the region
	Size             uint64 // Size of the region
	Prot             uint8  // Default protection
	GPA              uint64 // Guest physical address
	Slot             uint32 // KVM memory slot
}

// MemoryRegion is a MemorySpan with a given type that determines whether
// its presence bits can be modified or not.
type MemoryRegion struct {
	commons.ListElem // Allows to put the Memory region inside a list
	Tpe              RegType
	Span             MemorySpan
	Bitmap           []uint64 // Presence bitmap
	BitmapInit       bool
	Owner            *AddressSpace // The owner AddressSpace
	View             commons.VMAreas
	finalized        bool
}

type AddressSpace struct {
	Regions       commons.List        // Memory regions
	FreeAllocator *FreeSpaceAllocator // Managed free memory spans < (1 << 39)

	PTEAllocator *PageTableAllocator // relies on FreeAllocator.
	Tables       *pg.PageTables      // Page table as in ring0

	NextSlot uint32 // EPT mappings slots.

	// Used for emergency runtime growth
	EMR [50]*MemoryRegion
}

var (
	Views map[commons.SandId]*AddressSpace
)

/*				AddressSpace methods				*/

// VMAToMemoryRegion creates a memory region from the provided VMA.
// It consumes the provided argument, i.e., it should not be in a list.
func (a *AddressSpace) VMAToMemoryRegion(vma *commons.VMArea) *MemoryRegion {
	commons.Check(vma != nil && vma.Addr < vma.Addr+vma.Size)
	commons.Check(vma.Prev == nil && vma.Next == nil)
	mem := &MemoryRegion{}
	mem.Span.Start = vma.Addr
	mem.Span.Size = vma.Size
	mem.Span.Prot = vma.Prot
	mem.Span.Slot = ^uint32(0)
	mem.Owner = a

	// Add the view
	mem.View.Init()
	mem.View.AddBack(vma.ToElem())

	// Allocate a physical address for this memory region.
	if mem.Span.Start+mem.Span.Size <= uint64(commons.Limit39bits) {
		mem.Span.GPA = mem.Span.Start
	} else {
		mem.Span.GPA = a.FreeAllocator.Malloc(mem.Span.Size)
	}

	// Find the category for this memory region.
	mem.Tpe = guessTpe(vma)

	// Extensible regions do not have a bitmap.
	if mem.Tpe == EXTENSIBLE_REG {
		goto apply
	}
	mem.AllocBitmap()
apply:
	mem.Map(vma.Addr, vma.Size, vma.Prot, true)
	return mem
}

func (a *AddressSpace) Copy(fresh bool) *AddressSpace {
	doppler := &AddressSpace{}
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		cpy := m.Copy()
		doppler.Regions.AddBack(cpy.ToElem())
		cpy.Owner = doppler
	}

	// Same free and pte allocators.
	doppler.FreeAllocator = a.FreeAllocator
	doppler.PTEAllocator = a.PTEAllocator

	// If fresh, page tables and free allocator are not copied over.
	if fresh {
		doppler.FreeAllocator = a.FreeAllocator.Copy()
		doppler.PTEAllocator = &PageTableAllocator{}
		doppler.PTEAllocator.Initialize(doppler.FreeAllocator)
	}
	return doppler
}

// ApplyDomain changes the view of this address space to the one specified by
// this domain.
func (a *AddressSpace) ApplyDomain(d *commons.SandboxMemory) {
	commons.Check(a.Tables == nil && a.PTEAllocator != nil)
	//commons.Check(ASTemplate.Tables == nil)
	// Initialize the root page table.
	a.Tables = pg.New(a.PTEAllocator)
	a.ApplyVMAs(d.Static.Copy())
}

// Applies the vmas to the address space.
// @warning destroys view.
func (a *AddressSpace) ApplyVMAs(view *commons.VMAreas) {
	for v := commons.ToVMA(view.First); v != nil; {
		next := commons.ToVMA(v.Next)
		view.Remove(v.ToElem())
		a.Assign(v)
		v = next
	}
	// Now finalize and apply the changes.
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		m.Finalize()
	}
}

// RegisterGrowth registers the callback to let KVM know about new mappings.
func (a *AddressSpace) RegisterGrowth(f func(uint64, uint64, uint64, uint32)) {
	a.PTEAllocator.Register = f
}

func (a *AddressSpace) Seal() {
	//commons.Check(a.PTEAllocator.Danger == false)
	// From now on, we cannot rely on dynamic allocations inside PageTableAllocator
	a.PTEAllocator.Danger = true
}

// Assign finds the memory region to which this vma belongs.
func (a *AddressSpace) Assign(vma *commons.VMArea) {
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		if m.ContainsRegion(vma.Addr, vma.Size) {
			m.Assign(vma)
			return
		}
	}
}

func (a *AddressSpace) Print() {
	for r := ToMemoryRegion(a.Regions.First); r != nil; r = ToMemoryRegion(r.Next) {
		r.Print()
	}
}

//go:nosplit
func (a *AddressSpace) ValidAddress(addr uint64) bool {
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		if addr >= m.Span.Start && addr < m.Span.Start+m.Span.Size {
			return m.ValidAddress(addr)
		}
	}
	return false
}

//go:nosplit
func (a *AddressSpace) FindVirtualForPhys(phys uint64) (uint64, bool) {
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		if phys >= m.Span.GPA && phys < m.Span.GPA+m.Span.Size {
			return m.Span.Start + (phys - m.Span.GPA), true
		}
	}
	// Maybe in the page table allocator then.
	ptea := a.PTEAllocator
	for a := ToArena(ptea.All.First); a != nil; a = ToArena(a.Next) {
		if a.GPA <= phys && a.GPA+uint64(ARENA_TOTAL_SIZE) > phys {
			return a.HVA + (phys - a.GPA), true
		}
	}
	return 0, false
}

//go:nosplit
func (a *AddressSpace) FindMemoryRegion(addr uint64) *MemoryRegion {
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		if addr >= m.Span.Start && addr < m.Span.Start+m.Span.Size {
			return m
		}
	}
	return nil
}

//go:nosplit
func (a *AddressSpace) HasRights(addr uint64, prot uint8) bool {
	prots := pg.ConvertOpts(prot)
	_, pte, _ := a.Tables.FindMapping(uintptr(addr))
	return (pte&prots == prots)
}

//go:nosplit
func (a *AddressSpace) Toggle(on bool, start, size uintptr, prot uint8) {
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		if m.ContainsRegion(uint64(start), uint64(size)) {
			m.Toggle(on, uint64(start), uint64(size), prot)
			return
		}
	}
	// We did not have a match, check if we should add something.
	if on {
		//TODO check if this is ever called.
		panic("It is called")
	}
}

// ToggleDynamic only cares about removing entries from the page tables.
//go:nosplit
func (a *AddressSpace) ToggleDyn(on bool, start, size uintptr, prot uint8) {
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		if m.ContainsRegion(uint64(start), uint64(size)) {
			m.ToggleDyn(on, uint64(start), uint64(size), prot)
			return
		}
	}
	// We did not have a match, check if we should add something.
	if on {
		//TODO check if this is ever called.
		panic("It is called")
	}
}

//go:nosplit
func (a *AddressSpace) ContainsRegion(start, size uintptr) bool {
	for m := ToMemoryRegion(a.Regions.First); m != nil; m = ToMemoryRegion(m.Next) {
		if m.ContainsRegion(uint64(start), uint64(size)) {
			return true
		}
	}
	return false
}

//go:nosplit
func (a *AddressSpace) Extend(heap bool, m *MemoryRegion, start, size uint64, prot uint8) {
	if m == nil {
		m = &MemoryRegion{}
	}
	m.Tpe = EXTENSIBLE_REG
	m.Span.Start, m.Span.Size, m.Span.Prot = start, size, prot
	m.Owner = a
	//	m.Span.Slot = a.NextSlot
	//	a.NextSlot++
	if heap {
		m.Tpe = HEAP_REG
		commons.Check(size <= HEAP_REG_SIZE)
		s := m.Coordinates(start)
		e := m.Coordinates(start + size - 1)
		for c := s; c <= e; c++ {
			m.Bitmap[idX(c)] |= uint64(1 << idY(c))
		}
	}
	if m.Span.Start+m.Span.Size <= uint64(commons.Limit39bits) {
		m.Span.GPA = m.Span.Start
	} else {
		m.Span.GPA = a.FreeAllocator.Malloc(m.Span.Size)
	}
	a.Regions.AddBack(m.ToElem())
	m.ApplyRange(start, size, prot)
	m.finalized = true
	// TODO does not call setEPT???
}

//go:nosplit
func (a *AddressSpace) Extend2(m, orig *MemoryRegion) {
	if m == nil {
		m = &MemoryRegion{}
	}
	m.Tpe = EXTENSIBLE_REG
	m.Span.Start, m.Span.Size, m.Span.Prot = orig.Span.Start, orig.Span.Size, orig.Span.Prot
	m.Owner = a
	m.Span.Slot = a.NextSlot
	m.Tpe = orig.Tpe
	m.Span.GPA = orig.Span.GPA
	a.NextSlot++
	if m.Tpe == HEAP_REG {
		copy(m.Bitmap, orig.Bitmap)
	}
	a.Regions.AddBack(m.ToElem())
	m.ApplyRange(m.Span.Start, m.Span.Size, m.Span.Prot)
	m.finalized = true
}

//go:nosplit
func (a *AddressSpace) ExtendRuntime(orig *MemoryRegion) {
	commons.Check(orig != nil)
	if a.ContainsRegion(uintptr(orig.Span.Start), uintptr(orig.Span.Size)) {
		panic("Already exists")
		return
	}
	m := a.AcquireEMR()
	a.Extend2(m, orig)
}

// TODO Maybe it's not pte allocator.
func (a *AddressSpace) MapArenas(seal bool) {
	if seal {
		a.PTEAllocator.Danger = true
	}
	a.PTEAllocator.All.Foreach(func(e *commons.ListElem) {
		arena := ToArena(e)
		a.DefaultMap(uintptr(arena.HVA), ARENA_TOTAL_SIZE, uintptr(arena.GPA))
	})
	a.PTEAllocator.All.Foreach(func(e *commons.ListElem) {
		arena := ToArena(e)
		commons.Check(a.Tables.IsMapped(uintptr(arena.HVA)) == 1)
	})
	GodAS.PTEAllocator.All.Foreach(func(e *commons.ListElem) {
		arena := ToArena(e)
		commons.Check(GodAS.Tables.IsMapped(uintptr(arena.HVA)) == 1)
	})
}

func (a *AddressSpace) DefaultMap(start, size, gpa uintptr) {
	flags := pg.ConvertOpts(commons.R_VAL | commons.USER_VAL | commons.W_VAL)
	deflags := pg.ConvertOpts(commons.D_VAL)
	alloc := func(addr uintptr, lvl int) uintptr {
		if lvl > 0 {
			_, addr := a.PTEAllocator.NewPTEs2()
			return uintptr(addr)
		}
		gpa := (addr - uintptr(start)) + uintptr(gpa)
		return gpa
	}
	visit := func(va uintptr, pte *pg.PTE, lvl int) {
		if lvl == 0 {
			pte.SetFlags(flags)
			return
		}
		pte.SetFlags(deflags)
	}
	visitor := pg.Visitor{
		Applies: [4]bool{true, true, true, true},
		Create:  true,
		Alloc:   alloc,
		Visit:   visit,
	}
	a.Tables.Map(start, size, &visitor)
}

//go:nosplit
func (a *AddressSpace) AcquireEMR() *MemoryRegion {
	for i := range a.EMR {
		if a.EMR[i] != nil {
			result := a.EMR[i]
			a.EMR[i] = nil
			return result
		}
	}
	panic("Unable to acquire a new memory region :(")
	return nil
}

//go:nosplit
func (a *AddressSpace) Replenish() {
	for i := range a.EMR {
		if a.EMR[i] == nil {
			a.EMR[i] = &MemoryRegion{}
			if a.EMR[i] == nil {
				panic("Allocation failed??")
			}
			a.EMR[i].Bitmap = make([]uint64, HEAP_BITMAP)
		}
	}
}

/*				MemoryRegion methods				*/

//go:nosplit
func ToMemoryRegion(e *commons.ListElem) *MemoryRegion {
	return (*MemoryRegion)(unsafe.Pointer(e))
}

// AllocBitmap allocates the slice for the given memory region.
// We assume that Span.Start and Span.Size have been allocated.
// This should be called only once.
func (m *MemoryRegion) AllocBitmap() {
	commons.Check(m.Bitmap == nil)
	nbPages := m.Span.Size / uint64(commons.PageSize)
	if m.Span.Size%uint64(commons.PageSize) != 0 {
		nbPages += 1
	}
	nbEntries := nbPages / 64
	if nbPages%64 != 0 {
		nbEntries += 1
	}
	m.Bitmap = make([]uint64, nbEntries)
}

// Assign just registers the given vma as belonging to this region.
func (m *MemoryRegion) Assign(vma *commons.VMArea) {
	commons.Check(m.Span.Start <= vma.Addr && m.Span.Start+m.Span.Size >= vma.Addr+vma.Size)
	m.View.AddBack(vma.ToElem())
}

//go:nosplit
func (m *MemoryRegion) Map(start, size uint64, prot uint8, apply bool) {
	s := m.Coordinates(start)
	e := m.Coordinates(start + size - 1)
	if m.Tpe == EXTENSIBLE_REG /*|| m.Tpe == HEAP_REG*/ {
		// The entire bitmap is at one
		goto skip
	}
	commons.Check(m.Bitmap != nil)
	// toggle bits in the bitmap
	for c := s; c <= e && !m.BitmapInit; c++ {
		m.Bitmap[idX(c)] |= uint64(1 << idY(c))
	}
	m.BitmapInit = true
skip:
	if !apply {
		return
	}
	m.ApplyRange(start, size, prot)
}

//go:nosplit
func (m *MemoryRegion) ApplyRange(start, size uint64, prot uint8) {
	eflags := pg.ConvertOpts(m.Span.Prot & prot)
	deflags := pg.ConvertOpts(commons.D_VAL)
	alloc := func(addr uintptr, lvl int) uintptr {
		if lvl > 0 {
			_, addr := m.Owner.PTEAllocator.NewPTEs2()
			return uintptr(addr)
		}

		// This is a PTE entry, we map the physical page.
		gpa := (addr - uintptr(m.Span.Start)) + uintptr(m.Span.GPA)
		return gpa
	}
	visit := func(va uintptr, pte *pg.PTE, lvl int) {
		if lvl == 0 {
			pte.SetFlags(eflags)
			if m.Tpe == HEAP_REG {
				commons.Check(m.Bitmap != nil)
				s := m.Coordinates(uint64(va))
				b := m.Bitmap[idX(s)] & uint64(1<<idY(s))
				if b == 0 {
					pte.Unmap()
				}
			}
			return
		}
		pte.SetFlags(deflags)
	}
	visitor := pg.Visitor{
		Applies: [4]bool{true, true, true, true},
		Create:  true,
		Alloc:   alloc,
		Visit:   visit,
	}
	m.Owner.Tables.Map(uintptr(start), uintptr(size), &visitor)
}

// Finalize applies the memory region view to the page tables.
func (m *MemoryRegion) Finalize() {
	switch m.Tpe {
	case IMMUTABLE_REG:
		// This is the text, data, and rodata.
		// We go through each of them and mapp them.
		for v := commons.ToVMA(m.View.First); v != nil; v = commons.ToVMA(v.Next) {
			m.Map(v.Addr, v.Size, v.Prot, true)
		}
		//fallthrough
	case HEAP_REG:
		fallthrough
	default:
		m.Map(m.Span.Start, m.Span.Size, m.Span.Prot, true)
	}
	m.finalized = true
}

func (m *MemoryRegion) Print() {
	switch m.Tpe {
	case IMMUTABLE_REG:
		for v := commons.ToVMA(m.View.First); v != nil; v = commons.ToVMA(v.Next) {
			fmt.Printf("%x -- %x [%x] (%x)\n", v.Addr, v.Addr+v.Size, v.Size, v.Prot)
		}
	default:
		fmt.Printf("%x -- %x [%x] (%x)\n", m.Span.Start, m.Span.Start+m.Span.Size, m.Span.Size, m.Span.Prot)
	}
}

//go:nosplit
func (m *MemoryRegion) Unmap(start, size uintptr, apply bool) {
	s := m.Coordinates(uint64(start))
	e := m.Coordinates(uint64(start + size - 1))
	if m.Tpe == EXTENSIBLE_REG {
		panic("Unmap cannot be called on extensible region")
	}
	if m.Tpe == HEAP_REG {
		goto skip
	}
	// toggle bits in the bitmap
	for c := s; s <= e; c++ {
		m.Bitmap[idX(c)] &= ^(uint64(1 << idY(c)))
	}
skip:
	if apply {
		//TODO implement page tables
		panic("Not implemented yet")
	}
}

//go:nosplit
func (m *MemoryRegion) Coordinates(addr uint64) int {
	addr = addr - m.Span.Start
	page := (addr - (addr % commons.PageSize)) / commons.PageSize
	return int(page)
}

// Transpose takes an index and changes it into an address within the span.
//go:nosplit
func (m *MemoryRegion) Transpose(idx int) uint64 {
	base := uint64(idX(idx) * (64 * commons.PageSize))
	off := uint64(idY(idx) * commons.PageSize)
	addr := m.Span.Start + base + off
	commons.Check(addr < m.Span.Start+m.Span.Size)
	return addr
}

//go:nosplit
func (m *MemoryRegion) ToElem() *commons.ListElem {
	return (*commons.ListElem)(unsafe.Pointer(m))
}

func (m *MemoryRegion) Copy() *MemoryRegion {
	doppler := &MemoryRegion{}
	doppler.Tpe = m.Tpe
	doppler.Span = m.Span
	if m.Bitmap != nil {
		doppler.Bitmap = make([]uint64, len(m.Bitmap))
		for i := range m.Bitmap {
			doppler.Bitmap[i] = m.Bitmap[i]
		}
		doppler.BitmapInit = true
	}
	return doppler
}

// ValidAddress
//
//go:nosplit
func (m *MemoryRegion) ValidAddress(addr uint64) bool {
	if addr < m.Span.Start || addr >= m.Span.Start+m.Span.Size {
		return false
	}
	if m.Tpe == EXTENSIBLE_REG || len(m.Bitmap) == 0 || !m.finalized {
		return true
	}
	if m.Tpe == IMMUTABLE_REG {
		c := m.Coordinates(addr)
		return (m.Bitmap[idX(c)]&uint64(1<<idY(c)) != 0)
	}

	// At that point, we're the heap and need to look into page tables.
	return true
}

//go:nosplit
func (m *MemoryRegion) ContainsRegion(addr, size uint64) bool {
	// Not completely correct but oh well right now.
	return m.ValidAddress(addr) && m.ValidAddress(addr+size-1)
}

//go:nosplit
func (m *MemoryRegion) Toggle(on bool, start, size uint64, prot uint8) {
	if m.Tpe == EXTENSIBLE_REG {
		// Should not happen
		panic("You want to map something that is mapped?")
	} else if m.Tpe == IMMUTABLE_REG {
		panic("Trying to change immutable region.")
	} else if m.Tpe != HEAP_REG {
		panic("What are you then?!!")
	}
	// Update the bitmap
	commons.Check(m.Bitmap != nil)
	s := m.Coordinates(start)
	e := m.Coordinates(start + size - 1)
	for c := s; c <= e; c++ {
		if on {
			m.Bitmap[idX(c)] |= uint64(1 << idY(c))
		} else {
			m.Bitmap[idX(c)] &= ^(uint64(1 << idY(c)))
		}
	}
	deflags := pg.ConvertOpts(prot)
	// Now apply to pagetable.
	visit := func(va uintptr, pte *pg.PTE, lvl int) {
		if lvl != 0 {
			return
		}
		if on {
			// @aghosn the new prots can only be a subset of the default ones
			commons.Check(prot <= m.Span.Prot)
			pte.SetFlags(deflags)
			pte.Map()
			flags := pte.Flags()
			commons.Check(pg.CleanFlags(flags) == pg.CleanFlags(deflags))
		} else {
			pte.Unmap()
		}
	}
	visitor := pg.Visitor{
		Applies: [4]bool{true, false, false, false},
		Create:  false,
		Toogle:  true,
		Alloc:   nil,
		Visit:   visit,
	}
	m.Owner.Tables.Map(uintptr(start), uintptr(size), &visitor)
}

//go:nosplit
func (m *MemoryRegion) ToggleDyn(on bool, start, size uint64, prot uint8) {
	deflags := pg.ConvertOpts(prot)
	// Now apply to pagetable.
	visit := func(va uintptr, pte *pg.PTE, lvl int) {
		if lvl != 0 {
			return
		}
		if on && prot != commons.UNMAP_VAL {
			// @aghosn the new prots can only be a subset of the default ones
			//commons.Check(prot <= m.Span.Prot)
			pte.SetFlags(deflags)
			pte.Map()
			flags := pte.Flags()
			commons.Check(pg.CleanFlags(flags) == pg.CleanFlags(deflags))
		} else {
			pte.Unmap()
		}
	}
	visitor := pg.Visitor{
		Applies: [4]bool{true, false, false, false},
		Create:  false,
		Toogle:  true,
		Alloc:   nil,
		Visit:   visit,
	}
	m.Owner.Tables.Map(uintptr(start), uintptr(size), &visitor)
}

/*				Span methods				*/

//go:nosplit
func ToMemorySpan(e *commons.ListElem) *MemorySpan {
	return (*MemorySpan)(unsafe.Pointer(e))
}

func (s *MemorySpan) Copy() *MemorySpan {
	doppler := &MemorySpan{}
	*doppler = *s
	doppler.Prev = nil
	doppler.Next = nil
	return doppler
}

//go:nosplit
func (s *MemorySpan) ToElem() *commons.ListElem {
	return (*commons.ListElem)(unsafe.Pointer(s))
}

/*				Helper functions				*/

//go:nosplit
func guessTpe(head *commons.VMArea) RegType {
	isexec := head.Prot&commons.X_VAL == commons.X_VAL
	isread := head.Prot&commons.R_VAL == commons.R_VAL
	iswrit := head.Prot&commons.W_VAL == commons.W_VAL
	isheap := runtime.IsThisTheHeap(uintptr(head.Addr))
	ismeta := !isheap && head.Addr > HEAP_START
	ischeap := CHeap != 0 && CHeap == head.Addr

	// executable and readonly sections do not change.
	if !ismeta && (isexec || (isread && !iswrit)) && !ischeap {
		return IMMUTABLE_REG
	}
	if isheap {
		return HEAP_REG
	}
	if ismeta || ischeap {
		return EXTENSIBLE_REG
	}
	// Probably just data, so it is immutable.
	return IMMUTABLE_REG
}

//go:nosplit
func idX(idx int) int {
	return int(idx / 64)
}

//go:nosplit
func idY(idx int) int {
	return int(idx % 64)
}

//go:nosplit
func bitmapSize(length int) int {
	return length * 64
}
