package commons

import (
	"fmt"
	"log"
	"sort"
)

// VMAreas represents an address space, i.e., a list of VMArea.
type VMAreas struct {
	List
}

const (
	PageSize = 0x1000
)

func Convert(acc []*VMArea) *VMAreas {
	// Sort and coalesce
	sort.Slice(acc, func(i, j int) bool {
		return acc[i].Addr <= acc[j].Addr
	})
	space := &VMAreas{}
	space.List.Init()
	for _, s := range acc {
		space.List.AddBack(s.ToElem())
	}
	space.Coalesce()
	return space
}

func PackageToVMAs(p *Package) *VMAreas {
	vmareas := PackageToVMAreas(p, D_VAL)
	return Convert(vmareas)
}

// PackageToVMAreas translates a package into a slice of vmareas,
// applying the replacement view mask to the protection.
func PackageToVMAreas(p *Package, replace uint8) []*VMArea {
	acc := make([]*VMArea, 0)
	for i, s := range p.Sects {
		if s.Addr%PageSize != 0 {
			log.Printf("Section number %d\n", i)
			panic("Section address not aligned ")
		}
		area := SectVMA(&s)
		// @warning IMPORTANT Skip the empty sections (otherwise crashes)
		if area == nil {
			continue
		}
		area.Prot &= replace
		area.Prot |= USER_VAL
		acc = append(acc, area)
	}

	// map the dynamic sections
	for _, d := range p.Dynamic {
		area := SectVMA(&d)
		if area == nil {
			log.Fatalf("error, dynamic section should no be empty")
		}
		area.Prot &= replace
		area.Prot |= USER_VAL
		acc = append(acc, area)
	}
	return acc
}

// coalesce is called to merge vmareas
func (s *VMAreas) Coalesce() {
	for curr := s.First; curr != nil; curr = curr.Next {
		next := curr.Next
		if next == nil {
			return
		}
		currVma := ToVMA(curr)
		nextVma := ToVMA(next)
		for v, merged := currVma.merge(nextVma); merged && nextVma != nil; {
			s.Remove(next)
			if currVma != v {
				log.Fatalf("These should be equal %v %v\n", currVma, v)
			}
			next = curr.Next
			nextVma = ToVMA(curr.Next)
			v, merged = currVma.merge(nextVma)
		}
	}
}

// Map maps a VMAreas to the address space.
// So far the implementation is stupid and inefficient.
func (s *VMAreas) Map(vma *VMArea) {
	if s.IsEmpty() {
		s.AddBack(vma.ToElem())
		return
	}

	for v := ToVMA(s.First); v != nil; v = ToVMA(v.Next) {
		next := ToVMA(v.Next)
		if vma.Addr < v.Addr {
			s.InsertBefore(vma.ToElem(), v.ToElem())
			break
		}
		if vma.Addr >= v.Addr && (next == nil || vma.Addr <= next.Addr) {
			s.InsertAfter(vma.ToElem(), v.ToElem())
			break
		}
	}
	if vma.List == nil {
		// Probably already mapped.
		log.Printf("vma: %v\n", vma)
		panic("Failed to insert the vma.")
	}
	s.Coalesce()
}

// MapArea maps a vmarea inside another vmarea
func (s *VMAreas) MapArea(vm *VMAreas) {
	for v := ToVMA(vm.First); v != nil; {
		next := ToVMA(v.Next)
		vm.Remove(v.ToElem())
		s.Map(v)
		v = next
	}
}

func (s *VMAreas) MapAreaCopy(vm *VMAreas) {
	doppler := vm.Copy()
	s.MapArea(doppler)
}

func (s *VMAreas) UnmapArea(vm *VMAreas) {
	if vm == nil {
		return
	}
	for v := ToVMA(vm.First); v != nil; v = ToVMA(v.Next) {
		s.Unmap(v)
	}
}

// Unmap removes a VMArea from the address space.
//
//go:nosplit
func (s *VMAreas) Unmap(vma *VMArea) {
	for v := ToVMA(s.First); v != nil; v = ToVMA(v.Next) {
	begin:
		// Full overlap [xxx[vxvxvxvxvx]xxx]
		if v.intersect(vma) && v.Addr >= vma.Addr && v.Addr+v.Size <= vma.Addr+vma.Size {
			next := ToVMA(v.Next)
			s.Remove(v.ToElem())
			v = next
			if v == nil {
				break
			}
			goto begin
		}
		// Left case, reduces v : [vvvv[vxvxvxvx]xxx]
		if v.intersect(vma) && v.Addr < vma.Addr && vma.Addr+vma.Size >= v.Addr+v.Size {
			v.Size = vma.Addr - v.Addr
			continue
		}
		// Fully contained [vvvv[vxvxvx]vvvv], requires a split
		if v.intersect(vma) && v.Addr < vma.Addr && v.Addr+v.Size > vma.Addr+vma.Size {
			nstart := vma.Addr + vma.Size
			nsize := v.Addr + v.Size - nstart
			v.Size = vma.Addr - v.Addr
			s.Map(&VMArea{
				ListElem{},
				Section{nstart, nsize, v.Prot},
			})
			break
		}
		// Right case, contained: [[xvxv]vvvvvv] or [xxxx[xvxvxvxvx]vvvv]
		if v.intersect(vma) && v.Addr >= vma.Addr && v.Addr+v.Size > vma.Addr+vma.Size {
			nstart := vma.Addr + vma.Size
			nsize := v.Addr + v.Size - nstart
			v.Addr = nstart
			v.Size = nsize
			break
		}
	}
}

func (s *VMAreas) Mirror() *VMAreas {
	mirror := &VMAreas{}
	a := &VMArea{
		ListElem{},
		Section{
			Addr: 0x0,
			Size: uint64(Limit39bits),
		},
	}
	mirror.AddBack(a.ToElem())
	for v := ToVMA(s.First); v != nil && uintptr(v.Addr) < Limit39bits; v = ToVMA(v.Next) {
		mirror.Unmap(v)
	}
	return mirror
}

func (vs *VMAreas) Copy() *VMAreas {
	if vs == nil {
		return nil
	}
	doppler := &VMAreas{}
	for v := ToVMA(vs.First); v != nil; v = ToVMA(v.Next) {
		cpy := v.Copy()
		doppler.AddBack(cpy.ToElem())
	}
	return doppler
}

func (vs *VMAreas) Print() {
	for v := ToVMA(vs.First); v != nil; v = ToVMA(v.Next) {
		fmt.Printf("%x -+- %x (%x)\n", v.Addr, v.Addr+v.Size, v.Prot)
	}
}
