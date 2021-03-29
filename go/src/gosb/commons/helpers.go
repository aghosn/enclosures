package commons

/**
* author: aghosn
* Helper functions that we use to simulate some C code.
**/
import (
	"fmt"
	sc "syscall"
	"unsafe"
)

const (
	Limit39bits = uintptr(1 << 39)
)

//go:nosplit
func Ioctl(fd int, op, arg uintptr) (int, sc.Errno) {
	r1, _, err := sc.RawSyscall(sc.SYS_IOCTL, uintptr(fd), op, arg)
	return int(r1), err
}

func Mmap(addr, size, prot, flags uintptr, fd int, off uintptr) (uintptr, sc.Errno) {
	r1, _, err := sc.RawSyscall6(sc.SYS_MMAP, addr, size, prot, flags, uintptr(fd), off)
	return r1, err
}

func Munmap(addr, size uintptr) sc.Errno {
	_, _, err := sc.RawSyscall(sc.SYS_MUNMAP, addr, size, 0)
	return err
}

func Memcpy(dest, src, size uintptr) {
	if dest == 0 || src == 0 {
		panic("nil argument to copy")
	}
	for i := uintptr(0); i < size; i++ {
		d := (*byte)(unsafe.Pointer(dest + i))
		s := (*byte)(unsafe.Pointer(src + i))
		*d = *s
	}
}

//go:nosplit
func Check(condition bool) {
	if !condition {
		panic("Condition is not satisfied")
	}
}

//go:nosplit
func CheckE(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// @from gvisor
// ReplaceSignalHandler replaces the existing signal handler for the provided
// signal with the one that handles faults in safecopy-protected functions.
//
// It stores the value of the previously set handler in previous.
//
// This function will be called on initialization in order to install safecopy
// handlers for appropriate signals. These handlers will call the previous
// handler however, and if this is function is being used externally then the
// same courtesy is expected.
func ReplaceSignalHandler(sig sc.Signal, handler uintptr, previous *uintptr) error {
	var sa struct {
		handler  uintptr
		flags    uint64
		restorer uintptr
		mask     uint64
	}
	const maskLen = 8

	// Get the existing signal handler information, and save the current
	// handler. Once we replace it, we will use this pointer to fall back to
	// it when we receive other signals.
	if _, _, e := sc.RawSyscall6(
		sc.SYS_RT_SIGACTION, uintptr(sig), 0,
		uintptr(unsafe.Pointer(&sa)), maskLen, 0, 0); e != 0 {
		return e
	}

	// Fail if there isn't a previous handler.
	if sa.handler == 0 {
		return fmt.Errorf("previous handler for signal %x isn't set", sig)
	}

	*previous = sa.handler

	// Install our own handler.
	sa.handler = handler
	if _, _, e := sc.RawSyscall6(
		sc.SYS_RT_SIGACTION, uintptr(sig),
		uintptr(unsafe.Pointer(&sa)), 0, maskLen, 0, 0); e != 0 {
		return e
	}

	return nil
}
