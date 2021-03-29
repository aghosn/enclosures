package mpk

/*
* @author: CharlyCst, aghosn
*
 */

import (
	"errors"
	// "fmt"
	c "gosb/commons"
	g "gosb/globals"
	"runtime"
)

var (
	sbPKRU  map[c.SandId]PKRU
	pkgKeys map[int]Pkey

	// Statistics
	entries uint64
	exits   uint64
	escapes uint64
)

// MStart initializes PKRU of new threads
func MStart() {
	WritePKRU(AllRightsPKRU)
}

// Execute turns on sandbox isolation
//go:nosplit
func Execute(id c.SandId) {
	cid := runtime.GetmSbIds()
	if id == "" {
		if cid != id {
			escapes++
		}
		WritePKRU(AllRightsPKRU)
		runtime.AssignSbId(id, false)
		return
	}
	pkru, ok := sbPKRU[id]
	if !ok {
		println("[MPK BACKEND]: Could not find pkru ", id)
		return
	}
	entries++
	WritePKRU(pkru)
	runtime.AssignSbId(id, false)
}

// Prolog initialize isolation of the sandbox
//go:nosplit
func Prolog(id c.SandId) {
	pkru, ok := sbPKRU[id]
	if !ok {
		println("[MPK BACKEND]: Sandbox PKRU not found in prolog")
		return
	}
	entries++
	runtime.AssignSbId(id, false)
	WritePKRU(pkru)
}

// Epilog is called at the end of the execution of a given sandbox
//go:nosplit
func Epilog(id c.SandId) {
	runtime.AssignSbId("", true)
	// Clean PKRU
	WritePKRU(AllRightsPKRU)
	exits++
}

// Register a page for a given package
//go:nosplit
func Register(id int, start, size uintptr) {
	if id == 0 || id == -1 { // Runtime
		return
	}

	key, ok := pkgKeys[id]
	if !ok {
		println("[MPK BACKEND]: Register key not found")
		return
	}
	PkeyMprotect(start, uint64(size), SysProtRW, key)
}

// Transfer a page from one package to another
//go:nosplit
func Transfer(oldid, newid int, start, size uintptr) {
	if oldid == newid {
		return
	}

	if newid == 0 { // Runtime
		PkeyMprotect(start, uint64(size), SysProtRW, 0)
		return
	}
	key, ok := pkgKeys[newid]
	if !ok {
		return
	}
	oldKey, ok := pkgKeys[oldid]
	if !ok || oldKey != key {
		PkeyMprotect(start, uint64(size), SysProtRW, key)
	}
}

// allocateKey allocates MPK keys and tag sections with those keys
func allocateKey(sandboxKeys map[c.SandId][]int, pkgGroups [][]int) []Pkey {
	keys := make([]Pkey, 0, len(pkgGroups))
	for _, pkgGroup := range pkgGroups {
		key, err := PkeyAlloc()
		if err != nil {
			panic(err)
		}
		keys = append(keys, key)

		for _, pkgID := range pkgGroup {
			tagPackage(pkgID, key)
		}
	}

	return keys
}

func tagPackage(id int, key Pkey) {
	pkg, ok := g.IdToPkg[id]
	if !ok {
		panic(errors.New("Package not found"))
	}

	for _, section := range pkg.Sects {
		if section.Addr != 0 && section.Size > 0 {
			sysProt := getSectionProt(section)
			PkeyMprotect(uintptr(section.Addr), section.Size, sysProt, key)
		}
	}
}

func getSectionProt(section c.Section) SysProt {
	prot := SysProtR
	if section.Prot&c.W_VAL > 0 {
		prot = prot | SysProtRW
	}
	if section.Prot&c.X_VAL > 0 {
		prot = prot | SysProtRX
	}
	return prot
}

func getGroupProt(p int)

// computePKRU initializes `sbPKRU` with corresponding PKRUs
func computePKRU(sandboxKeys map[c.SandId][]int, sandboxProt map[c.SandId][]Prot, keys []Pkey) {
	sbPKRU = make(map[c.SandId]PKRU, len(sandboxKeys))
	for sbID, keyIDs := range sandboxKeys {
		sbProts := sandboxProt[sbID]
		pkru := NoRightsPKRU
		for idx, keyID := range keyIDs {
			key := keys[keyID]
			prot := sbProts[idx]
			pkru = pkru.Update(key, prot)
		}
		sbPKRU[sbID] = pkru
	}
}

// Init relies on domains and packages, they must be initialized before the call
func Init() {
	WritePKRU(AllRightsPKRU)
	n := len(g.AllPackages)
	pkgAppearsIn := make(map[int][]c.SandId, n)
	pkgSbProt := make(map[int]map[c.SandId]Prot) // PkgID -> sbID -> mpk prot

	for sbID, sb := range g.Sandboxes {
		for pkgID := range sb.View {
			if pkgID == 0 { // Runtime
				continue
			}
			sbGroup, ok := pkgAppearsIn[pkgID]
			if !ok {
				sbGroup = make([]c.SandId, 0)
			}
			sbProt, ok := pkgSbProt[pkgID]
			if !ok {
				sbProt = make(map[c.SandId]Prot)
				pkgSbProt[pkgID] = sbProt
			}
			pkgAppearsIn[pkgID] = append(sbGroup, sbID)
			view, ok := sb.View[pkgID]
			if !ok {
				panic("Missing view")
			}
			sbProt[sbID] = getMPKProt(view)
		}
	}

	sbKeys := make(map[c.SandId][]int)
	sbProts := make(map[c.SandId][]Prot)
	//TODO remove this loop, it's useless.
	/*for i := range sbKeys {
		sbKeys[i] = make([]int, 0)
		sbProts[i] = make([]Prot, 0)
	}*/

	pkgGroups := make([][]int, 0)
	for len(pkgAppearsIn) > 0 {
		key := len(pkgGroups)
		group := make([]int, 0)
		pkgA_ID, SbGroupA := popMap(pkgAppearsIn)
		for pkgB_ID, SbGroupB := range pkgAppearsIn {
			if testCompatibility(pkgA_ID, pkgB_ID, SbGroupA, SbGroupB, pkgSbProt) {
				group = append(group, pkgB_ID)
			}
		}
		for _, pkgID := range group {
			delete(pkgAppearsIn, pkgID)
		}
		// Add group key to sandbox
		for _, sbID := range SbGroupA {
			prot := pkgSbProt[pkgA_ID][sbID]
			sbKeys[sbID] = append(sbKeys[sbID], key)
			sbProts[sbID] = append(sbProts[sbID], prot)
		}
		pkgGroups = append(pkgGroups, group)
	}

	// We have an allocation for the keys!
	keys := allocateKey(sbKeys, pkgGroups)
	computePKRU(sbKeys, sbProts, keys)

	pkgKeys = make(map[int]Pkey, len(pkgAppearsIn))
	for idx, group := range pkgGroups {
		key := keys[idx]
		for _, pkg := range group {
			pkgKeys[pkg] = key
		}
	}
}

func testCompatibility(aID, bID int, a, b []c.SandId, pkgSbProt map[int]map[c.SandId]Prot) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
		sbID := a[i]
		if pkgSbProt[aID][sbID] != pkgSbProt[bID][sbID] {
			return false
		}
	}
	return true
}

func popMap(m map[int][]c.SandId) (int, []c.SandId) {
	for id, group := range m {
		return id, group
	}
	return -1, nil
}

func getMPKProt(p uint8) Prot {
	if p&c.W_VAL > 0 {
		return ProtRWX
	} else if p&c.R_VAL > 0 {
		return ProtRX
	}
	return ProtX
}
