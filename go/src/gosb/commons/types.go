package commons

type SandId = string

type SandboxDomain struct {
	Id       SandId
	Func     string
	Sys      SyscallMask
	View     map[string]uint8
	Pkgs     []string
	Pristine bool
}

type Package struct {
	Name    string
	Id      int
	Sects   []Section
	Dynamic []Section
}

type Section struct {
	Addr uint64
	Size uint64
	Prot uint8
}

type SandboxMemory struct {
	Static  *VMAreas
	Config  *SandboxDomain
	View    map[int]uint8
	Entered bool
}

func (p *Package) AddSection(addr, size uint64, prot uint8) {
	update := true
	for _, s := range p.Sects {
		if s.Addr == addr {
			Check(s.Size == size && s.Prot == prot)
			update = false
		}
	}
	if update {
		p.Sects = append(p.Sects, Section{addr, size, prot})
	}
}
