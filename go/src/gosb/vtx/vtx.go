package vtx

import (
	"fmt"
	"gosb/commons"
	"gosb/globals"
	"gosb/vtx/platform/kvm"
	mv "gosb/vtx/platform/memview"
	"os"
	"runtime"
	"sync"
	"syscall"
)

const (
	_KVM_DRIVER_PATH = "/dev/kvm"
	_OUT_MODE        = ""
	_FUCK_MODE       = "fuck"
)

var (
	once  sync.Once
	kvmFd *os.File
	//views   map[commons.SandId]*mv.AddressSpace
	machine *kvm.Machine
	vm      *kvm.KVM
)

func Init() {
	// Set the signal handler and the god view.
	kvm.KVMInit()
	mv.InitializeGod()

	// Create all the sandboxes memory views
	mv.Views = make(map[commons.SandId]*mv.AddressSpace)
	for _, d := range globals.Sandboxes {
		// Skip over the non-sandbox
		if d.Config.Id == "-1" {
			continue
		}
		mem := mv.GodAS.Copy(false)
		mem.ApplyDomain(d)
		mv.Views[d.Config.Id] = mem
	}

	// Initialize the kvm state.
	var err error
	kvmFd, err = os.OpenFile(_KVM_DRIVER_PATH, syscall.O_RDWR, 0)
	commons.Check(err == nil)
	err = kvm.UpdateGlobalOnce(int(kvmFd.Fd()))
	commons.Check(err == nil)
	machine = kvm.CreateVirtualMachine(int(kvmFd.Fd()), true)
	vm = &kvm.KVM{machine, nil, "God", 0}

	// Map the page allocator.
	mv.GodAS.MapArenas(true)
	for _, v := range mv.Views {
		v.MapArenas(true)
	}
}

//go:nosplit
func Prolog(id commons.SandId) {
	vcpu := runtime.GetVcpu()
	commons.Check(!runtime.IsG0())
	//TODO add to the stack of sandboxes.
	v, ok := mv.Views[id]
	s, ok1 := globals.Sandboxes[id]
	commons.Check(ok && ok1)
	if vcpu != 0 {
		goto end
	}
	prolog_internal(true)
	vcpu = runtime.GetVcpu()
end:
	commons.Check(vcpu != 0)
	kvm.RedSwitch(uintptr(v.Tables.CR3(false, 0)))
	kvm.SetVCPUAttributes(vcpu, v, &s.Config.Sys)
	runtime.AssignSbId(id, false)
}

//go:nosplit
func Epilog(id commons.SandId) {
	sid := runtime.GetmSbIds()
	commons.Check(!runtime.IsG0())
	commons.Check(runtime.GetVcpu() != 0)
	commons.Check(sid == id)
	//TODO use the stack to know where to return instead.
	kvm.Redpill(kvm.RED_GOD)
	runtime.AssignSbId(_OUT_MODE, true)
}

//go:nosplit
func Execute(id commons.SandId) {
	sid := runtime.GetmSbIds()
	vcpu := runtime.GetVcpu()
	commons.Check(sid == _OUT_MODE || vcpu != 0)
	// need to bail for a clone.
	if id == _FUCK_MODE {
		if vcpu != 0 {
			kvm.Redpill(kvm.RED_EXIT)
		}
		runtime.AssignSbId(_OUT_MODE, false)
		runtime.AssignVcpu(0)
		return
	}

	// Already in the correct context, continue
	if sid == id {
		return
	}

	mem, ok := mv.Views[id]
	filter, ok1 := globals.Sandboxes[id]
	commons.Check((ok && ok1) || id == _OUT_MODE)

	if vcpu == 0 && id != _OUT_MODE {
		commons.Check(sid == _OUT_MODE)
		prolog_internal(false)
	}
	vcpu = runtime.GetVcpu()
	commons.Check(vcpu != 0)

	// Inside the VM
	if id == _OUT_MODE {
		kvm.Redpill(kvm.RED_GOD)
		kvm.SetVCPUAttributes(vcpu, mv.GodAS, &commons.SyscallAll)
	} else {
		kvm.RedSwitch(uintptr(mem.Tables.CR3(false, 0)))
		kvm.SetVCPUAttributes(vcpu, mem, &filter.Config.Sys)
	}
	runtime.AssignSbId(id, false)
}

//go:nosplit
func Register(id int, start, size uintptr) {
	panic("Called")
}

//go:nosplit
func Transfer(oldid, newid int, start, size uintptr) {
	if oldid == newid {
		throw("Useless transfer")
	}
	lunmap, ok := globals.PkgDeps[oldid]
	lmap, ok1 := globals.PkgDeps[newid]
	if !(ok || ok1) {
		return
	}
	mv.GodMu.Lock()
	if ok {
		for _, u := range lunmap {
			if view, ok2 := mv.Views[u]; ok2 {
				view.Toggle(false, start, size, commons.UNMAP_VAL)
			}
		}
	}

	if ok1 {
		for _, u := range lmap {
			if view, ok2 := mv.Views[u]; ok2 {
				sand, ok3 := globals.Sandboxes[u]
				commons.Check(ok3)
				prot, ok3 := sand.View[newid]
				commons.Check(ok3)
				view.Toggle(true, start, size, prot&commons.HEAP_VAL)
			}
		}
	}
	mv.GodMu.Unlock()
}

//go:nosplit
func prolog_internal(replenish bool) {
	if replenish {
		vm.Machine.Replenish()
	}
	vm.SwitchToUser()
	runtime.UnlockOSThread()
	// From here, we made the switch to the VM
	return
}

// RuntimeGrowth extends the runtime memory.
// @warning cannot do dynamic allocation
//go:nosplit
func RuntimeGrowth(isheap bool, id int, start, size uintptr) {
	size = uintptr(commons.Round(uint64(size), true))
	lmap, ok := globals.PkgDeps[id]
	mv.GodMu.Lock()
	mem := mv.GodAS.AcquireEMR()
	mv.GodAS.Extend(isheap, mem, uint64(start), uint64(size), commons.HEAP_VAL)
	if !ok {
		goto end
	}
	for _, m := range lmap {
		if v, ok1 := mv.Views[m]; ok1 {
			v.ExtendRuntime(mem)
		}
	}

end:
	mv.GodMu.Unlock()
}

/* Helper functions */
// All the updates we might have missed
func UpdateAll() {
	for v := commons.ToVMA(mv.Updates.First); v != nil; v = commons.ToVMA(v.Next) {
		isheap := runtime.IsThisTheHeap(uintptr(v.Addr))
		RuntimeGrowth(isheap, 0, uintptr(v.Addr), uintptr(v.Size))
	}
}

// tryRedpill exits the VM iff we are in a VM.
// It returns true if we were in a VM, and the current sbid.
//
//go:nosplit
func tryRedpill() (bool, string) {
	msbid := runtime.GetmSbIds()
	vcpu := runtime.GetVcpu()
	if vcpu == 0 {
		commons.Check(msbid == _OUT_MODE)
		return false, msbid
	}
	kvm.Redpill(kvm.RED_EXIT)
	runtime.AssignSbId(_OUT_MODE, false)
	runtime.AssignVcpu(0)
	return true, msbid
}

// tryBluepill tries to return to the provided sandbox if do is true
// and id is not empty.
//
//go:nosplit
func tryBluepill(do bool, id string) {
	if !do || id == "" {
		return
	}
	v, ok := mv.Views[id]
	f, ok1 := globals.Sandboxes[id]
	vcpu := runtime.GetVcpu()
	commons.Check(ok && ok1 && vcpu != 0)
	prolog_internal(false)
	inside = true
	kvm.RedSwitch(uintptr(v.Tables.CR3(false, 0)))
	kvm.SetVCPUAttributes(vcpu, v, &f.Config.Sys)
	runtime.AssignSbId(id, false)

}

//go:nosplit
func tryInHost(f func()) {
	do, msbid := tryRedpill()
	inside = false
	f()
	tryBluepill(do, msbid)
}

//go:nosplit
func VTXEntry(do bool, id string) {
	tryBluepill(do, id)
}

//go:nosplit
func VTXExit() (bool, string) {
	return tryRedpill()
}

func Stats() {
}

func UpdateMissing() {
	if mv.GodAS == nil {
		return
	}
	tryInHost(func() {
		areas := mv.ParseProcessAddressSpace(commons.USER_VAL)
		for _, v := range areas {
			if !mv.GodAS.ContainsRegion(uintptr(v.Addr), uintptr(v.Size)) {
				mv.GodAS.Replenish()
				mem := mv.GodAS.AcquireEMR()
				mv.GodAS.Extend(false, mem, v.Addr, v.Size, v.Prot)
				for _, view := range mv.Views {
					view.Replenish()
					view.ExtendRuntime(mem)
				}
			}
		}
		fmt.Println("Do we have dirties?", mv.GodAS.PTEAllocator.Dirties)
	})
}
