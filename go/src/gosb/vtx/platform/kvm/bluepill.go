package kvm

import (
	"gosb/commons"
	"gosb/vtx/arch"
	"gosb/vtx/platform/ring0"
	"log"
	"reflect"
	"syscall"
)

const (
	// Redpill values
	RED_EXIT   = 0x111
	RED_NORM   = 0x555
	RED_GOD    = 0x777
	RED_SWITCH = 0x888
	RED_CHECK  = 0x999
)

// bluepille1 asm to enter guest mode.
func bluepill1(*VCPU)

// bluepill enters guest mode.
func bluepill(v *VCPU) {
	v.Entries++
	bluepill1(v)
}

// sighandler is the signal entry point.
func sighandler()

// dieTrampoline is the assembly trampoline. This calls dieHandler.
//
// This uses an architecture-specific calling convention, documented in
// dieArchSetup and the assembly implementation for dieTrampoline.
func dieTrampoline()

var (
	// bounceSignal is the signal used for bouncing KVM.
	//
	// We use SIGCHLD because it is not masked by the runtime, and
	// it will be ignored properly by other parts of the kernel.
	bounceSignal = syscall.SIGCHLD

	// bounceSignalMask has only bounceSignal set.
	bounceSignalMask = uint64(1 << (uint64(bounceSignal) - 1))

	// bounce is the interrupt vector used to return to the kernel.
	bounce = uint32(ring0.VirtualizationException)

	// savedHandler is a pointer to the previous handler.
	//
	// This is called by bluepillHandler.
	savedHandler uintptr

	// dieTrampolineAddr is the address of dieTrampoline.
	dieTrampolineAddr uintptr
)

// redpill invokes a syscall with -1.
//
//go:nosplit
func redpill(tag uintptr) {
	syscall.RawSyscall(^uintptr(0), tag /*0x111*/, 0x222, 0x333)
}

// Redpill invokes a syscall with -1
//
//go:nosplit
func Redpill(tag uintptr) {
	redpill(tag)
}

// RedSwitch invokes syscall with -1 and target env
//
//go:nosplit
func RedSwitch(cr3 uintptr) {
	syscall.RawSyscall(^uintptr(0), RED_SWITCH, cr3, 0x333)
}

// dieHandler is called by dieTrampoline.
//
//go:nosplit
func dieHandler(c *VCPU) {
	throw(c.dieState.message)
}

// die is called to set the VCPU up to panic.
//
// This loads VCPU state, and sets up a call for the trampoline.
//
//go:nosplit
func (c *VCPU) die(context *arch.SignalContext64, msg string) {
	// Save the death message, which will be thrown.
	c.dieState.message = msg

	// Setup the trampoline.
	dieArchSetup(c, context, &c.dieState.guestRegs)
}
func KVMInit() {
	// Install the handler.
	if err := commons.ReplaceSignalHandler(bluepillSignal, reflect.ValueOf(sighandler).Pointer(), &savedHandler); err != nil {
		log.Fatalf("Unable to set handler for signal %d: %v", bluepillSignal, err)
	}

	// Extract the address for the trampoline.
	dieTrampolineAddr = reflect.ValueOf(dieTrampoline).Pointer()
}
