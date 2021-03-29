package pagetables

const (
	pteShift = 12
	pmdShift = 21
	pudShift = 30
	pgdShift = 39
	fakShift = 48

	executeDisable = 1 << 63
	entriesPerPage = 512
)

// Page Table levels
const (
	_LVL_PTE   = 0
	_LVL_PDE   = 1
	_LVL_PDPTE = 2
	_LVL_PML4  = 3
	_LVL_FAKE  = 4
)

var (
	pdshift = [5]int{
		pteShift,
		pmdShift,
		pudShift,
		pgdShift,
		fakShift,
	}
)

// Page Table constants
const (
	_NPTBITS = 9 // log2(entriesPerPage)
	_PDXMASK = ((1 << _NPTBITS) - 1)
)

// PDX returns the index for the given address and level.
//go:nosplit
func PDX(addr uintptr, n int) int {
	return int(((addr) >> pdshift[n]) & _PDXMASK)
}

// PDADDR returns the address for the given level.
func PDADDR(n int, i uintptr) uintptr {
	return ((i) << pdshift[n])
}

// PTEs is a collection of entries.
type PTEs [entriesPerPage]PTE
