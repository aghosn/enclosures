package gc

import (
	"cmd/compile/internal/ssa"
	"cmd/compile/internal/syntax"
	"cmd/compile/internal/types"
	"cmd/internal/bio"
	"fmt"
)

type Pkg = types.Pkg

type PkgSet = map[*Pkg]bool

var sandboxes []*Node

var sandboxToPkgs map[*Node][]*Pkg

const (
	SandboxHeader = "\xeegosandx"
	SandboxFooter = "\xefgosandx"
)

func (n *Node) SandboxName() string {
	if !n.IsSandbox || n.Op != ODCLFUNC || n.Func == nil || n.Func.Nname == nil {
		panic("Unable to get sandbox name")
	}
	return fmt.Sprintf("%v", n.Func.Nname)
}

func registerSandboxes(top []*Node) {
	if sandboxToPkgs == nil {
		sandboxToPkgs = make(map[*Node][]*Pkg)
	}
	for _, v := range top {
		if v.Op == ODCLFUNC && v.IsSandbox {
			sandboxes = append(sandboxes, v)
			pkgs := gatherPackages(v)
			if _, ok := sandboxToPkgs[v]; ok {
				panic("Sandbox appears twice as a DCLFUNC")
			}
			sandboxToPkgs[v] = pkgs
		}
	}
}

func gatherPackages(n *Node) []*Pkg {
	var keys []*Pkg
	uniq := make(PkgSet)
	aggreg := make(map[*Node]*Pkg)
	gatherPackages1(n, aggreg)
	for _, p := range aggreg {
		if _, ok := uniq[p]; !ok && p != nil {
			uniq[p] = true
			keys = append(keys, p)
		}
	}
	return keys
}

// gatherPackages1 the aggreg is there to make sure we don't visit the same node
// too many times.
// This is a recursive function, that's why I think that we need the aggreg to
// avoid infinite loops. Let's see if that is the case. If not, I can simplify
// the function.
func gatherPackages1(n *Node, aggreg map[*Node]*Pkg) {
	if n == nil {
		return
	}
	if _, ok := aggreg[n]; ok {
		return
	}
	if p := getPackage(n); p != nil {
		// @aghosn we ignore empty path which just means local package.
		if p.Path != "" && p.Path != "go.itab" && p.Path != "go.runtime" {
			aggreg[n] = p
		}

	}
	gatherPackages1(n.Left, aggreg)
	gatherPackages1(n.Right, aggreg)
	gatherPackagesSlice(n.Ninit.Slice(), aggreg)
	gatherPackagesSlice(n.Nbody.Slice(), aggreg)
	gatherPackagesSlice(n.List.Slice(), aggreg)
	gatherPackagesSlice(n.Rlist.Slice(), aggreg)
}

func gatherPackagesSlice(nodes []*Node, aggreg map[*Node]*Pkg) {
	for _, v := range nodes {
		gatherPackages1(v, aggreg)
	}
}

// getPackage does its best to find a package for a given node.
func getPackage(n *Node) *Pkg {
	if n == nil {
		return nil
	}
	// We get the package directly from the symbol.
	if n.Sym != nil && n.Sym.Pkg != nil {
		return n.Sym.Pkg
	}
	// Special cases.
	switch n.Op {
	case ODCLFUNC:
		if n.Func == nil {
			panic("DCLFUNC has nil Func attribute.")
		}
		fname := n.Func.Nname
		if fname != nil && fname.Sym != nil && fname.Sym.Pkg != nil {
			return fname.Sym.Pkg
		}
		panic("DCLFUNC symbol without package.")
	case ONAME:
		name := n.Name
		if name == nil {
			panic("NAME node has nil name.")
		}
		if name.Pkg != nil {
			return name.Pkg
		}
		if p := getPackage(name.Pack); p != nil {
			return p
		}
		panic("NAME symbol without package.")
	}
	//TODO(aghosn) if we reach here, it means that we either failed,
	//or do not need to handle the node.
	return nil
}

// dumpSandObj dumps sandbox information inside the archive.
func dumpSandObj(bout *bio.Writer) {
	printSandObjHeader(bout)
	dumpSandboxes(bout)
	printSandObjFooter(bout)
}

// printSandObjHeader writes the go sandboxes header, that for the moment
// contains only the number of entries.
func printSandObjHeader(bout *bio.Writer) {
	fmt.Fprintf(bout, "%v\n", SandboxHeader)
	fmt.Fprintf(bout, "%v\n", len(sandboxes))
	bout.Flush()
}

// printSandObjFooter footer for a sandbox object entry
func printSandObjFooter(bout *bio.Writer) {
	fmt.Fprintf(bout, "%v\n", SandboxFooter)
	bout.Flush()
}

// dumpSandboxes dumps all the sandboxes.
func dumpSandboxes(bout *bio.Writer) {
	for _, s := range sandboxes {
		dumpSandbox(s, bout)
	}
}

// dumpSandbox writes a sandbox information to the object file.
func dumpSandbox(s *Node, bout *bio.Writer) {
	// Sanity checks
	if s == nil || !s.IsSandbox || len(s.Id) == 0 || len(s.Mem) == 0 || len(s.Sys) == 0 {
		panic("Malformed sandbox")
	}
	if _, ok := sandboxToPkgs[s]; !ok {
		panic("Missing package information for sandbox")
	}
	// dump the sandbox symbol.
	fmt.Fprintf(bout, "%v\n", myimportpath+"."+s.SandboxName())
	// dump the sandbox configuration
	fmt.Fprintf(bout, "%v;%v;%v\n", s.Id, s.Mem, s.Sys)
	// dump package dependencies
	unfilteredpkgs, _ := sandboxToPkgs[s]

	// filter unwanted packages
	pkgs := make([]string, 0)
	for _, p := range unfilteredpkgs {
		if p.Path == "go.builtin" {
			continue
		}
		pkgs = append(pkgs, p.Path)
	}

	fmt.Fprintf(bout, "%v\n", len(pkgs))
	for _, p := range pkgs {
		fmt.Fprintf(bout, "%v\n", p)
	}
}

// ssa external function

func newobjectPkgArg(s *state) *ssa.Value {
	itpe := types.Types[types.TINT]
	return s.constInt64(itpe, int64(syntax.PkgId))
}
