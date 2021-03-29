package globals

import (
	c "gosb/commons"
	"sort"
)

func AggregatePackages() {
	n := len(AllPackages)
	pkgAppearsIn := make(map[int][]c.SandId, n)
	pkgSbProt := make(map[int]map[c.SandId]uint8)
	for sid, sb := range Sandboxes {
		for pid := range sb.View {
			/*if pid == 0 {
				continue
			}*/

			sbGroup, ok := pkgAppearsIn[pid]
			if !ok {
				sbGroup = make([]c.SandId, 0)
			}
			sbProt, ok := pkgSbProt[pid]
			if !ok {
				sbProt = make(map[c.SandId]uint8)
				pkgSbProt[pid] = sbProt
			}
			pkgAppearsIn[pid] = append(sbGroup, sid)
			view, ok := sb.View[pid]
			if !ok {
				panic("Missing view")
			}
			sbProt[sid] = view
		}
	}

	pkgGroups := make([][]int, 0)
	for len(pkgAppearsIn) > 0 {
		group := make([]int, 0)
		pidA, sbGA := popMap(pkgAppearsIn)
		for pidB, sbGB := range pkgAppearsIn {
			if compatible(pidA, pidB, sbGA, sbGB, pkgSbProt) {
				group = append(group, pidB)
			}
		}
		for _, pid := range group {
			delete(pkgAppearsIn, pid)
		}
		sort.Ints(group)
		pkgGroups = append(pkgGroups, group)
	}
	RtIds = make(map[int]int)
	for _, g := range pkgGroups {
		c.Check(len(g) > 0)
		id := g[0]
		for _, i := range g {
			RtIds[i] = id
		}
	}

	// Compute keys, generate the -1 id
	RtKeys = make(map[int][]int)
	for _, p := range AllPackages {
		id, ok := RtIds[p.Id]
		if !ok {
			id = -1
		}
		list, _ := RtKeys[id]
		RtKeys[id] = append(list, p.Id)
	}
}

func popMap(m map[int][]c.SandId) (int, []c.SandId) {
	for id, group := range m {
		return id, group
	}
	return -1, nil
}

func compatible(pa, pb int, ga, gb []c.SandId, prots map[int]map[c.SandId]uint8) bool {
	if len(ga) != len(gb) {
		return false
	}
	for i := 0; i < len(ga); i++ {
		if ga[i] != gb[i] {
			return false
		}
		sbId := ga[i]
		if prots[pa][sbId] != prots[pb][sbId] {
			return false
		}
	}
	return true
}

// ComputeMemoryView for dynamic sandboxes computes the view based on the available
// information.
func ComputeMemoryView(sbdeps []string, deps map[string][]string, view map[string]uint8) map[string]uint8 {
	result := make(map[string]uint8)
	toadd := make(map[string]bool)
	// Init will all the initial packages, including the views.
	for _, v := range sbdeps {
		c.Check(v != "")
		toadd[v] = true
	}
	for k, _ := range view {
		c.Check(k != "")
		toadd[k] = true
	}

	for {
		if len(toadd) == 0 {
			break
		}
		// we are popping the list.
		current := ""
		for k, _ := range toadd {
			current = k
			delete(toadd, k)
			break
		}
		// We already visited this node and added its dependencies
		if _, ok := result[current]; ok {
			continue
		}
		c.Check(current != "")

		// We did not visit this node and are adding its dependencies.
		cdeps, _ := deps[current]
		for _, v := range cdeps {
			c.Check(v != "")
			toadd[v] = true
		}

		// Check if there is a special restriction on this view.
		prot := c.D_VAL
		if p, ok := view[current]; ok {
			prot = p
		}
		result[current] = prot

	}
	return result
}
