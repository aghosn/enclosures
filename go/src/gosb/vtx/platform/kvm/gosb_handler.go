package kvm

import (
	c "gosb/commons"
	"gosb/vtx/platform/memview"
	"gosb/vtx/platform/ring0"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	_SYSCALL_INSTR = uint16(0x050f)
)

type sysHType = uint8

const (
	syshandlerErr1      sysHType = iota // something was wrong
	syshandlerErr2      sysHType = iota // something was wrong
	syshandlerPFW       sysHType = iota // page fault missing write
	syshandlerSNF       sysHType = iota // TODO debugging
	syshandlerPF        sysHType = iota // page fault missing not mapped
	syshandlerException sysHType = iota
	syshandlerValid     sysHType = iota // valid system call
	syshandlerInvalid   sysHType = iota // unallowed system call
	syshandlerBail      sysHType = iota // redpill
)

var (
	MRTRip     uint64  = 0
	MRTFsbase  uint64  = 0
	MRTFault   uintptr = 0
	MRTSpanId  int     = 0
	MRTEntry   uintptr = 0
	MRTEntry2  uintptr = 0
	MRTSpan    uintptr = 0
	MRTAddr    uintptr = 0
	MRTAddr2   uintptr = 0
	MRTFd      int     = 0
	MRTMaped   int     = 0
	MRTMaped2  int     = 0
	MRTValid   bool    = false
	MRTFound   bool    = false
	MRTSigproc uint64  = 0
	MRTSigalt  uint64  = 0
	MRTBkr     uint64  = 0
)

//go:nosplit
func kvmSyscallHandler(vcpu *VCPU) sysHType {
	regs := vcpu.Registers()

	// 1. Check that the Rip is valid, @later use seccomp too to disambiguate kern/user.
	// No lock, this part never changes.
	c.Check(vcpu.Memview != nil)
	if !vcpu.Memview.ValidAddress(regs.Rip) && vcpu.machine.HasRights(regs.Rip, c.X_VAL) {
		return syshandlerErr1
	}

	// 2. Check that Rip is a syscall.
	instr := (*uint16)(unsafe.Pointer(uintptr(regs.Rip - 2)))
	if *instr == _SYSCALL_INSTR {
		// It is a redpill.
		if regs.Rax == ^uint64(0) {
			vcpu.Exits++
			return syshandlerBail
		}

		// Validate the syscall
		c.Check(vcpu.Sysfilter != nil)
		// Filter the system call
		idx, idy := c.SysCoords(int(regs.Rax))
		if vcpu.Sysfilter[idx]&(1<<idy) == 0 {
			return syshandlerInvalid
		}

		// Perform the syscall, here we will interpose.
		// 3. Do a raw syscall now.
		//TODO try to avoid the sigprocmask
		if regs.Rax == syscall.SYS_RT_SIGPROCMASK || regs.Rax == syscall.SYS_SIGALTSTACK {
			regs.Rax = 0
			regs.Rdx = 0
			return syshandlerValid
		}

		if regs.Rax == syscall.SYS_BRK {
			if !(regs.Rdi >= memview.CheapStart && regs.Rdi < (memview.CheapStart+memview.CheapSize)) {
				throw("sbrk out of range")
			} else {
				MRTBkr++
			}
		}
		r1, r2, err := syscall.RawSyscall6(uintptr(regs.Rax),
			uintptr(regs.Rdi), uintptr(regs.Rsi), uintptr(regs.Rdx),
			uintptr(regs.R10), uintptr(regs.R8), uintptr(regs.R9))

		if err != 0 {
			regs.Rax = uint64(-err)
		} else {
			regs.Rax = uint64(r1)
			regs.Rdx = uint64(r2)
		}
		regs.Rdx = uint64(r2)
		vcpu.Escapes++
		return syshandlerValid
	}

	// This is a breakpoint
	if vcpu.exceptionCode == int(ring0.Breakpoint) {
		vcpu.exceptionCode = -1
		regs.Rip--
		return syshandlerValid
	}

	if vcpu.exceptionCode == int(ring0.PageFault) {
		// Lock as it might be modified
		vcpu.machine.Mu.Lock()

		// Check if we have a concurrency issue.
		// The thread as been reshuffled to service that thread and is not properly
		// mapped and hence we should go back.
		if vcpu.Memview.ValidAddress(uint64(vcpu.FaultAddr)) {
			if vcpu.Memview.HasRights(uint64(vcpu.FaultAddr), c.R_VAL|c.USER_VAL|c.W_VAL) {
				MRTRip = vcpu.Registers().Rip
				MRTFsbase = vcpu.Registers().Fs_base
				MRTFault = vcpu.FaultAddr
				MRTAddr, _, MRTEntry = vcpu.machine.MemView.Tables.FindMapping(MRTFault)
				vcpu.machine.Mu.Unlock()
				return syshandlerSNF
			}
			if vcpu.machine.MemView.HasRights(uint64(vcpu.FaultAddr), c.R_VAL) {
				vcpu.machine.Mu.Unlock()
				MRTFault = vcpu.FaultAddr
				MRTMaped = vcpu.Memview.Tables.IsMapped(MRTFault)
				MRTAddr, _, MRTEntry = vcpu.Memview.Tables.FindMapping(MRTFault)
				MRTAddr2, _, MRTEntry2 = memview.GodAS.Tables.FindMapping(MRTFault)
				MRTMaped2 = memview.GodAS.Tables.IsMapped(MRTFault)
				return syshandlerPFW
			}
		}
		MRTFault = vcpu.FaultAddr
		MRTMaped = vcpu.Memview.Tables.IsMapped(MRTFault)
		MRTMaped2 = memview.GodAS.Tables.IsMapped(MRTFault)
		if MRTMaped == 1 {
			MRTAddr, _, MRTEntry = vcpu.machine.MemView.Tables.FindMapping(MRTFault)
		}
		MRTSpanId = runtime.SpanIdOf(vcpu.FaultAddr)
		if MRTSpanId != -666 {
			MRTSpan, _, _ = runtime.GosbSpanOf(MRTFault)
		}
		MRTFd = vcpu.machine.fd
		MRTValid = vcpu.Memview.ValidAddress(uint64(MRTFault))
		vcpu.machine.Mu.Unlock()
		return syshandlerPF
	}
	// Something went wrong that we did not account for.
	// Get the registers directly from KVM to make sure we have the correct ones
	// in gdb.
	if vcpu.exceptionCode != 0 {
		vcpu.getUserRegisters(&uregs)
		vcpu.getSystemRegisters(&sregs)
		return syshandlerException
	}
	return syshandlerErr2
}
