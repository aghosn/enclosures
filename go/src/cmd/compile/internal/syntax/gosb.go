package syntax

import (
	"strconv"
)

var sandboxCounter int = 0

// Identifier for the package
var PkgId int

//TODO(aghosn) see if we want to have two ints instead of a string
func generateSandboxId() *BasicLit {
	b := new(BasicLit)
	b.Kind = StringLit
	b.Value = "\"" + strconv.Itoa(PkgId) + ":" + strconv.Itoa(sandboxCounter) + "\""
	sandboxCounter++
	return b
}

func (p *parser) sandboxConfig() (string, string, string, []Stmt) {
	if trace {
		defer p.trace("sandboxType")()
	}

	pos := p.pos()

	// Parse ["mem", "sys"], generate unique sandbox id
	p.want(_Lbrack)
	memory := p.oliteral()
	p.want(_Comma)
	syscalls := p.oliteral()
	p.want(_Rbrack)

	id := generateSandboxId()
	id.pos = memory.pos

	config := []Expr{id, memory, syscalls}

	//call to preinit, replace with constant from somewhere.
	prolog := sandboxGenerateCall("sandbox_prolog", config)
	prologStmt := new(ExprStmt)
	prologStmt.X = prolog
	prologStmt.pos = pos

	epilog_call := sandboxGenerateCall("sandbox_epilog", config)
	epilogStmt := new(CallStmt)
	epilogStmt.Tok = _Defer
	epilogStmt.Call = epilog_call
	epilogStmt.pos = pos
	return id.Value, memory.Value, syscalls.Value, []Stmt{prologStmt, epilogStmt}
}

func sandboxGenerateCall(name string, args []Expr) *CallExpr {
	pname := new(SBInternal)
	pname.Value = name

	pcall := new(CallExpr)
	pcall.Fun = pname
	pcall.ArgList = args
	pcall.HasDots = false
	return pcall
}
