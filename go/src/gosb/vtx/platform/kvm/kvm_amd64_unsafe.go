package kvm

import (
	"fmt"
	"gosb/commons"
	"syscall"
	"unsafe"
)

var (
	runDataSize    int
	hasGuestPCID   bool
	cpuidSupported = cpuidEntries{nr: _KVM_NR_CPUID_ENTRIES}
)

func updateSystemValues(fd int) error {
	// Extract the mmap size.
	sz, errno := commons.Ioctl(fd, _KVM_GET_VCPU_MMAP_SIZE, 0)
	if errno != 0 {
		return fmt.Errorf("getting VCPU mmap size: %v", errno)
	}

	// Save the data.
	runDataSize = int(sz)

	// Must do the dance to figure out the number of entries.
	_, errno = commons.Ioctl(fd, _KVM_GET_SUPPORTED_CPUID,
		uintptr(unsafe.Pointer(&cpuidSupported)))
	if errno != 0 && errno != syscall.ENOMEM {
		// Some other error occurred.
		return fmt.Errorf("getting supported CPUID: %v", errno)
	}

	// The number should now be correct.
	_, errno = commons.Ioctl(fd, _KVM_GET_SUPPORTED_CPUID, uintptr(unsafe.Pointer(&cpuidSupported)))
	if errno != 0 {
		// Didn't work with the right number.
		return fmt.Errorf("getting supported CPUID (2nd attempt): %v", errno)
	}

	// Calculate whether guestPCID is supported.
	//
	// FIXME(ascannell): These should go through the much more pleasant
	// cpuid package interfaces, once a way to accept raw kvm CPUID entries
	// is plumbed (or some rough equivalent).
	for i := 0; i < int(cpuidSupported.nr); i++ {
		entry := cpuidSupported.entries[i]
		if entry.function == 1 && entry.index == 0 && entry.ecx&(1<<17) != 0 {
			hasGuestPCID = true // Found matching PCID in guest feature set.
		}
	}

	// Success.
	return nil
}
