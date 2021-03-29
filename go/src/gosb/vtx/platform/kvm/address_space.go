package kvm

import (
	"gosb/vtx/platform/ring0/pagetables"
)

// addressSpace is a wrapper for PageTables.
type addressSpace struct {
	// machine is the underlying machine.
	machine *Machine

	// pageTables are for this particular address space.
	pageTables *pagetables.PageTables
}
