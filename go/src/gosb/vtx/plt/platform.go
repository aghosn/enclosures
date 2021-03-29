package plt

import "fmt"

var (
	// ErrContextSignal is returned by Context.Switch() to indicate that the
	// Context was interrupted by a signal.
	ErrContextSignal = fmt.Errorf("interrupted by signal")

	// ErrContextSignalCPUID is equivalent to ErrContextSignal, except that
	// a check should be done for execution of the CPUID instruction. If
	// the current instruction pointer is a CPUID instruction, then this
	// should be emulated appropriately. If not, then the given signal
	// should be handled per above.
	ErrContextSignalCPUID = fmt.Errorf("interrupted by signal, possible CPUID")

	// ErrContextInterrupt is returned by Context.Switch() to indicate that the
	// Context was interrupted by a call to Context.Interrupt().
	ErrContextInterrupt = fmt.Errorf("interrupted by platform.Context.Interrupt()")

	// ErrContextCPUPreempted is returned by Context.Switch() to indicate that
	// one of the following occurred:
	//
	// - The CPU executing the Context is not the CPU passed to
	// Context.Switch().
	//
	// - The CPU executing the Context may have executed another Context since
	// the last time it executed this one; or the CPU has previously executed
	// another Context, and has never executed this one.
	//
	// - Platform.PreemptAllCPUs() was called since the last return from
	// Context.Switch().
	ErrContextCPUPreempted = fmt.Errorf("interrupted by CPU preemption")
)

// AddressSpace represents a virtual address space in which a Context can
// execute.
type AddressSpace interface {
	// MapFile creates a shared mapping of offsets fr from f at address addr.
	// Any existing overlapping mappings are silently replaced.
	//
	// If precommit is true, the platform should eagerly commit resources (e.g.
	// physical memory) to the mapping. The precommit flag is advisory and
	// implementations may choose to ignore it.
	//
	// Preconditions: addr and fr must be page-aligned. fr.Length() > 0.
	// at.Any() == true. At least one reference must be held on all pages in
	// fr, and must continue to be held as long as pages are mapped.
	//MapFile(addr usermem.Addr, f File, fr FileRange, at usermem.AccessType, precommit bool) error

	// Unmap unmaps the given range.
	//
	// Preconditions: addr is page-aligned. length > 0.
	//Unmap(addr usermem.Addr, length uint64)

	// Release releases this address space. After releasing, a new AddressSpace
	// must be acquired via platform.NewAddressSpace().
	//Release()

	// AddressSpaceIO methods are supported iff the associated platform's
	// Platform.SupportsAddressSpaceIO() == true. AddressSpaces for which this
	// does not hold may panic if AddressSpaceIO methods are invoked.
	//AddressSpaceIO
}
