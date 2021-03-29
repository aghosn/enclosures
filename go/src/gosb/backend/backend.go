package backend

import (
	c "gosb/commons"
)

type Backend = int

type BackendConfig struct {
	Tpe Backend
	//Functions for hooks in the runtime
	Init          func()
	Prolog        func(id c.SandId)
	Epilog        func(id c.SandId)
	Transfer      func(oldid, newid int, start, size uintptr)
	Register      func(id int, start, size uintptr)
	Execute       func(id c.SandId)
	Mstart        func()
	RuntimeGrowth func(isheap bool, id int, start, size uintptr)
	Stats         func()
}

const (
	SIM_BACKEND  Backend = iota
	VTX_BACKEND  Backend = iota
	MPK_BACKEND  Backend = iota
	DVTX_BACKEND Backend = iota
	DMPK_BACKEND Backend = iota
	BACKEND_SIZE Backend = iota
)

var (
	BackendNames = []string{
		"SIM",
		"VTX",
		"MPK",
		"DVTX",
		"DMPK",
	}
)
