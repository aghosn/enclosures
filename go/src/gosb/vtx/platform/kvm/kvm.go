package kvm

import (
	"gosb/commons"
	"gosb/globals"
	mv "gosb/vtx/platform/memview"
	"gosb/vtx/platform/ring0"
	"log"
	"reflect"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	bluepillretaddr = uint64(reflect.ValueOf(Bluepillret).Pointer())
)

// Bluepillret does a simple return to avoid doing a CLI again.
//
//go:nosplit
func Bluepillret()

type KVM struct {
	Machine *Machine

	// Pointer to the sandbox memory
	Sand *commons.SandboxMemory

	// Id for the sandbox, this is important for pristine
	Id  commons.SandId
	Pid int
}

// Copy allows to duplicate a sandbox for pristine execution.
// TODO(@aghosn) probably lock
func (k *KVM) Copy(fd int) *KVM {
	v := New(fd, k.Sand, k.Machine.MemView)
	v.Id, v.Pid = globals.PristineId(v.Sand.Config.Id)
	return v
}

// New creates a VM with KVM, and initializes its machine and pagetables.
func New(fd int, d *commons.SandboxMemory, template *mv.AddressSpace) *KVM {
	// Create a new VM fd.
	var (
		vm    int
		errno syscall.Errno
	)
	for {
		vm, errno = commons.Ioctl(fd, _KVM_CREATE_VM, 0)
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 {
			log.Fatalf("creating VM: %v\n", errno)
		}
		break
	}
	machine, err := newMachine(vm, d, template)
	if err != nil {
		log.Fatalf("error creating the machine: %v\n", err)
	}
	kvm := &KVM{Machine: machine, Sand: d, Id: d.Config.Id}
	kvm.Machine.MemView.Replenish()
	return kvm
}

//go:nosplit
func (k *KVM) Map(start, size uintptr, prot uint8) {
	k.Machine.MemView.Toggle(true, start, size, prot)
}

// @warning cannot do dynamic allocation.
//
//go:nosplit
func (k *KVM) ExtendRuntime(heap bool, start, size uintptr, prot uint8) {
	size = uintptr(commons.Round(uint64(size), true))
	if k.Machine.MemView.ContainsRegion(start, size) {
		// Nothing to do, we already mapped it.
		panic("Already exists!")
		return
	}
	// We have to map a new region.
	m := k.Machine.MemView.AcquireEMR()
	var err syscall.Errno
	k.Machine.MemView.Extend(heap, m, uint64(start), uint64(size), prot)
	m.Span.Slot, err = k.Machine.setEPTRegion(
		&k.Machine.MemView.NextSlot, m.Span.GPA, m.Span.Size, m.Span.Start, 0)
	if err != 0 {
		if size%commons.PageSize != 0 {
			panic("Size is shit")
		}
		panic("Error mapping slot")
	}
}

//go:nosplit
func (k *KVM) ExtendRuntime2(orig *mv.MemoryRegion) {
	commons.Check(orig != nil)
	if k.Machine.MemView.ContainsRegion(uintptr(orig.Span.Start), uintptr(orig.Span.Size)) {
		panic("Already exists")
		return
	}
	var err syscall.Errno
	m := k.Machine.MemView.AcquireEMR()
	k.Machine.MemView.Extend2(m, orig)
	m.Span.Slot, err = k.Machine.setEPTRegion(
		&k.Machine.MemView.NextSlot, m.Span.GPA, m.Span.Size, m.Span.Start, 0)
	if err != 0 {
		panic("Error mapping slot")
	}
}

//go:nosplit
func (k *KVM) Unmap(start, size uintptr) {
	k.Machine.MemView.Toggle(false, start, size, commons.UNMAP_VAL)
}

// Returns with os thread locked.
//go:nosplit
func (k *KVM) SwitchToUser() {
	c := k.Machine.Get()
	opts := ring0.SwitchOpts{
		Registers:   &c.uregs,
		PageTables:  k.Machine.MemView.Tables,
		Flush:       true,
		FullRestore: true,
	}
	opts.Registers.Rip = bluepillretaddr //uint64(reflect.ValueOf(Bluepillret).Pointer())
	GetFs(&opts.Registers.Fs_base)       // making sure we get the correct FS value.
	// Now we are at the boundary were things should be stable.
	if runtime.Iscgo() && !k.Machine.MemView.ValidAddress(opts.Registers.Fs_base) {
		runtime.RegisterPthread(c.id)
	}
	commons.Check(k.Id != "")
	runtime.AssignSbId(k.Id, false)
	runtime.AssignVcpu(uintptr(unsafe.Pointer(c)))
	if !c.entered {
		c.SwitchToUser(opts)
		return
	}
	// The vcpu was already entered, we just return to it.
	bluepill(c)
}
