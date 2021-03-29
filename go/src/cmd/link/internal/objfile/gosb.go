package objfile

import (
	gosb "gosb/commons"
	"io/ioutil"
	"strconv"
	"strings"
)

type SBObjEntry struct {
	Func     string
	Id       string
	Mem      string
	Sys      string
	Packages []string
	Extras   []gosb.Entry
	Pristine bool
}

const (
	sandboxheader = "\xeegosandx\n"
	sandboxfooter = "\xefgosandx\n"
)

// Sandboxes we parsed by looking at object files
var (
	Sandboxes      []SBObjEntry
	SBMap          map[string]*SBObjEntry
	SegregatedPkgs map[string]bool
)

func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func registerPackages(pkgs []string) {
	assert(SegregatedPkgs != nil, "Uninitialized SegregatedPkgs!")
	for _, v := range pkgs {
		SegregatedPkgs[v] = true
	}
}

// readSandboxObj parses an object file to gather sandbox information.
// We accumulate this information inside the above global variables.
func readSandboxObj(path string) {
	// Get the entire data.
	data, err := ioutil.ReadFile(path)
	assert(err == nil, "Error reading file")
	file := string(data)
	content := strings.Split(file, sandboxheader)
	// filter only the sandboxes entries
	var sbs []string = nil
	for i, v := range content {
		if i%2 == 1 {
			assert(strings.Contains(v, sandboxfooter), "Malformed sandbox entry: missing footer")
			split := strings.Split(v, sandboxfooter)
			assert(len(split) <= 2, "Malformed sandbox entry: more than two elements in split")
			if len(split[0]) > 0 {
				sbs = append(sbs, split[0])
			}
		}
	}
	if len(sbs) > 0 {
		registerSandboxes(sbs)
	}
}

func registerSandboxes(sbs []string) {
	if SegregatedPkgs == nil {
		SegregatedPkgs = make(map[string]bool)
		SBMap = make(map[string]*SBObjEntry)
	}
	for _, v := range sbs {
		contents := strings.Split(v, "\n")
		assert(len(contents) > 0, "Empty sandbox entry")
		size, err := strconv.Atoi(contents[0])
		assert(err == nil, "error parsing initial size")
		assert(size > 0, "Malformed sandbox entry")
		contents = contents[1:]
		for i := 0; i < size; i++ {
			content := contents
			name, content := content[0], content[1:]
			assert(len(name) > 0, "Empty sandbox name")
			config, content := strings.Split(content[0], ";"), content[1:]
			assert(len(config) == 3, "Malformed configuration")
			nbPkgs, err := strconv.Atoi(content[0])
			assert(err == nil, "Error parsing number of packages")
			content = content[1:]
			// Parse memory view
			extras, pristine, err := gosb.ParseMemoryView(config[1])
			if err != nil {
				panic(err.Error())
			}
			pkgs := make([]string, nbPkgs)
			copy(pkgs, content)
			content = content[nbPkgs:]
			Sandboxes = append(Sandboxes, SBObjEntry{name, config[0], config[1], config[2], pkgs, extras, pristine})
			// Finally add these packages to the ones that need to be bloated
			for _, e := range extras {
				pkgs = append(pkgs, e.Name)
			}
			SBMap[name] = &Sandboxes[len(Sandboxes)-1]
			registerPackages(pkgs)
			contents = content
		}
	}
}
