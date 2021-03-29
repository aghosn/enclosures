package linux

// From <asm/prctl.h>
// Flags are used in syscall arch_prctl(2).
const (
	ARCH_SET_GS    = 0x1001
	ARCH_SET_FS    = 0x1002
	ARCH_GET_FS    = 0x1003
	ARCH_GET_GS    = 0x1004
	ARCH_SET_CPUID = 0x1012
)
