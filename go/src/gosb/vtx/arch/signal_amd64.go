package arch

import (
	"gosb/vtx/linux"
)

// SignalContext64 is equivalent to struct sigcontext, the type passed as the
// second argument to signal handlers set by signal(2).
type SignalContext64 struct {
	R8      uint64
	R9      uint64
	R10     uint64
	R11     uint64
	R12     uint64
	R13     uint64
	R14     uint64
	R15     uint64
	Rdi     uint64
	Rsi     uint64
	Rbp     uint64
	Rbx     uint64
	Rdx     uint64
	Rax     uint64
	Rcx     uint64
	Rsp     uint64
	Rip     uint64
	Eflags  uint64
	Cs      uint16
	Gs      uint16 // always 0 on amd64.
	Fs      uint16 // always 0 on amd64.
	Ss      uint16 // only restored if _UC_STRICT_RESTORE_SS (unsupported).
	Err     uint64
	Trapno  uint64
	Oldmask linux.SignalSet
	Cr2     uint64
	// Pointer to a struct _fpstate. See b/33003106#comment8.
	Fpstate  uint64
	Reserved [8]uint64
}

// Flags for UContext64.Flags.
const (
	_UC_FP_XSTATE         = 1
	_UC_SIGCONTEXT_SS     = 2
	_UC_STRICT_RESTORE_SS = 4
)

// UContext64 is equivalent to ucontext_t on 64-bit x86.
type UContext64 struct {
	Flags    uint64
	Link     uint64
	Stack    SignalStack
	MContext SignalContext64
	Sigset   linux.SignalSet
}
