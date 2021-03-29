package ring0

import (
	"unsafe"
)

// eface mirrors runtime.eface.
type eface struct {
	typ  uintptr
	data unsafe.Pointer
}

// kernelAddr returns the kernel virtual address for the given object.
//
//go:nosplit
func kernelAddr(obj interface{}) uintptr {
	e := (*eface)(unsafe.Pointer(&obj))
	return KernelStartAddress | uintptr(e.data)
}

// kernelFunc returns the address of the given function.
//
//go:nosplit
func kernelFunc(fn func()) uintptr {
	fnptr := (**uintptr)(unsafe.Pointer(&fn))
	return /*KernelStartAddress |*/ **fnptr
}
