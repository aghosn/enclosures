package memview

import (
	c "gosb/commons"
	pg "gosb/vtx/platform/ring0/pagetables"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Globals that are shared by everyone.
// These include:
// (1) A synchronized freespace allocator, used to map portions of the address
//	space that are above the 40bits limit. It is shared as VMs might have to
//	update each other.
// (2) God address space. This is a representation of the current program as
//	an address space that runtime routines can switch to without leaving the VM.
var (
	FreeSpace  *FreeSpaceAllocator = nil
	GodAS      *AddressSpace       = nil
	GodMu      runtime.GosbMutex
	CHeap      uint64 = 0
	CheapStart uint64 = 0
	CheapSize  uint64 = 0
)

// Due to concurrency issue, we might have delayed updates between
// initialization of the full memory view, and setting up the hooks
// in the runtime.
var (
	EUpdates [50]*c.VMArea
	CurrE    int = 0
	Updates  c.VMAreas
)

const (
	EXTRA_CHEAP_SIZE = uint64(0x10000000)
	CHEAP_INCR_FLAG  = "INC_HEAP"
)

// Initialize creates a view of the entire address space and the GodAS.
// (1) parse the entire address space from self proc.
// (2) create the corresponding vmas.
// (3) mirror the full address space.
// (4) create a corresponding address space with the associated page tables.
func InitializeGod() {
	// Register the hook with the runtime.
	for i := range EUpdates {
		EUpdates[i] = &c.VMArea{}
	}
	runtime.RegisterEmergencyGrowth(EmergencyGrowth)

	// Start parsing the address space.
	fvmas := ParseProcessAddressSpace(c.USER_VAL)
	full := c.Convert(fvmas)
	GodAS = &AddressSpace{}
	FreeSpace = &FreeSpaceAllocator{}

	// Create the free space allocator.
	free := full.Mirror()
	FreeSpace.Initialize(free, false)
	GodAS.FreeAllocator = FreeSpace

	// Create the page tables
	GodAS.PTEAllocator = &PageTableAllocator{}
	GodAS.PTEAllocator.Initialize(GodAS.FreeAllocator)
	GodAS.Tables = pg.New(GodAS.PTEAllocator)

	// Create the memory regions for GodAS
	for v := c.ToVMA(full.First); v != nil; {
		next := c.ToVMA(v.Next)
		full.Remove(v.ToElem())
		region := GodAS.VMAToMemoryRegion(v)
		GodAS.Regions.AddBack(region.ToElem())
		// update the loop
		v = next
	}
	GodAS.Replenish()
}

//go:nosplit
func EmergencyGrowth(isheap bool, id int, start, size uintptr) {
	v := acquireUpdate()
	c.Check(v != nil)
	v.Addr, v.Size, v.Prot = uint64(start), uint64(size), c.HEAP_VAL
	Updates.AddBack(v.ToElem())
}

//go:nosplit
func acquireUpdate() *c.VMArea {
	if CurrE < len(EUpdates) {
		i := CurrE
		CurrE++
		return EUpdates[i]
	}
	return nil
}

// ParseProcessAddressSpace parses the self proc map to get the entire address space.
// defProt is the common set of flags we want for this.
func ParseProcessAddressSpace(defProt uint8) []*c.VMArea {
	dat, err := ioutil.ReadFile("/proc/self/maps")
	if err != nil {
		log.Fatalf(err.Error())
	}
	tvmas := strings.Split(string(dat), "\n")
	vmareas := make([]*c.VMArea, 0)
	for _, v := range tvmas {
		if len(v) == 0 || strings.Contains(v, "vsyscall") || strings.Contains(v, "anon_inode:kvm-vcpu") {
			continue
		}
		fields := strings.Fields(v)
		if len(fields) < 5 {
			log.Fatalf("error incomplete entry in /proc/self/maps: %v\n", fields)
		}
		// Parsing addresses.
		bounds := strings.Split(fields[0], "-")
		if len(bounds) != 2 {
			log.Fatalf("error founding bounds of area: %v\n", bounds)
		}
		start, err := strconv.ParseUint(bounds[0], 16, 64)
		end, err1 := strconv.ParseUint(bounds[1], 16, 64)
		if err != nil || err != nil {
			log.Fatalf("error parsing bounds of area: %v %v\n", err, err1)
		}
		// Parsing access rights.
		rstr := fields[1]
		rights := uint8(0)
		if strings.Contains(rstr, "r") {
			rights |= c.R_VAL
		}
		// This doesn't work for some C dependencies that have ---p
		/*rights := uint8(c.R_VAL)
		if !strings.Contains(rstr, "r") {
			log.Fatalf("missing read rights parsed from self proc: %v\n", rstr)
		}*/
		if strings.Contains(rstr, "w") {
			rights |= c.W_VAL
		}
		if strings.Contains(rstr, "x") {
			rights |= c.X_VAL
		}

		vm := &c.VMArea{
			c.ListElem{},
			c.Section{
				Addr: uint64(start),
				Size: uint64(end - start),
				Prot: uint8(rights | defProt),
			},
		}
		vmareas = append(vmareas, vm)
		if rights == c.W_VAL|c.R_VAL && strings.Contains(v, "[heap]") {
			CHeap = uint64(start)
			vm.Prot |= uint8(c.S_VAL)
			if os.Getenv(CHEAP_INCR_FLAG) != "" {
				vm.Size += EXTRA_CHEAP_SIZE
			}
			CheapStart = uint64(start)
			CheapSize = uint64(vm.Size)
		}
	}
	return vmareas
}

func UpdateHeap() *c.VMArea {
	dat, err := ioutil.ReadFile("/proc/self/maps")
	if err != nil {
		log.Fatalf(err.Error())
	}
	tvmas := strings.Split(string(dat), "\n")
	for _, l := range tvmas {
		if !strings.Contains(l, "[heap]") {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) < 5 {
			log.Fatalf("error incomplete entry in /proc/self/maps: %v\n", fields)
		}
		// Parsing addresses.
		bounds := strings.Split(fields[0], "-")
		if len(bounds) != 2 {
			log.Fatalf("error founding bounds of area: %v\n", bounds)
		}
		start, err := strconv.ParseUint(bounds[0], 16, 64)
		end, err1 := strconv.ParseUint(bounds[1], 16, 64)
		c.Check(err == nil)
		c.Check(err1 == nil)
		c.Check(CHeap != 0)
		c.Check(start == CHeap)
		vma := &c.VMArea{
			c.ListElem{},
			c.Section{
				Addr: uint64(start),
				Size: uint64(end - start),
				Prot: 0,
			},
		}
		return vma
	}
	panic("Could not find heap")
	return nil
}
