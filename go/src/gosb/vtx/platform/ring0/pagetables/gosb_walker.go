package pagetables

import (
	gc "gosb/commons"
	"unsafe"
)

/**
* @author: aghosn
*
* I did not like the page table implementation inside gvisor.
* As a result, I just wrote my own interface for them here using my previously
* implemented page walker.
 */

type Visitor struct {
	// Applies is true if we should apply the given visitor function to an entry
	// of level idx.
	Applies [4]bool

	// Create is true if the mapper should install a mapping for an absent entry.
	Create bool

	// Toggle is a used to say that we want to visit a page that is not valid
	Toogle bool

	// Alloc is an allocator function.
	// This can come from the allocator itself, and is used to either allocate
	// a new PTEs or to insert the address mapping.
	// Returns the GPA of the new PTE
	Alloc func(curr uintptr, lvl int) uintptr

	// Visit is a function called upon visiting an entry.
	Visit func(va uintptr, pte *PTE, lvl int)
}

// Map iterates over the provided range of address and applies the visitor.
func (p *PageTables) Map(start, length uintptr, v *Visitor) {
	end := start + length - 1
	p.pageWalk(p.root, start, end, _LVL_PML4, v)
}

// pageWalk is our homebrewed recursive pagewalker.
//
//TODO(aghosn) implement a go:nosplit version.
func (p *PageTables) pageWalk(root *PTEs, start, end uintptr, lvl int, v *Visitor) {
	if lvl < 0 || lvl > _LVL_PML4 {
		panic("wrong pageWalk level")
	}
	sfirst, send := PDX(start, lvl), PDX(end, lvl)
	baseVa := start & ^(PDADDR(lvl+1, 1) - 1)
	for i := sfirst; i <= send; i++ {
		curVa := baseVa + PDADDR(lvl, uintptr(i))
		entry := &root[i]
		if !entry.Valid() && v.Create {
			newPteGpa := v.Alloc(curVa, lvl)
			// Simply mark the page as present, rely on f to add the bits.
			entry.SetAddr(newPteGpa)
		}
		if entry.Valid() && v.Applies[lvl] {
			v.Visit(curVa, entry, lvl)
		} else if !entry.Valid() && v.Applies[lvl] && v.Toogle {
			v.Visit(curVa, entry, lvl)
		}
		nstart, nend := start, end
		if i != sfirst {
			nstart = curVa
		}
		if i != send {
			nend = curVa + PDADDR(lvl, 1) - 1
		}
		// Early stop to avoid a nested page.
		if lvl > 0 {
			p.pageWalk(p.Allocator.LookupPTEs(entry.Address()), nstart, nend, lvl-1, v)
		}
	}
}

// ConvertOpts converts the common protections into page table ones.
//
//go:nosplit
func ConvertOpts(prot uint8) uintptr {
	val := uintptr(accessed)
	if prot&gc.X_VAL == 0 {
		val |= executeDisable
	}
	if prot&gc.W_VAL != 0 {
		val |= writable
	}
	if prot&gc.R_VAL == gc.R_VAL {
		val |= present
	}
	if prot&gc.USER_VAL == gc.USER_VAL {
		val |= user
	} else {
		val &= ^uintptr(user)
	}
	return uintptr(val)
}

// CleanFlags removes runtime information to return only access rights
//
//go:nosplit
func CleanFlags(flags uintptr) uintptr {
	mask := uintptr(present | executeDisable | writable | user)
	return (flags & mask)
}

//go:nosplit
func (p *PageTables) FindMapping(addr uintptr) (uintptr, uintptr, uintptr) {
	addr = addr - (addr % gc.PageSize)
	s4, s3 := PDX(addr, _LVL_PML4), PDX(addr, _LVL_PDPTE)
	s2, s1 := PDX(addr, _LVL_PDE), PDX(addr, _LVL_PTE)
	pdpte := p.Allocator.LookupPTEs(p.root[s4].Address())
	gc.Check(pdpte != nil)
	pte := p.Allocator.LookupPTEs(pdpte[s3].Address())
	gc.Check(pte != nil)
	page := p.Allocator.LookupPTEs(pte[s2].Address())
	gc.Check(page != nil)
	return page[s1].Address(), page[s1].Flags(), uintptr(page[s1])
}

//go:nosplit
func (p *PageTables) FindPages(addr uintptr) (uintptr, uintptr, uintptr, uintptr) {
	addr = addr - (addr % gc.PageSize)
	s4, s3 := PDX(addr, _LVL_PML4), PDX(addr, _LVL_PDPTE)
	s2, _ := PDX(addr, _LVL_PDE), PDX(addr, _LVL_PTE)
	pdpte := p.Allocator.LookupPTEs(p.root[s4].Address())
	gc.Check(pdpte != nil)
	pte := p.Allocator.LookupPTEs(pdpte[s3].Address())
	gc.Check(pte != nil)
	page := p.Allocator.LookupPTEs(pte[s2].Address())
	gc.Check(page != nil)
	return uintptr(unsafe.Pointer(p.root)), uintptr(unsafe.Pointer(pdpte)),
		uintptr(unsafe.Pointer(pte)), uintptr(unsafe.Pointer(pte))
}

//go:nosplit
func (p *PageTables) IsMapped(addr uintptr) int {
	addr = addr - (addr % gc.PageSize)
	s4, s3 := PDX(addr, _LVL_PML4), PDX(addr, _LVL_PDPTE)
	s2, s1 := PDX(addr, _LVL_PDE), PDX(addr, _LVL_PTE)
	pdpte := p.Allocator.LookupPTEs(p.root[s4].Address())
	if !p.root[s4].Valid() || pdpte == nil {
		return -4
	}
	pte := p.Allocator.LookupPTEs(pdpte[s3].Address())
	if !pdpte[s3].Valid() || pte == nil {
		return -3
	}
	page := p.Allocator.LookupPTEs(pte[s2].Address())
	if !pte[s2].Valid() || page == nil {
		return -2
	}
	if page[s1].Valid() {
		return 1
	}
	if page[s1] == 0 {
		return -11
	}
	return -1
}

//go:nosplit
func (p *PageTables) Clear(addr uintptr) {
	addr = addr - (addr % gc.PageSize)
	s4, s3 := PDX(addr, _LVL_PML4), PDX(addr, _LVL_PDPTE)
	s2, s1 := PDX(addr, _LVL_PDE), PDX(addr, _LVL_PTE)
	pdpte := p.Allocator.LookupPTEs(p.root[s4].Address())
	pte := p.Allocator.LookupPTEs(pdpte[s3].Address())
	page := p.Allocator.LookupPTEs(pte[s2].Address())
	page[s1].SetFlags(0x0 | user)
}
