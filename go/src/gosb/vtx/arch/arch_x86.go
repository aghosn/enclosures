package arch

import (
	"gosb/vtx/cpuid"
)

// x86FPState is x86 floating point state.
type x86FPState []byte

// initX86FPState (defined in asm files) sets up initial state.
func initX86FPState(data *FloatingPointData, useXsave bool)

const (
	// minXstateBytes is the minimum size in bytes of an x86 XSAVE area, equal
	// to the size of the XSAVE legacy area (512 bytes) plus the size of the
	// XSAVE header (64 bytes). Equivalently, minXstateBytes is GDB's
	// X86_XSTATE_SSE_SIZE.
	minXstateBytes = 512 + 64

	// userXstateXCR0Offset is the offset in bytes of the USER_XSTATE_XCR0_WORD
	// field in Linux's struct user_xstateregs, which is the type manipulated
	// by ptrace(PTRACE_GET/SETREGSET, NT_X86_XSTATE). Equivalently,
	// userXstateXCR0Offset is GDB's I386_LINUX_XSAVE_XCR0_OFFSET.
	userXstateXCR0Offset = 464

	// xstateBVOffset is the offset in bytes of the XSTATE_BV field in an x86
	// XSAVE area.
	xstateBVOffset = 512

	// xsaveHeaderZeroedOffset and xsaveHeaderZeroedBytes indicate parts of the
	// XSAVE header that we coerce to zero: "Bytes 15:8 of the XSAVE header is
	// a state-component bitmap called XCOMP_BV. ... Bytes 63:16 of the XSAVE
	// header are reserved." - Intel SDM Vol. 1, Section 13.4.2 "XSAVE Header".
	// Linux ignores XCOMP_BV, but it's able to recover from XRSTOR #GP
	// exceptions resulting from invalid values; we aren't. Linux also never
	// uses the compacted format when doing XSAVE and doesn't even define the
	// compaction extensions to XSAVE as a CPU feature, so for simplicity we
	// assume no one is using them.
	xsaveHeaderZeroedOffset = 512 + 8
	xsaveHeaderZeroedBytes  = 64 - 8
)

// newX86FPState returns an initialized floating point state.
//
// The returned state is large enough to store all floating point state
// supported by host, even if the app won't use much of it due to a restricted
// FeatureSet. Since they may still be able to see state not advertised by
// CPUID we must ensure it does not contain any sentry state.
func newX86FPState() x86FPState {
	f := x86FPState(newX86FPStateSlice())
	initX86FPState(f.FloatingPointData(), cpuid.HostFeatureSet().UseXsave())
	return f
}

func newX86FPStateSlice() []byte {
	size, align := cpuid.HostFeatureSet().ExtendedStateSize()
	capacity := size
	// Always use at least 4096 bytes.
	//
	// For the KVM platform, this state is a fixed 4096 bytes, so make sure
	// that the underlying array is at _least_ that size otherwise we will
	// corrupt random memory. This is not a pleasant thing to debug.
	if capacity < 4096 {
		capacity = 4096
	}
	return alignedBytes(capacity, align)[:size]
}

// FloatingPointData returns the raw data pointer.
func (f x86FPState) FloatingPointData() *FloatingPointData {
	return (*FloatingPointData)(&f[0])
}
