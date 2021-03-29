package kvm

import (
	"fmt"
	"gosb/commons"
	"gosb/vtx/linux"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// loadSegments copies the current segments.
//
// This may be called from within the signal context and throws on error.
//
//go:nosplit
func (c *VCPU) loadSegments(tid uint64) {
	if _, _, errno := syscall.RawSyscall(
		syscall.SYS_ARCH_PRCTL,
		linux.ARCH_GET_FS,
		uintptr(unsafe.Pointer(&c.CPU.Registers().Fs_base)),
		0); errno != 0 {
		throw("getting FS segment")
	}

	if c.CPU.Registers() == nil {
		panic("Wut")
	}
	if _, _, errno := syscall.RawSyscall(
		syscall.SYS_ARCH_PRCTL,
		linux.ARCH_GET_GS,
		uintptr(unsafe.Pointer(&c.CPU.Registers().Gs_base)),
		0); errno != 0 {
		throw("getting GS segment")
	}
	atomic.StoreUint64(&c.tid, tid)
}

// setSignalMask sets the VCPU signal mask.
//
// This must be called prior to running the VCPU.
func (c *VCPU) setSignalMask() error {
	// The layout of this structure implies that it will not necessarily be
	// the same layout chosen by the Go compiler. It gets fudged here.
	var data struct {
		length uint32
		mask1  uint32
		mask2  uint32
		_      uint32
	}
	data.length = 8 // Fixed sigset size.
	data.mask1 = ^uint32(bounceSignalMask & 0xffffffff)
	data.mask2 = ^uint32(bounceSignalMask >> 32)
	if _, errno := commons.Ioctl(c.fd, _KVM_SET_SIGNAL_MASK,
		uintptr(unsafe.Pointer(&data))); errno != 0 {
		return fmt.Errorf("error setting signal mask: %v\n", errno)
	}
	return nil
}

// setCPUID sets the CPUID to be used by the guest.
func (c *VCPU) setCPUID() error {
	if _, errno := commons.Ioctl(c.fd, _KVM_SET_CPUID2, uintptr(unsafe.Pointer(&cpuidSupported))); errno != 0 {
		return fmt.Errorf("error setting CPUID: %v", errno)
	}
	return nil
}

// setUserRegisters sets user registers in the VCPU.
func (c *VCPU) setUserRegisters(uregs *userRegs) error {
	if _, errno := commons.Ioctl(c.fd, _KVM_SET_REGS, uintptr(unsafe.Pointer(uregs))); errno != 0 {
		return fmt.Errorf("error setting user registers: %v", errno)
	}
	return nil
}

// getUserRegisters reloads user registers in the VCPU.
//
// This is safe to call from a nosplit context.
//
//go:nosplit
func (c *VCPU) getUserRegisters(uregs *userRegs) syscall.Errno {
	if _, errno := commons.Ioctl(c.fd, _KVM_GET_REGS, uintptr(unsafe.Pointer(uregs))); errno != 0 {
		return errno
	}
	return 0
}

// setSystemRegisters sets system registers.
//go:nosplit
func (c *VCPU) setSystemRegisters(sregs *systemRegs) error {
	if _, errno := commons.Ioctl(c.fd, _KVM_SET_SREGS, uintptr(unsafe.Pointer(sregs))); errno != 0 {
		return fmt.Errorf("error setting system registers: %v", errno)
	}
	return nil
}

// getSystemRegisters gets system registers.
//go:nosplit
func (c *VCPU) getSystemRegisters(sregs *systemRegs) error {
	if _, errno := commons.Ioctl(c.fd, _KVM_GET_SREGS, uintptr(unsafe.Pointer(sregs))); errno != 0 {
		return fmt.Errorf("error setting system registers: %v", errno)
	}
	return nil
}
