package main

/**
* This file defines the API exposed by the dynamic library that allows to call
* into the gosb backend library.
* Here, we simply implement forwarders that translate C types into Go types.
 */

import (
	"C"
	"fmt"
	"gosb"
	"gosb/backend"
	"gosb/benchmark"
	"gosb/commons"
	"gosb/globals"
	"gosb/vtx"
	"os"
	"unsafe"
)

//TODO remove afterwards, for debugging
var (
	returnedFromP bool = false
)

var (
	inited         = false
	knownSandboxes map[string]int
	StackIds       []commons.SandId = nil // Stack of sandbox ids we entered
	references     map[uintptr]int
	canSwitch      = false
)

//export SB_Initialize
func SB_Initialize() {
	str := os.Getenv(benchmark.BE_FLAG)
	b := backend.BACKEND_SIZE
	switch str {
	case "":
		fallthrough
	case "SIM":
		b = backend.SIM_BACKEND
	case "DVTX":
		b = backend.DVTX_BACKEND
	case "DMPK":
		b = backend.DMPK_BACKEND
	}
	commons.Check(b != backend.BACKEND_SIZE)
	globals.DynGetId = getid
	globals.DynGetPrevId = getPrevId
	gosb.DynInitialize(b)
	inited = true
	if b == backend.DVTX_BACKEND {
		canSwitch = true
	}
	references = make(map[uintptr]int)
	knownSandboxes = make(map[string]int)
}

//export SB_Prolog
func SB_Prolog(id *C.char) {
	commons.Check(inited)
	str := C.GoString(id)
	StackIds = append(StackIds, str)
	returnedFromP = false
	gosb.DynProlog(str)
	returnedFromP = true
}

var (
	returnedFromEpi = false
)

//export SB_Epilog
func SB_Epilog(id *C.char) {
	commons.Check(inited)
	str := C.GoString(id)
	commons.Check(StackIds != nil && len(StackIds) >= 1)
	pid := StackIds[len(StackIds)-1]
	commons.Check(pid == str)
	gosb.DynEpilog(str)
	StackIds = StackIds[:len(StackIds)-1]
}

//export SB_RegisterDependency
func SB_RegisterDependency(current, dependency *C.char) {
	c, d := C.GoString(current), C.GoString(dependency)
	// This happens
	if _, ok := commons.PythonSynthetic[d]; ok {
		return
	}
	/*if d == "pooch" || d == "nt" || d == "pooch.utils" || d == "msvcrt" {
		return
	}*/
	gosb.DynAddDependency(c, d)
}

//export SB_RegisterPackageId
func SB_RegisterPackageId(name *C.char, id C.int) {
	n := C.GoString(name)
	if _, ok := commons.PythonSynthetic[n]; ok {
		panic("This package should not exist!")
		return
	}
	gosb.DynRegisterId(n, int(id))
}

//export SB_RegisterSandbox
func SB_RegisterSandbox(pid C.int, id, mem, sys *C.char) {
	i, m, s := C.GoString(id), C.GoString(mem), C.GoString(sys)
	knownSandboxes[i] = int(pid)
	gosb.DynRegisterSandbox(i, m, s)
}

//export SB_GetSandboxPid
func SB_GetSandboxPid(id *C.char) C.int {
	i, ok := knownSandboxes[C.GoString(id)]
	if !ok {
		return C.int(-1)
	}
	return C.int(i)
}

//export SB_RegisterSandboxDependency
func SB_RegisterSandboxDependency(id *C.char, pkg *C.char) {
	i, p := C.GoString(id), C.GoString(pkg)
	// It is a synthetic package.
	if _, ok := commons.PythonSynthetic[p]; ok {
		return
	}
	gosb.DynRegisterSandboxDependency(i, p)
}

//export SB_RegisterGrowth
func SB_RegisterGrowth(isrt C.int, addr unsafe.Pointer, size C.size_t) {
	gosb.ExtendSpace(int(isrt) == 1, uintptr(addr), uintptr(size))
}

//export SB_AddSection
func SB_AddSection(id C.int, addr unsafe.Pointer, size C.size_t) {
	gosb.DynAddSection(int(id), uintptr(addr), uintptr(size))
}

//export SB_switch_rt
func SB_switch_rt() {
	if canSwitch && getid() != "GOD" {
		vtx.DynToGod()
	}
}

//export SB_switch_in
func SB_switch_in() {
	if canSwitch && getid() != "GOD" {
		vtx.DynGoBack()
	}
}

//export SB_refcount
func SB_refcount(ptr unsafe.Pointer, curr C.int, incr C.int) {
	addr := uintptr(ptr)
	val := int(curr) + int(incr)
	references[addr] = val
}

//export SB_checkref
func SB_checkref(pkg C.int, ptr unsafe.Pointer, curr C.int, incr C.int) {
	id := getid()
	if globals.DynIsRO(id, int(pkg)) {
		SB_refcount(ptr, curr, incr)
	}
}

//export SB_isRO
func SB_isRO(pkg C.int) C.int {
	id := getid()
	if !canSwitch || id == "GOD" {
		return C.int(0)
	}
	if globals.DynIsRO(id, int(pkg)) {
		return C.int(1)
	}
	return C.int(0)
}

//export SB_showref
func SB_showref() {
	fmt.Println("How many refs did we capture? ", len(references))
	fmt.Println("The RO ", globals.SBRefCountSkip)
}

func getid() commons.SandId {
	if len(StackIds) == 0 {
		return "GOD"
	}
	return StackIds[len(StackIds)-1]
}

func getPrevId() commons.SandId {
	if len(StackIds) <= 1 {
		return "GOD"
	}
	return StackIds[len(StackIds)-2]
}

func main() {
	fmt.Println("Hey how")
}
