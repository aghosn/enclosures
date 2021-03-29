package kvm

import (
	"gosb/vtx/arch"
	"gosb/vtx/platform/ring0"
	"unsafe"
)

// dieArchSetup initializes the state for dieTrampoline.
//
// The amd64 dieTrampoline requires the VCPU to be set in BX, and the last RIP
// to be in AX. The trampoline then simulates a call to dieHandler from the
// provided RIP.
//
//go:nosplit
func dieArchSetup(c *VCPU, context *arch.SignalContext64, guestRegs *userRegs) {
	// Reload all registers to have an accurate stack trace when we return
	// to host mode. This means that the stack should be unwound correctly.
	if errno := c.getUserRegisters(&c.dieState.guestRegs); errno != 0 {
		throw(c.dieState.message)
	}

	if errno := c.getSystemRegisters(&c.dieState.sysRegs); errno != nil {
		throw(c.dieState.message)
	}
	// Small hack to keep track of registers
	if c.exceptionCode == int(ring0.PageFault) {
		c.dieState.guestRegs.R15 = c.dieState.sysRegs.CR2
	}

	// If the VCPU is in user mode, we set the stack to the stored stack
	// value in the VCPU itself. We don't want to unwind the user stack.
	_, isuser := c.ErrorCode()
	if (guestRegs.RFLAGS&ring0.UserFlagsSet == ring0.UserFlagsSet) || isuser {
		regs := c.CPU.Registers()
		context.Rax = regs.Rip
		//context.Rax = regs.Rax
		context.Rsp = regs.Rsp
		context.Rbp = regs.Rbp
	} else {
		context.Rax = guestRegs.RIP
		context.Rsp = guestRegs.RSP
		context.Rbp = guestRegs.RBP
		context.Eflags = guestRegs.RFLAGS
		// TODO(aghosn) remove afterwards
		context.Rsp = c.CPU.Registers().Rsp
	}
	context.Rbx = uint64(uintptr(unsafe.Pointer(c)))
	context.Rip = uint64(dieTrampolineAddr)
}
