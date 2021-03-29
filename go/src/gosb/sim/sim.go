package sim

import (
	c "gosb/commons"
	"runtime"
)

var (
	countEntries map[c.SandId]int
)

//go:noinline
//go:nosplit
func Init() {
}

//go:noinline
//go:nosplit
func Prolog(id c.SandId) {
	runtime.AssignSbId(id, false)
}

//go:noinline
//go:nosplit
func Epilog(id c.SandId) {
	runtime.AssignSbId("", true)
}

//go:nosplit
func Transfer(oldid, newid int, start, size uintptr) {
	// Cannot do anything for now because of malloc
	return
}

//go:nosplit
func Register(id int, start, size uintptr) {
	// Cannot do anything now because of malloc
	return
}

//go:nosplit
func RuntimeGrowth(isheap bool, id int, start, size uintptr) {
	// Nothing to do
	return
}

//go:nosplit
func Execute(id c.SandId) {
	runtime.AssignSbId(id, false)
	return
}

func Stats() {
}
