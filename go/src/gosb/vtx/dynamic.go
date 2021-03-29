package vtx

import (
	"gosb/commons"
	"gosb/globals"
	"gosb/vtx/platform/kvm"
	mv "gosb/vtx/platform/memview"
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

/* Implementation of the dynamic version of the vtx backend.
* We have some weird requirements due to the dynamicity of python that we try to
* account for here.
* */

var (
	ionce  sync.Once
	inside bool = false
)

func DInit() {
	// Delay the initialization to the first prolog.
}

func internalInit() {
	ionce.Do(func() {
		kvm.KVMInit()
		mv.InitializeGod()

		// Skip the init of sandboxes as we probably don't have them.
		mv.Views = make(map[commons.SandId]*mv.AddressSpace)

		// Initialize the kvm state.
		var err error
		kvmFd, err = os.OpenFile(_KVM_DRIVER_PATH, syscall.O_RDWR, 0)
		commons.Check(err == nil)
		err = kvm.UpdateGlobalOnce(int(kvmFd.Fd()))
		commons.Check(err == nil)
		machine = kvm.CreateVirtualMachine(int(kvmFd.Fd()), false)
		vm = &kvm.KVM{machine, nil, "God", 0}

		// Map the page allocator.
		mv.GodAS.MapArenas(false)
	})
}

func DProlog(id commons.SandId) {
	internalInit()
	commons.Check(mv.Views != nil)
	sb, ok := globals.Sandboxes[id]
	commons.Check(ok)
	mem, ok := mv.Views[id]
	if ok {
		goto entering
	}

	// We need to create the sandbox.
	commons.Check(mv.GodAS != nil)
	dynTryInHost(func() {
		mem = mv.GodAS.Copy(false)
		mem.ApplyDomain(sb)
		//TODO(aghosn) re-enable
		disablePkgs(mem, sb)
		mv.Views[sb.Config.Id] = mem
		machine.UpdateEPTSlots(func(start, size, gpa uintptr) {
			mv.GodAS.DefaultMap(start, size, gpa)
			for _, v := range mv.Views {
				v.DefaultMap(start, size, gpa)
			}
		})
		mem.MapArenas(false)
	})
	globals.DynEnd()
entering:
	dprolog(sb, mem)
}

//go:nosplit
func dprolog(sb *commons.SandboxMemory, mem *mv.AddressSpace) {
	commons.Check(mv.GodAS != nil)
	if !inside {
		prolog_internal(true)
		inside = true
	}
	vcpu := runtime.GetVcpu()
	kvm.RedSwitch(uintptr(mem.Tables.CR3(false, 0)))
	kvm.SetVCPUAttributes(vcpu, mem, &sb.Config.Sys)
}

//go:nosplit
func DEpilog(id commons.SandId) {
	vcpu := runtime.GetVcpu()
	commons.Check(inside)
	commons.Check(vcpu != 0)
	commons.Check(globals.DynGetPrevId != nil)
	// For the moment disallow nested sandboxes.
	commons.Check(globals.DynGetPrevId() == "GOD")
	kvm.Redpill(kvm.RED_GOD)
	kvm.SetVCPUAttributes(vcpu, mv.GodAS, &commons.SyscallAll)
}

func DRuntimeGrowth(isheap bool, id int, start, size uintptr) {
	if mv.GodAS == nil {
		return
	}
	size = uintptr(commons.Round(uint64(size), true))
	mem := &mv.MemoryRegion{}
	mv.GodAS.Extend(false, mem, uint64(start), uint64(size), commons.HEAP_VAL)
	for _, v := range mv.Views {
		cpy := &mv.MemoryRegion{}
		v.Extend2(cpy, mem)
	}
	if inside {
		vcpu := (*kvm.VCPU)(unsafe.Pointer(runtime.GetVcpu()))
		commons.Check(vcpu != nil && vcpu.Memview != nil)
	}
	// Register the address with KVM
	commons.Check(machine != nil)
	commons.Check(vm != nil && vm.Machine != nil)
	commons.Check(vm.Machine == machine)
	commons.Check(mem.Span.Slot == 0)
	var err syscall.Errno
	mem.Span.Slot, err = vm.Machine.DynSetEPTRegion(
		&mv.GodAS.NextSlot, mem.Span.GPA, mem.Span.Size, mem.Span.Start, 1)
	if err != 0 {
		panic("Error dynamically mapping slot")
	}
	commons.Check(mv.GodAS.PTEAllocator.Dirties == 0)
}

func DynTransfer(oldid, newid int, start, size uintptr) {
	commons.Check(oldid == -1)
	if mv.GodAS == nil {
		return
	}
	size = uintptr(commons.Round(uint64(size), true))
	mem := &mv.MemoryRegion{}
	mv.GodAS.Extend(false, mem, uint64(start), uint64(size), commons.HEAP_VAL)
	for sid, v := range mv.Views {
		if sb, ok := globals.Sandboxes[sid]; ok {
			if _, ok1 := sb.View[newid]; !ok1 {
				if inside {
					vcpu := (*kvm.VCPU)(unsafe.Pointer(runtime.GetVcpu()))
					// If we are not the ones mapping it, skip it.
					commons.Check(vcpu != nil && vcpu.Memview != nil)
					if vcpu.Memview != v {
						continue
					}
				}
			}
		}
		cpy := &mv.MemoryRegion{}
		v.Extend2(cpy, mem)
	}
	if inside {
		vcpu := (*kvm.VCPU)(unsafe.Pointer(runtime.GetVcpu()))
		commons.Check(vcpu != nil && vcpu.Memview != nil)
	}
	// Register the address with KVM
	commons.Check(machine != nil)
	commons.Check(vm != nil && vm.Machine != nil)
	commons.Check(vm.Machine == machine)
	commons.Check(mem.Span.Slot == 0)
	var err syscall.Errno
	mem.Span.Slot, err = vm.Machine.DynSetEPTRegion(
		&mv.GodAS.NextSlot, mem.Span.GPA, mem.Span.Size, mem.Span.Start, 1)
	if err != 0 {
		panic("Error dynamically mapping slot")
	}
	commons.Check(mv.GodAS.PTEAllocator.Dirties == 0)
}

/* Helper functions */

//go:nosplit
func dynTryInHost(f func()) {
	commons.Check(globals.DynGetId != nil)
	if !inside {
		f()
		return
	}
	// We are inside
	kvm.Redpill(kvm.RED_EXIT)
	runtime.AssignVcpu(0)
	inside = false
	f()
	prolog_internal(false)
	vcpu := (*kvm.VCPU)(unsafe.Pointer(runtime.GetVcpu()))
	commons.Check(vcpu != nil && vcpu.Memview != nil)
	inside = true
}

//go:nosplit
func DynToGod() {
	if !inside {
		return
	}
	kvm.Redpill(kvm.RED_GOD)
}

//go:nosplit
func DynGoBack() {
	if !inside {
		return
	}
	vcpu := (*kvm.VCPU)(unsafe.Pointer(runtime.GetVcpu()))
	commons.Check(vcpu != nil && vcpu.Memview != nil)
	kvm.RedSwitch(uintptr(vcpu.Memview.Tables.CR3(false, 0)))
}

//TODO(aghosn) should also fix access rights here
// disablePkgs removes all the pkgs that should not be available.
func disablePkgs(mem *mv.AddressSpace, sb *commons.SandboxMemory) {
	for _, pkg := range globals.AllPackages {
		prot := commons.UNMAP_VAL
		// Supposed to be there, leave it.
		if v, ok := sb.Config.View[pkg.Name]; ok && v == commons.DEF_VAL {
			continue
		} else if ok && v != commons.U_VAL {
			prot = v
		}
		i, err := globals.DynFindId(pkg.Name)
		if v, ok := sb.View[i]; ok && err == nil && v == commons.DEF_VAL {
			continue
		} else if ok && v != commons.U_VAL {
			prot = v
		}
		if prot != commons.UNMAP_VAL {
			prot |= commons.USER_VAL
		}
		//fmt.Printf("[%v]: (%x)\n", pkg.Name, prot)
		// Not supposed to be mapped.
		for _, s := range pkg.Sects {
			mem.ToggleDyn(true, uintptr(s.Addr), uintptr(s.Size), prot /*commons.UNMAP_VAL*/)
			//fmt.Printf("%x -- %x [%v]: 0x%x\n", s.Addr, s.Addr+s.Size, pkg.Name, prot)
		}
	}
}
