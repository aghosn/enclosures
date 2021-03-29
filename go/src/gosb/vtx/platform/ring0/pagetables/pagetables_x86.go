package pagetables

import (
	"sync/atomic"
	"unsafe"
)

// CR3 returns the CR3 value for these tables.
//
// This may be called in interrupt contexts. A PCID of zero always implies a
// flush and should be passed when PCIDs are not enabled. See pcids_x86.go for
// more information.
//
//go:nosplit
func (p *PageTables) CR3(noFlush bool, pcid uint16) uint64 {
	// Bit 63 is set to avoid flushing the PCID (per SDM 4.10.4.1).
	const noFlushBit uint64 = 0x8000000000000000
	if noFlush && pcid != 0 {
		return noFlushBit | uint64(p.rootPhysical) | uint64(pcid)
	}
	return uint64(p.rootPhysical) | uint64(pcid)
}

// Bits in page table entries.
const (
	present      = 0x001
	writable     = 0x002
	user         = 0x004
	writeThrough = 0x008
	cacheDisable = 0x010
	accessed     = 0x020
	dirty        = 0x040
	super        = 0x080
	global       = 0x100
	optionMask   = executeDisable | 0xfff
)

// PTE is a page table entry.
type PTE uintptr

// Clear clears this PTE, including super page information.
//
//go:nosplit
func (p *PTE) Clear() {
	atomic.StoreUintptr((*uintptr)(p), 0)
}

// Valid returns true iff this entry is valid.
//
//go:nosplit
func (p *PTE) Valid() bool {
	return atomic.LoadUintptr((*uintptr)(p))&present != 0
}

// SetSuper sets this page as a super page.
//
// The page must not be valid or a panic will result.
//
//go:nosplit
func (p *PTE) SetSuper() {
	if p.Valid() {
		// This is not allowed.
		panic("SetSuper called on valid page!")
	}
	atomic.StoreUintptr((*uintptr)(p), super)
}

// IsSuper returns true iff this page is a super page.
//
//go:nosplit
func (p *PTE) IsSuper() bool {
	return atomic.LoadUintptr((*uintptr)(p))&super != 0
}

// Address extracts the address. This should only be used if Valid returns true.
//
//go:nosplit
func (p *PTE) Address() uintptr {
	return atomic.LoadUintptr((*uintptr)(p)) &^ optionMask
}

// Flags extracs the entry's flags.
//
//go:nosplit
func (p *PTE) Flags() uintptr {
	return atomic.LoadUintptr((*uintptr)(p)) & optionMask
}

// SetAddr atomically sets the address for this page table entry.
// Carefull it removes rights!
//
//go:nosplit
func (p *PTE) SetAddr(addr uintptr) {
	v := (addr &^ optionMask) | present | accessed
	atomic.StoreUintptr((*uintptr)(p), v)
}

//go:nosplit
func (p *PTE) SetFlags(flags uintptr) {
	v := p.Address()
	v |= flags | accessed
	atomic.StoreUintptr((*uintptr)(p), v)
}

func (p *PTE) AddressAsPTES() *PTEs {
	addr := p.Address()
	return (*PTEs)(unsafe.Pointer(addr))
}

//go:nosplit
func (p *PTE) Unmap() {
	flag := p.Flags()
	flag = ((flag >> 1) << 1)
	p.SetFlags(flag)
}

//go:nosplit
func (p *PTE) Map() {
	flag := p.Flags() | present
	p.SetFlags(flag)
}
