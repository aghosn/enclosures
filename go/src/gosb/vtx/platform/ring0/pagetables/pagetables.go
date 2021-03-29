package pagetables

import (
	"gosb/commons"
)

// PageTables is a set of page tables.
type PageTables struct {
	// Allocator is used to allocate nodes.
	Allocator Allocator

	// root is the pagetable root.
	root *PTEs

	// rootPhysical is the cached physical address of the root.
	//
	// This is saved only to prevent constant translation.
	rootPhysical uintptr
}

// New returns new PageTables.
func New(a Allocator) *PageTables {
	p := new(PageTables)
	p.Init(a)
	return p
}

// Init initializes a set of PageTables.
//
//go:nosplit
func (p *PageTables) Init(allocator Allocator) {
	p.Allocator = allocator
	p.root = p.Allocator.NewPTEs()
	p.rootPhysical = p.Allocator.PhysicalFor(p.root)
	commons.Check(p.rootPhysical != ^uintptr(0))
}
