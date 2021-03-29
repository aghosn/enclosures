// Copyright 2018 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ring0

import (
	"syscall"
)

// This is an assembly function.
//
// The sysenter function is invoked in two situations:
//
// (1) The guest kernel executed a syscall.
// (2) The guest application executed a syscall.
//
// In both cases, we save the current state inside vcpu, switch to kernel stack
// push vcpu and exit. Upon return we pop and iret to the vcpu.
func sysenter2()

// swapgs swaps the current GS value.
//
// This must be called prior to sysret/iret.
func swapgs()

// sysret returns to userspace from a system call.
//
// The return code is the vector that interrupted execution.
//
// See stubs.go for a note regarding the frame size of this function.
func sysret(*CPU, *syscall.PtraceRegs) Vector

// "iret is the cadillac of CPL switching."
//
//				-- Neel Natu
//
// iret is nearly identical to sysret, except an iret is used to fully restore
// all user state. This must be called in cases where all registers need to be
// restored.
func iret(*CPU, *syscall.PtraceRegs) Vector

// exception is the generic exception entry.
//
// This is called by the individual stub definitions.
func exception2()

// resume is a stub that restores the CPU kernel registers.
//
// This is used when processing kernel exceptions and syscalls.
func resume()

// resume is a stub that restores the CPU user registers.
//
// This is used when processing kernel exceptions and syscalls.
func resumeUser()

// Start is the CPU entrypoint.
//
// The following start conditions must be satisfied:
//
//  * AX should contain the CPU pointer.
//  * c.GDT() should be loaded as the GDT.
//  * c.IDT() should be loaded as the IDT.
//  * c.CR0() should be the current CR0 value.
//  * c.CR3() should be set to the kernel PageTables.
//  * c.CR4() should be the current CR4 value.
//  * c.EFER() should be the current EFER value.
//
// The CPU state will be set to c.Registers().
func Start()

// Exception stubs.
func divideByZero()
func debug()
func nmi()
func breakpoint()
func overflow()
func boundRangeExceeded()
func invalidOpcode()
func deviceNotAvailable()
func doubleFault()
func coprocessorSegmentOverrun()
func invalidTSS()
func segmentNotPresent()
func stackSegmentFault()
func generalProtectionFault()
func pageFault()
func x87FloatingPointException()
func alignmentCheck()
func machineCheck()
func simdFloatingPointException()
func virtualizationException()
func securityException()
func syscallInt80()

// Exception handler index.
var handlers = map[Vector]func(){
	DivideByZero:               divideByZero,
	Debug:                      debug,
	NMI:                        nmi,
	Breakpoint:                 breakpoint,
	Overflow:                   overflow,
	BoundRangeExceeded:         boundRangeExceeded,
	InvalidOpcode:              invalidOpcode,
	DeviceNotAvailable:         deviceNotAvailable,
	DoubleFault:                doubleFault,
	CoprocessorSegmentOverrun:  coprocessorSegmentOverrun,
	InvalidTSS:                 invalidTSS,
	SegmentNotPresent:          segmentNotPresent,
	StackSegmentFault:          stackSegmentFault,
	GeneralProtectionFault:     generalProtectionFault,
	PageFault:                  pageFault,
	X87FloatingPointException:  x87FloatingPointException,
	AlignmentCheck:             alignmentCheck,
	MachineCheck:               machineCheck,
	SIMDFloatingPointException: simdFloatingPointException,
	VirtualizationException:    virtualizationException,
	SecurityException:          securityException,
	SyscallInt80:               syscallInt80,
}
