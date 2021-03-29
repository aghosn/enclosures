package globals

/**
* @author: aghosn
*
* This file holds the global data that we will use in every other packages.
* We have to isolate them to allow multi-package access to them.
 */
import (
	"debug/elf"
	"errors"
	"fmt"
	c "gosb/commons"
	"strings"
	"sync/atomic"
)

const (
	BackendPrefix = "gosb"

	// Non-mappable sandbox.
	TrustedSandbox  = "-1"
	TrustedPackages = "non-bloat"
)

var (
	// For debugging for the moment
	IsDynamic bool = false
	// Symbols
	Symbols   []elf.Symbol
	NameToSym map[string]*elf.Symbol

	// Packages
	AllPackages     []*c.Package
	BackendPackages []*c.Package
	NextPkgId       uint32

	// PC to package sorted list
	PcToPkg []*c.Package

	// VMareas
	CommonVMAs   *c.VMAreas
	TrustedSpace *c.VMAreas

	// Maps
	NameToPkg map[string]*c.Package
	IdToPkg   map[int]*c.Package
	NameToId  map[string]int
	RtIds     map[int]int
	RtKeys    map[int][]int

	// Sandboxes
	Configurations []*c.SandboxDomain
	SandboxFuncs   map[c.SandId]*c.VMArea
	Sandboxes      map[c.SandId]*c.SandboxMemory

	// Pristine Information
	IsPristine map[c.SandId]bool

	// Dependencies
	PkgDeps map[int][]c.SandId
)

type IdFunc func() c.SandId

// For dynamic usage
var (
	// Callback to get the current id.
	DynGetId IdFunc = nil

	// Callback to get the previous id
	DynGetPrevId IdFunc = nil
)

// PristineId generates a new pristine id for the sandbox.
func PristineId(id string) (string, int) {
	pid := atomic.AddUint32(&NextPkgId, 1)
	return fmt.Sprintf("p:%v:%v", pid, id), int(pid)
}

func DynFindId(name string) (int, error) {
	// Fast path
	if id, ok := NameToId[name]; ok {
		return id, nil
	}

	// Slow path
	for k, v := range NameToId {
		if strings.HasSuffix(k, fmt.Sprintf(".%s", name)) {
			return v, nil
		}
	}

	// This is an exception
	if name == "os.path" {
		return DynFindId("posixpath")
	}
	return -1, errors.New(fmt.Sprintf("Unable to find an id for %s", name))
}
