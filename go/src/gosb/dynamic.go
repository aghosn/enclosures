package gosb

import (
	"errors"
	"fmt"
	"gosb/backend"
	"gosb/commons"
	"gosb/globals"
	mv "gosb/vtx/platform/memview"
	"runtime"
	"strings"
)

var (
	P2PDeps     map[string][]string
	PNames      map[string]bool
	IdToName    map[int]string
	SandboxDeps map[string][]string

	// Go tries to hold information about the address space.
	InitialSpace *commons.VMAreas
	//GodSpace     *commons.VMAreas

)

func DynInitialize(back backend.Backend) {
	once.Do(func() {
		runtime.GOMAXPROCS(1)
		commons.Check(globals.DynGetId != nil)
		globals.IsDynamic = true
		globals.IdToPkg = make(map[int]*commons.Package)
		globals.NameToPkg = make(map[string]*commons.Package)
		globals.NameToId = make(map[string]int)
		globals.Sandboxes = make(map[commons.SandId]*commons.SandboxMemory)
		globals.Configurations = make([]*commons.SandboxDomain, 0)
		P2PDeps = make(map[string][]string)
		PNames = make(map[string]bool)
		IdToName = make(map[int]string)
		SandboxDeps = make(map[string][]string)
		fvmas := mv.ParseProcessAddressSpace(commons.USER_VAL)
		InitialSpace = commons.Convert(fvmas)
		//GodSpace = InitialSpace.Copy()
		// Do the normal backend initialization and see what happens.
		initBackend(back)
	})
}

func DynAddDependency(current, dependency string) {
	commons.Check(P2PDeps != nil)
	commons.Check(current != "")
	commons.Check(dependency != "")
	entry, _ := P2PDeps[current]
	P2PDeps[current] = append(entry, dependency)
	PNames[current] = true
	PNames[dependency] = true
}

func DynRegisterId(name string, id int) {
	commons.Check(name != "")
	PNames[name] = true
	if prev, ok := IdToName[id]; ok {
		// Nested package.
		if prev != "module" {
			_, ok1 := globals.IdToPkg[id]
			commons.Check(ok1)
		}
		//commons.Check(prev == "module")
	}
	IdToName[id] = name

	// Get or create the package.
	pkg, ok := globals.IdToPkg[id]
	if !ok {
		pkg = &commons.Package{
			Name: name,
			Id:   id,
		}
		globals.AllPackages = append(globals.AllPackages, pkg)
	}
	commons.Check(pkg != nil)
	globals.IdToPkg[id] = pkg
	if name != "module" {
		globals.NameToId[name] = id
		globals.NameToPkg[name] = pkg
		pkg.Name = name
	}
}

func DynRegisterSandbox(id, mem, sys string) {
	m, _, err := commons.ParseMemoryView(mem)
	s, err1 := commons.ParseSyscalls(sys)
	commons.Check(err == nil)
	commons.Check(err1 == nil)
	if e, ok := globals.Sandboxes[id]; ok {
		commons.Check(e.Config != nil)
		return
	}

	memview := make(map[string]uint8)
	for _, v := range m {
		memview[v.Name] = v.Perm
	}

	// Adding the heap vals
	for v, _ := range commons.PythonRuntime {
		if _, ok := memview[v]; ok {
			continue
		}
		memview[v] = commons.D_VAL
	}
	// Compute and translate the memview.
	//memview = globals.ComputeMemoryView(deps, P2PDeps, memview)
	// TODO compute the view.
	config := &commons.SandboxDomain{
		Id:       id,
		Func:     "a fake func",
		Sys:      s,
		View:     memview, //TODO complete the view
		Pkgs:     nil,     //TODO complete the pkgs
		Pristine: false,
	}
	globals.Configurations = append(globals.Configurations, config)

	sb := &commons.SandboxMemory{
		Static:  nil, //TODO Compute this.
		Config:  config,
		View:    nil,
		Entered: false,
	}
	globals.Sandboxes[id] = sb
}

func DynRegisterSandboxDependency(id, pkg string) {
	e, _ := SandboxDeps[id]
	SandboxDeps[id] = append(e, pkg)
}

func findFullName(pkg string) (string, error) {
	// Fast path
	if _, ok := globals.NameToId[pkg]; ok {
		return pkg, nil
	}

	// Slow path
	for k, _ := range globals.NameToId {
		if strings.HasSuffix(k, fmt.Sprintf(".%s", pkg)) {
			return k, nil
		}
	}

	// Couldn't find it inside the Id, look in deps
	for k, _ := range P2PDeps {
		if strings.HasSuffix(k, fmt.Sprintf(".%s", pkg)) {
			return k, nil
		}
	}
	return pkg, errors.New(fmt.Sprintf("Unable to find a full name for %s", pkg))
}

func DynProlog(id string) {
	sb, ok := globals.Sandboxes[id]
	commons.Check(ok)
	// Already did the init dance
	if sb.Entered {
		currBackend.Prolog(id)
		return
	}

	globals.DynStart()
	// Compute full names for deps
	memview := sb.Config.View
	commons.Check(memview != nil)
	cleanupDeps(id)
	d, ok := SandboxDeps[id]
	//commons.Check(ok)
	deps := make([]string, len(d))
	for i, v := range d {
		n, err := findFullName(v)
		commons.Check(err == nil)
		deps[i] = n
		if p, ok := memview[v]; ok {
			delete(memview, v)
			memview[n] = p
		}
	}
	// Apparently we're missing dependencies, let's fix that.
	fixDependencies()

	// We need to update the view.
	sb.Config.View = globals.ComputeMemoryView(deps, P2PDeps, memview)
	mmv := make(map[int]uint8)
	//fmt.Println("The view of that sandbox: ", sb.Config.View)
	//fmt.Println("The deps", P2PDeps)
	for k, v := range sb.Config.View {
		i, e := globals.DynFindId(k)
		if e != nil {
			//fmt.Println(e)
			continue
		}
		commons.Check(e == nil)
		mmv[i] = v
	}

	// TODO(aghosn) remove afterwards, this is for debugging.
	for k, _ := range commons.PythonFix {
		i, _ := globals.DynFindId(k)
		mmv[i] = commons.D_VAL
	}
	commons.Check(sb.View == nil)
	sb.View = mmv

	globals.DynRegisterRef(id, mmv)
	// Create the package list too.
	for k, _ := range sb.View {
		n, ok := globals.IdToPkg[k]
		commons.Check(ok)
		sb.Config.Pkgs = append(sb.Config.Pkgs, n.Name)
	}

	sb.Entered = true
	vmareas := make([]*commons.VMArea, 0)
	// Compute the static
	for k, v := range sb.Config.View {
		pkg, ok := globals.NameToPkg[k]
		if !ok {
			continue
		}
		vmareas = append(vmareas, commons.PackageToVMAreas(pkg, v)...)
	}

	// Map the InitialSpace too, this corresponds to runtime.
	sb.Static = commons.Convert(vmareas)
	sb.Static.MapAreaCopy(InitialSpace)

	// Call the actual backend.
	commons.Check(currBackend != nil)
	currBackend.Prolog(id)
}

func DynEpilog(id commons.SandId) {
	_, ok := globals.Sandboxes[id]
	commons.Check(ok)
	commons.Check(globals.DynGetId != nil)
	commons.Check(globals.DynGetId() == id)
	currBackend.Epilog(id)
}

func ExtendSpace(isrt bool, addr, size uintptr) {
	commons.Check(InitialSpace != nil)
	commons.Check(isrt)
	vma := &commons.VMArea{}
	vma.Addr, vma.Size, vma.Prot = uint64(addr), uint64(size), commons.HEAP_VAL
	//TODO these updates are not reflected by the backend.
	//GodSpace.Map(vma)
	commons.Check(currBackend != nil)
	currBackend.RuntimeGrowth(isrt, 0, addr, size)
	if isrt {
		cpy := vma.Copy()
		InitialSpace.Map(cpy)
		for _, sb := range globals.Sandboxes {
			sb.Static.Map(vma.Copy())
		}
	}
}

// This should never be called before registerid
func DynAddSection(id int, start, size uintptr) {
	pkg, ok := globals.IdToPkg[id]
	commons.Check(ok && pkg != nil)
	//fmt.Printf("%x -- %x [%v]\n", start, start+size, id)
	pkg.AddSection(uint64(start), uint64(size), commons.HEAP_VAL)
	if currBackend.Transfer != nil {
		currBackend.Transfer(-1, id, start, size)
	}
}

// Fixing dependencies
func fixDependencies() {
	for n, _ := range PNames {
		for n2, _ := range PNames {
			if strings.HasPrefix(n2, fmt.Sprintf("%s.", n)) {
				prev := P2PDeps[n]
				P2PDeps[n] = append(prev, n2)
			}
		}
	}
}

//cleaning up dependencies.
//For the moment we're not good enough in Python internals to
//correctly register dependencies. As a result, we get confused by
//variables and package invocations.
func cleanupDeps(id string) {
	d, ok := SandboxDeps[id]
	// Nothing to do
	if !ok {
		return
	}
	deps := make([]string, 0)
	for _, v := range d {
		n, err := findFullName(v)
		if err != nil {
			// This is probably an issue.
			continue
		}
		deps = append(deps, n)
	}
	SandboxDeps[id] = deps
}
