package work

import (
	"bytes"
	"fmt"
	"strings"
)

var (
	allDeps []byte
)

func rejectAction(a *Action) bool {
	if a.Package != nil && a.Package.ImportPath == "" {
		panic("Incomplete name")
	}
	b := a.Package == nil || a.Package.ImportPath == ""
	c := !strings.Contains(a.Mode, "build") && !strings.Contains(a.Mode, "built-in")
	return b || c
}

func registerDependencies(root *Action) {
	var workq []*Action
	var inWorkq = make(map[*Action]int)
	var out bytes.Buffer
	ids := 1
	add := func(a *Action) {
		if _, ok := inWorkq[a]; ok {
			return
		}
		if !rejectAction(a) && a.Package.ImportPath == "runtime" {
			inWorkq[a] = 0
			a.spkgId = 0
		} else {
			inWorkq[a] = ids
			a.spkgId = ids
			ids++
		}
		workq = append(workq, a)
	}
	add(root)

	for i := 0; i < len(workq); i++ {
		for _, dep := range workq[i].Deps {
			add(dep)
		}
	}

	for _, a := range workq {
		id := a.spkgId
		if rejectAction(a) {
			continue
		}
		pname := a.Package.ImportPath
		if pname == "command-line-arguments" {
			pname = "main"
		}
		fmt.Fprintf(&out, "packagedecl %s=%d\n", pname, id)
		dependencies := make([]*Action, 0)
		for _, a1 := range a.Deps {
			if rejectAction(a1) {
				continue
			}
			dependencies = append(dependencies, a1)
		}
		if len(dependencies) == 0 {
			continue
		}
		fmt.Fprintf(&out, "packagedep %d=", id)
		for j, a1 := range dependencies {
			fmt.Fprintf(&out, "%d", inWorkq[a1])
			if j == len(dependencies)-1 {
				fmt.Fprintf(&out, "\n")
			} else {
				fmt.Fprintf(&out, ",")
			}
		}
	}
	allDeps = out.Bytes()
}
