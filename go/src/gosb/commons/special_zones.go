package commons

const (
	// reservedMemory is a chunk of physical memory reserved starting at
	// physical address zero. There are some special pages in this region,
	// so we just call the whole thing off.
	ReservedMemory = 0x100000000
)

const (
	TrustedPkgName = "non-bloat"
	StmpPkgName    = "shared-stmp"
)

var (
	SymToFix = map[string]bool{
		"type.*":              true,
		"typerel.*":           true,
		"go.string.*":         true,
		"go.func.*":           true,
		"runtime.gcbits.*":    true,
		"go.funcrel.*":        true,
		"runtime.itablink":    true,
		"runtime.findfunctab": true,
	}

	ExtraDependencies = []string{
		"gosb",
	}
)
