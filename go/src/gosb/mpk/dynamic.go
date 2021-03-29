package mpk

import (
	"gosb/commons"
	"gosb/globals"
	"sync"
)

var (
	ionce sync.Once
	// Since python is single threaded, we can hold a mirror
	// of the pkru value inside a global variable for debugging.
	pkruMirror PKRU = AllRightsPKRU
)

func DInit() {
	// Let's delay the initialization to the first prolog
	//WritePKRU(AllRightsPKRU)
}

func internalInit() {
	ionce.Do(func() {
		sbPKRU = make(map[commons.SandId]PKRU)
		pkgKeys = make(map[int]Pkey)
	})
}

func DProlog(id commons.SandId) {
	internalInit()
	commons.Check(sbPKRU != nil)
	pkru, ok := sbPKRU[id]
	if ok {
		dprolog(pkru)
		return
	}
	// Check the sandbox does exists
	_, ok = globals.Sandboxes[id]
	commons.Check(ok)
	// First time we see this sandbox, compute the pkru.
	globals.AggregatePackages()
	//TODO compute key allocation
	//fmt.Println("Here is the RTIds ", globals.RtIds)
	//fmt.Println("The RtKeys ", globals.RtKeys)
	dprolog(pkru)
}

func dprolog(p PKRU) {
	//WritePKRU(p)
	pkruMirror = p
}

func DEpilog(id commons.SandId) {
	commons.Check(globals.DynGetPrevId != nil)
	// Disallow nesting for the moment
	commons.Check(globals.DynGetPrevId() == "GOD")
	//WritePKRU(AllRightsPKRU)
	pkruMirror = AllRightsPKRU
}

func DTransfer(oldid, newid int, start, size uintptr) {
	panic("Should not be called")
}

func DRuntimeGrowth(isheap bool, id int, start, size uintptr) {
	// @aghosn, nothing to do, it's going to be tagged 0 by default.
}

/** Helper functions **/
func attributeKeys() {
	// TODO check the total number of required keys.
	// Allocate them if needed
	// Compare with what packages are tagged with
}
