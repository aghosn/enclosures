package ld

import (
	"cmd/link/internal/objfile"
	"cmd/link/internal/sym"
	"encoding/json"
	lb "gosb/commons"
	"log"
	"sort"
	"strings"
)

var (
	nonbloat *lb.Package
	//	stmps     *lb.Package
	Bloats    map[string]*lb.Package
	toSym     map[*lb.Section][]*sym.Symbol
	lookup    map[int]string
	domains   []*lb.SandboxDomain
	sectNames = []string{".fake", ".bloated", ".sandboxes"}
)

// computeBloats initializes global state and computes all dependencies for each
// package that requires to be bloated.
func (ctxt *Link) computeBloats() {
	if Bloats == nil {
		Bloats = make(map[string]*lb.Package)
	}
	if toSym == nil {
		toSym = make(map[*lb.Section][]*sym.Symbol)
	}
	// Build a reverse map for names and ids
	lookup = make(map[int]string)
	for k, v := range ctxt.PackageDecl {
		lookup[v] = k
	}
	// Define the function for transitive dependencies
	check := func(s string) bool {
		_, ok := Bloats[s]
		return ok
	}
	create := func(ctxt *Link, id int, deps []int) {
		pkg, ok := lookup[id]
		if !ok {
			log.Fatalf("No name for id %v\n", id)
		}
		pkgInfo := &lb.Package{Name: pkg, Id: id, Sects: make([]lb.Section, sym.SABIALIAS)}
		// Initialize protections
		for i := range pkgInfo.Sects {
			pkgInfo.Sects[i].Prot = symKindtoProt(sym.SymKind(i))
		}
		Bloats[pkg] = pkgInfo
	}

	// Get the transitive dependencies for each package
	ctxt.registerExtraPackages()
	for k := range objfile.SegregatedPkgs {
		ctxt.gosb_walkTransDeps(k, create, check)
	}
	// Add an entry for non-bloated packages, and shared stmps
	nonbloat = &lb.Package{lb.TrustedPkgName, -1, make([]lb.Section, sym.SABIALIAS), nil}

	for i := range nonbloat.Sects {
		nonbloat.Sects[i].Prot = symKindtoProt(sym.SymKind(i))
	}
	// For all the sandboxes, we get the transitive dependencies & generate
	// the sandboxes informations.
	ctxt.gosb_generateDomains()
}

// addExtraPackages registers packages that are not sandbox dependencies
// but that we still want to bloat.
func (ctxt *Link) registerExtraPackages() {
	if objfile.SegregatedPkgs != nil {
		objfile.SegregatedPkgs["gosb"] = true
		// cgo does not track dependencies well. maybe relocate into runtime.
		objfile.SegregatedPkgs["runtime/cgo"] = true
		objfile.SegregatedPkgs["runtime/cgo2"] = true
		if _, ok := ctxt.PackageDecl["runtime/cgo"]; ok {
			Bloats["runtime/cgo2"] = &lb.Package{"runtime/cgo2", -3, make([]lb.Section, sym.SABIALIAS), nil}
		}
	}
}

func (ctxt *Link) extraDependencies() []string {
	res := make([]string, len(lb.ExtraDependencies))
	copy(res, lb.ExtraDependencies)
	if _, ok := ctxt.PackageDecl["runtime/cgo"]; ok {
		res = append(res, "runtime/cgo")
		res = append(res, "runtime/cgo2")
	}
	return res
}

func (ctxt *Link) gosb_generateDomains() {
	for _, v := range objfile.Sandboxes {
		sb := &lb.SandboxDomain{}
		sb.Id = v.Id
		sb.Func = v.Func
		sb.Pristine = v.Pristine
		var err error
		sb.Sys, err = lb.ParseSyscalls(v.Sys)
		sb.View = nil
		if err != nil {
			log.Fatalf("Error parsing %v: %v\n", v.Sys, err.Error())
		}
		visited := make(map[string]*lb.Package)
		// No op, we don't have to do anything
		f := func(ctxt *Link, id int, deps []int) {}
		// Have we visited that node before?
		c := func(s string) bool {
			if _, ok := visited[s]; ok {
				return true
			}
			pack, ok := Bloats[s]
			if !ok && (s == "go.runtime" || s == "go.itab") {
				panic("We said we ignored go.runtime and go.itab")
			}
			if !ok {
				log.Fatalf("Error '%v' should have a package by now.\n", s)
			}
			visited[s] = pack
			return false
		}
		// Add the packages from the view here if we want transitive deps.
		for _, p := range v.Extras {
			v.Packages = append(v.Packages, p.Name)
		}
		v.Packages = append(v.Packages, ctxt.extraDependencies()...)
		for _, p := range v.Packages {
			if p == "go.itab" || p == "go.runtime" {
				panic("go.itab and go.runtime should not be here")
			}
			ctxt.gosb_walkTransDeps(p, f, c)
		}
		// Handle the extras and their permissions!
		memView := make(map[string]uint8)
		for _, p := range v.Extras {
			memView[p.Name] = p.Perm
		}
		// Finally, we set the packages and the memory view
		for _, pack := range visited {
			sb.Pkgs = append(sb.Pkgs, pack.Name)
		}
		sb.View = memView
		domains = append(domains, sb)
	}
	// Create a fake sandbox for the nonbloated domain
	nonbloatDomain := &lb.SandboxDomain{}
	nonbloatDomain.Id = "-1"
	nonbloatDomain.Func = "-1"
	nonbloatDomain.Sys = [lb.BITMASK_LEN]uint64{0, 0, 0, 0, 0}
	nonbloatDomain.View = nil
	nonbloatDomain.Pkgs = []string{nonbloat.Name}
	domains = append(domains, nonbloatDomain)
}

// gosb_walkTransDeps allows to follow transitive dependencies applying the given f method.\
// It is used to 1) generate the list of packages to bloat, and 2) to find all dependencies
// for sandboxes.
func (ctxt *Link) gosb_walkTransDeps(top string, f func(ctxt *Link, id int, deps []int), check func(s string) bool) {
	// We check that the package has a decl
	// If it does not, it is probably a fake package that is part of the runtime.
	// Ids in the following steps will correspond to runtime so we're fine.
	// TODO(aghosn) this prevents type and go.itab, go.runtime from being added
	// to the deps... Let's see later if there is a problem.
	id, ok := ctxt.PackageDecl[top]
	if !ok && top == "type" {
		return
	}
	// Call the check
	if check(top) {
		return
	}
	// Handle the entry
	deps, ok := ctxt.PackageDeps[id]
	f(ctxt, id, deps)

	for _, v := range deps {
		name, ok := lookup[v]
		if !ok {
			log.Fatalf("Missing name for package %v\n\n%v\n", v, lookup)
		}
		ctxt.gosb_walkTransDeps(name, f, check)
	}
}

// gosb_reorderSymbols sorts symbols per package, puts all the bloated packages
// after the non bloated ones. This function keeps information about the non-bloated
// part as well.
// Sandboxes symbols are put at the very end of things.
// We also have to handle the sandbox information.
func gosb_reorderSymbols(sel int, syms []*sym.Symbol) []*sym.Symbol {
	// Fast exit if we do not have sandboxes or if it is a section we don't care about
	if len(objfile.Sandboxes) == 0 || ignoreSection(sel) || len(syms) == 0 {
		return syms
	}

	// We divide symbols into bloated per package, unbloated lists, and sandbox
	// symbols.
	regSyms := make([]*sym.Symbol, 0)
	bloated := make(map[string][]*sym.Symbol)
	sandSyms := make([]*sym.Symbol, 0)
	specials := make([]*sym.Symbol, 0)
	for _, s := range syms {
		// Safety check to avoid go.itab and go.runtime
		if s.File == "go.runtime" || s.File == "go.itab" {
			panic("We have a symbol that belongs to go.[itab|runtime]")
		}
		// Sandbox symbol itself needs to be seggragated
		if _, ok := objfile.SBMap[s.Name]; ok {
			sandSyms = append(sandSyms, s)
			s.Align = 0x1000
		} else if isSandboxStkObj(s.Name, s) || s.Name == "main.main.stkobj" {
			// Isolate stack object for sandbox code
			sandSyms = append(sandSyms, s)
			s.Align = 0x1000
			if s.Size < 0x1000 {
				s.Size = 0x1000
			}
		} else if _, ok := Bloats[s.File]; ok {
			e, ok1 := bloated[s.File]
			if !ok1 {
				e = make([]*sym.Symbol, 0)
			}
			bloated[s.File] = append(e, s)
		} else if s.Name == "runtime.pclntab" {
			s.Align = 0x1000
			specials = append(specials, s)
		} else {
			regSyms = append(regSyms, s)
		}
	}
	// We sort the two according to packages
	fmap := make([][]*sym.Symbol, 0)
	for k, v := range bloated {
		if v == nil {
			continue
		}
		// We register the package here cause we'll need the symbol later.
		if b, ok := Bloats[k]; ok {
			toSym[&b.Sects[sel]] = v
		} else {
			log.Fatalf("Unable to find the package for %v\n", k)
		}
		fmap = append(fmap, v)
	}
	// We sort the bloated packages
	sort.Slice(fmap, func(i, j int) bool {
		return strings.Compare(fmap[i][0].File, fmap[j][0].File) == -1
	})

	// We register the regsyms as well for the nonbloated.
	toSym[&nonbloat.Sects[sel]] = regSyms
	// Align symbols
	for _, s := range fmap {
		s[0].Align = 0x1000
		regSyms = append(regSyms, s...)
	}
	regSyms = append(regSyms, specials...)
	regSyms = append(regSyms, sandSyms...)
	return regSyms
}

func gosb_generateContent(sect string) []byte {
	switch sect {
	case ".fake":
		return []byte("fake")
	case ".bloated":
		return gosb_dumpPackages()
	case ".sandboxes":
		return gosb_dumpSandboxes()
	default:
		panic("Unknown value for gosb_generateContent")
	}
	return nil
}

func isSandboxStkObj(name string, s *sym.Symbol) bool {
	if !strings.Contains(name, ".stkobj") {
		return false
	}
	split := strings.Split(name, ".stkobj")
	if len(split) == 0 {
		return false
	}
	_, ok := objfile.SBMap[split[0]]
	return ok
}

// gosb_dumpPackages returns the marshalled json bytes that correspond
// to packages. we go through each register section to set the addresses.
func gosb_dumpPackages() []byte {
	// Register the final addresses for the sections.
	// This actually also handles the non-bloat part.
	for k, v := range toSym {
		if len(v) == 0 {
			continue
		}
		first := v[0]
		last := v[len(v)-1]
		if first.Value > last.Value {
			log.Fatalf("Symbols not ordered (%v) %v-%v\n", v, first.Value, last.Value)
		}
		_, ok := Bloats[first.File]
		// The symbol is part of a bloat
		if ok && (first.Align != 0x1000 || first.Value%0x1000 != 0) {
			log.Fatalf("Wrong alignment for bloated section %v: %v - %x| - %x\n", first.File, first, first.Align, first.Value)
		}
		// We just verify that symbols are increasing and all belong to the same
		// package.
		gosb_verifySymbols(v, ok)
		k.Addr = uint64(first.Value)
		k.Size = uint64(last.Value-first.Value) + uint64(last.Size)
		k.Prot = lb.W_VAL | lb.R_VAL
		if first.Sect != nil {
			k.Prot = first.Sect.Rwx
		}
	}
	// Create the dump for the bloated packages
	res := make([]*lb.Package, 0)
	for _, b := range Bloats {
		res = append(res, b)
	}
	// Add the non-bloated part
	res = append(res, nonbloat)
	b, err := json.Marshal(res)
	if err != nil {
		log.Fatalf("Unable to marshal %v\n", err.Error())
	}
	return b
}

func gosb_verifySymbols(syms []*sym.Symbol, aligned bool) {
	if len(syms) == 0 {
		return
	}
	for i := range syms {
		if i == 0 && aligned && syms[i].Value%0x1000 != 0 {
			log.Fatalf("Wrong alignment for %v: %v\n", syms[i].File, syms[i].Value)
		} else if i == 0 {
			continue
		}
		prev := syms[i-1]
		if prev.Value+prev.Size > syms[i].Value && syms[i].Outer != prev {
			log.Fatalf("Not ordered properly %v -- %v [%v]\n", prev.Value+prev.Size, syms[i].Value, prev.File)
		}
		if aligned && prev.File != syms[i].File {
			log.Fatalf("Different packages %v -- %v\n", prev.File, syms[i].File)
		}
	}
}

func gosb_dumpSandboxes() []byte {
	res, err := json.Marshal(domains)
	if err != nil {
		log.Fatalf("Error mashalling sandboxes %v\n", err.Error())
	}
	return res
}

// Translate a section's idx into protection
func symKindtoProt(s sym.SymKind) uint8 {
	prot := lb.R_VAL
	// executable
	if s == sym.STEXT || s == sym.SELFRXSECT {
		prot |= lb.X_VAL
		return prot
	}
	// read-only
	if s >= sym.STYPE && s <= sym.SPCLNTAB {
		// nothing to do
		return prot
	}
	// writable
	if s >= sym.SFirstWritable && s <= sym.SHOSTOBJ {
		prot |= lb.W_VAL
		return prot
	}
	// Debugging, TODO(aghosn) what should we do?
	return prot
}

// fixingStupidSymbols Go introduces symbols at link-time that act as aggregators
// for in-package symbols and are then indexed by the runtime.
// This renders bloating extremely difficult. As we result, we fix the link
// hacks by associating all these symbols with the runtime, and making sure
// the Outer symbol has the correct size (instead of 0, as specified by default
// go).
// Apparently they even fuck up the sub thing.... motherfuckers.
func (ctxt *Link) fixingStupidSymbols() {
	if !HasSandboxes() {
		return
	}
	for _, s := range ctxt.Syms.Allsym {
		// One of the blacklisted symbols introduced by the linker.
		_, contains := lb.SymToFix[s.Name]
		nopackage := s.File == ""
		noouter := s.Outer == nil
		if contains && nopackage && noouter {
			s.File = "runtime"
			continue
		}

		// stmp_* fuckers that are not made available by static analysis.
		// we relocate them inside the fake stmpPkg
		_, bloated := objfile.SegregatedPkgs[s.File]
		if !bloated && isStaticTemp(s.Name) {
			s.File = "runtime" //lb.StmpPkgName
		}

		// Some cgo symbols have missing packages.
		if strings.HasPrefix(s.Name, "runtime/cgo") && s.File == "" {
			s.File = "runtime/cgo"
		}

		// Missing symbols.
		if (strings.HasPrefix(s.Name, "$f64") || strings.HasPrefix(s.Name, "$f32")) && s.File == "" {
			s.File = "runtime"
		}

		// Make got plt part of the runtime.
		if s.Name == ".got.plt" {
			s.File = "runtime"
		}

		// Some symbols reference outer without being subs.
		if s.Outer != nil {
			if _, relocate := lb.SymToFix[s.Outer.Name]; relocate {
				s.File = "runtime"
			} else if strings.HasPrefix(s.Name, "x_cgo") && s.File == "" {
				s.File = "runtime/cgo"
			}
			continue
		}

		// itab symbols need to be made available too
		if /*!bloated &&*/ noouter && strings.HasPrefix(s.Name, "go.itab.") {
			s.File = "runtime"
		}
	}
}

func (ctxt *Link) StupidText() {
	// Fixing cgo functions
	for _, s := range ctxt.Textp {
		if s.File == "" && (strings.HasPrefix(s.Name, "runtime/cgo(.text") ||
			(s.Outer != nil && strings.HasPrefix(s.Outer.Name, "runtime/cgo(.text"))) {
			s.File = "runtime/cgo2"
		}
	}
}
