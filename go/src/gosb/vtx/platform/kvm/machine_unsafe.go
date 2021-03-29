package kvm

import (
	"fmt"
	"gosb/commons"
	"sync/atomic"
	"syscall"
	"unsafe"
)

//go:linkname entersyscall runtime.entersyscall
func entersyscall()

//go:linkname exitsyscall runtime.exitsyscall
func exitsyscall()

// setEPTRegion initializes a region.
//
// This may be called from bluepillHandler, and therefore returns an errno
// directly (instead of wrapping in an error) to avoid allocations.
//
//go:nosplit
func (m *Machine) setEPTRegion(slot *uint32, physical, length, virtual uint64, flags uint32) (uint32, syscall.Errno) {
	commons.Check(flags == 1)
	v := atomic.AddUint32(slot, 1)
	userRegion := userMemoryRegion{
		slot:          uint32(v),
		flags:         uint32(flags),
		guestPhysAddr: uint64(physical),
		memorySize:    uint64(length),
		userspaceAddr: uint64(virtual),
	}

	// Set the region.
	_, errno := commons.Ioctl(m.fd, _KVM_SET_USER_MEMORY_REGION, uintptr(unsafe.Pointer(&userRegion)))
	return v, errno
}

//go:nosplit
func (m *Machine) DynSetEPTRegion(slot *uint32, physical, length, virtual uint64, flags uint32) (uint32, syscall.Errno) {
	return m.setEPTRegion(slot, physical, length, virtual, flags)
}

// mapRunData maps the vCPU run data.
func mapRunData(fd int) (*runData, error) {
	r, errno := commons.Mmap(0, uintptr(runDataSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED, fd, 0)
	if errno != 0 {
		return nil, fmt.Errorf("error mapping runData: %v", errno)
	}
	return (*runData)(unsafe.Pointer(r)), nil
}
