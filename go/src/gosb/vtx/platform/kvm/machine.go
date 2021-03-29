package kvm

import (
	"gosb/commons"
	"gosb/vtx/arch"
	"gosb/vtx/atomicbitops"
	mv "gosb/vtx/platform/memview"
	"gosb/vtx/platform/procid"
	"gosb/vtx/platform/ring0"
	"log"
	"reflect"
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"
)

type Machine struct {
	// fd is the vm fd
	fd int

	// Memory view for this machine
	MemView *mv.AddressSpace

	// Pointer to the God view
	GodView uintptr

	// kernel is the set of global structures.
	kernel ring0.Kernel

	// @aghosn mutex for us
	//mu runtime.GosbMutex

	// vcpus available to this machine
	vcpus map[int]*VCPU

	// maxVCPUs is the maximum number of VCPUs supported by the machine.
	maxVCPUs int

	Start uintptr

	// For address space extension.
	Mu runtime.GosbMutex
}

const (
	// VCPUReady is an alias for all the below clear.
	VCPUReady uint32 = 0

	// VCPUser indicates that the VCPU is in or about to enter user mode.
	VCPUUser uint32 = 1 << 0

	// VCPUGuest indicates the VCPU is in guest mode.
	VCPUGuest uint32 = 1 << 1

	cpuScale = 4
)

// VCPU is a single KVM VCPU.
type VCPU struct {
	// CPU is the kernel CPU data.
	//
	// This must be the first element of this structure, it is referenced
	// by the bluepill code (see bluepill_amd64.s).
	ring0.CPU

	// id is the VCPU id.
	id int

	// fd is the VCPU fd.
	fd int

	// tid is the last set tid.
	tid uint64
	// state is the VCPU state.
	//
	// This is a bitmask of the three fields (VCPU*) described above.
	state uint32

	// runData for this VCPU.
	runData *runData

	// machine associated with this VCPU.
	machine *Machine

	// VCPUArchState is the architecture-specific state.
	VCPUArchState

	dieState dieState

	// let's us decide whether the vcpu should be changed.
	entered bool

	// marking the exception error.
	exceptionCode int

	// cr2 for the fault
	FaultAddr uintptr

	// fault information
	Info arch.SignalInfo

	uregs syscall.PtraceRegs

	// Current memview and sys filter
	Memview   *mv.AddressSpace
	Sysfilter *commons.SyscallMask

	// Counters for statistics
	Entries uint64 // # of calls to bluepill
	Exits   uint64 // # of calls to redpill
	Escapes uint64 // # of unvoluntary exits, e.g., syscalls
}

type dieState struct {
	// message is thrown from die.
	message string

	// guestRegs is used to store register state during VCPU.die() to prevent
	// allocation inside nosplit function.
	guestRegs userRegs

	sysRegs systemRegs
}

func (m *Machine) newVCPU() *VCPU {
	id := len(m.vcpus)
	// Create the VCPU.
	fd, errno := commons.Ioctl(m.fd, _KVM_CREATE_VCPU, uintptr(id))
	if errno != 0 {
		log.Printf("error creating new VCPU: %v\n", errno)
	}

	c := &VCPU{
		id:        id,
		fd:        fd,
		machine:   m,
		Memview:   mv.GodAS,
		Sysfilter: &commons.SyscallAll,
	}
	c.CPU.Init(&m.kernel, c)
	m.vcpus[c.id] = c

	// Ensure the signal mask is correct.
	if err := c.setSignalMask(); err != nil {
		log.Fatalf("error setting signal mask: %v\n", err)
	}

	// Map the run data.
	runData, err := mapRunData(int(fd))
	if err != nil {
		log.Fatalf("error mapping run data: %v\n", err)
	}
	c.runData = runData

	// Initialize architecture state.
	if err := c.initArchState(); err != nil {
		log.Fatalf("error initialization VCPU state: %v\n", err)
	}
	return c
}

//go:nosplit
func (m *Machine) Replenish() {
	m.MemView.Replenish()
}

/*
//go:nosplit
func (m *Machine) ValidAddress(addr uint64) bool {
	//TODO: aghosn, should fix this because now we have multiple cr3...
	return m.MemView.ValidAddress(addr)
}
*/
//go:nosplit
func (m *Machine) HasRights(addr uint64, prot uint8) bool {
	return m.MemView.HasRights(addr, prot)
}

//go:nosplit
func (m *Machine) Fd() int {
	return m.fd
}

func newMachine(vm int, d *commons.SandboxMemory, template *mv.AddressSpace) (*Machine, error) {
	memview := template.Copy(false)
	memview.ApplyDomain(d)
	// Create the machine.
	m := &Machine{
		fd:      vm,
		MemView: memview,
		GodView: uintptr(mv.GodAS.Tables.CR3(false, 0)),
		vcpus:   make(map[int]*VCPU),
	}
	memview.RegisterGrowth(
		//go:nosplit
		func(p, l, v uint64, f uint32) {
			m.setEPTRegion(&memview.NextSlot, p, l, v, f)
		})
	memview.Seal()
	m.Start = reflect.ValueOf(ring0.Start).Pointer()
	m.kernel.Init(ring0.KernelOpts{PageTables: memview.Tables})
	m.maxVCPUs = runtime.GOMAXPROCS(0) * cpuScale
	maxVCPUs, errno := commons.Ioctl(m.fd, _KVM_CHECK_EXTENSION, _KVM_CAP_MAX_VCPUS)
	if errno != 0 && maxVCPUs < m.maxVCPUs {
		m.maxVCPUs = _KVM_NR_VCPUS
	}
	// Register the memory address range.
	m.SetAllEPTSlots()

	// Initialize architecture state.
	if err := m.initArchState(); err != nil {
		log.Fatalf("Error initializing machine %v\n", err)
	}
	return m, nil
}

// CreateVirtualMachine creates a single virtual machine based on the default view.
func CreateVirtualMachine(kvmfd int, seal bool) *Machine {
	var (
		vm    int
		errno syscall.Errno
	)
	for {
		vm, errno = commons.Ioctl(kvmfd, _KVM_CREATE_VM, 0)
		if errno == syscall.EINTR {
			continue
		}
		commons.Check(errno == 0)
		break
	}
	m := &Machine{
		fd:      vm,
		MemView: mv.GodAS,
		GodView: uintptr(mv.GodAS.Tables.CR3(false, 0)),
		vcpus:   make(map[int]*VCPU),
	}
	mv.GodAS.RegisterGrowth(
		//go:nosplit
		func(p, l, v uint64, f uint32) {
			m.setEPTRegion(&mv.GodAS.NextSlot, p, l, v, f)
		})
	if seal {
		mv.GodAS.Seal()
	}
	m.Start = reflect.ValueOf(ring0.Start).Pointer()
	m.kernel.Init(ring0.KernelOpts{PageTables: mv.GodAS.Tables})
	m.maxVCPUs = runtime.GOMAXPROCS(0) * cpuScale
	maxVCPUs, errno := commons.Ioctl(m.fd, _KVM_CHECK_EXTENSION, _KVM_CAP_MAX_VCPUS)
	if errno != 0 && maxVCPUs < m.maxVCPUs {
		m.maxVCPUs = _KVM_NR_VCPUS
	}

	// Register the memory address range.
	m.SetAllEPTSlots()

	// Initialize architecture state.
	if err := m.initArchState(); err != nil {
		log.Fatalf("Error initializing machine %v\n", err)
	}
	return m
}

// CreateVCPU attempts to allocate a new vcpu.
// This should only be called in a normal go state as it does allocation
// and hence will split the stack.
func (m *Machine) CreateVCPU() {
	m.Mu.Lock()
	if len(m.vcpus) < m.maxVCPUs {
		id := len(m.vcpus)
		if _, ok := m.vcpus[id]; ok {
			panic("Duplicated cpu id")
		}
		_ = m.newVCPU()
	}
	m.Mu.Unlock()
}

// returns with os thread locked
//go:nosplit
func (m *Machine) Get() *VCPU {
	m.Mu.Lock()
	runtime.LockOSThread()
	for _, c := range m.vcpus {
		if atomic.CompareAndSwapUint32(&c.state, VCPUReady, VCPUUser) {
			m.Mu.Unlock()
			tid := procid.Current()
			c.loadSegments(tid)
			return c
		}
	}
	// Failure, should be impossible.
	runtime.UnlockOSThread()
	if len(m.vcpus) < m.maxVCPUs {
		m.Mu.Unlock()
		panic("Unable to get a cpu but still have space.")
	}
	m.Mu.Unlock()
	panic("Unable to get a cpu")
	return nil
}

func (m *Machine) Put(c *VCPU) {
	c.unlock()
}

// lock marks the VCPU as in user mode.
//
// This should only be called directly when known to be safe, i.e. when
// the VCPU is owned by the current TID with no chance of theft.
//
//go:nosplit
func (c *VCPU) lock() {
	atomicbitops.OrUint32(&c.state, VCPUUser)
}

// unlock clears the VCPUUser bit.
//
//go:nosplit
func (c *VCPU) unlock() {
	atomic.SwapUint32(&c.state, VCPUReady)
}

//go:nosplit
func (c *VCPU) MMIOFault(phys uint64) {
	commons.Check(c.Memview != nil)
	virt, ok := c.Memview.FindVirtualForPhys(phys)
	if !ok {
		throw("couldn't find the address")
	}
	commons.Check(c.Memview.ValidAddress(virt))
	commons.Check(c.Memview.HasRights(virt, commons.R_VAL|commons.USER_VAL|commons.W_VAL))
	data := (*[8]byte)(unsafe.Pointer(&c.runData.data[1]))
	length := (uintptr)((uint32)(c.runData.data[2]))
	write := (uint8)(((c.runData.data[2] >> 32) & 0xff)) != 0
	for i := uintptr(0); i < length; i++ {
		b := bytePtr(uintptr(virt) + i)
		if write {
			*b = data[i]
		} else {
			data[i] = *b
		}
	}
}

//go:nosplit
func SetVCPUAttributes(vcpuptr uintptr, view *mv.AddressSpace, sys *commons.SyscallMask) {
	commons.Check(view != nil)
	vcpu := (*VCPU)(unsafe.Pointer(vcpuptr))
	vcpu.Memview = view
	vcpu.Sysfilter = sys
}

func (m *Machine) CollectStats() (uint64, uint64, uint64) {
	e, ex, es := uint64(0), uint64(0), uint64(0)
	for _, v := range m.vcpus {
		e += v.Entries
		ex += v.Exits
		es += v.Escapes
	}
	return e, ex, es
}
