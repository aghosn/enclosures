package vtx

import (
	"unsafe"
)

// Just a place holder to keep go from complaining.
type __unsafePointer = unsafe.Pointer

// Usefull for errors that cannot split the stack
//
//go:linkname throw runtime.throw
func throw(string)
