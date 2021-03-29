package commons

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
)

// This file defines the format for sandbox configuration.
// These are the two strings passed to the sandbox, i.e., sandbox["main:R", "syscall"].\
//
// The first one represents the memory view, i.e., a refinement of the memory access rights
// over the default ones of this sandbox. By default, the sandbox get the original access rights to its
// code and data dependencies. This argument allows to further reduce these, or increase rights on packages
// that are not part of the sandbox dependencies (e.g., explicitely allow access to a pointer generated in
// another package).
// The grammar is:
// perm := [R]?[W]?[X]? || P
// entry := name:rights
// config := entry1,entry2,... // separated by commas
//
// The second argument represent syscall classes that are whitelisted for this sandbox.

type Entry struct {
	Name string
	Perm uint8
}

const (
	DELIMITER_PKGS  = ","
	DELIMITER_ENTRY = ":"
	SELF_IDENTIFIER = "self"

	// Permissions
	UNMAP    = "U"
	PRISTINE = "P"
	READ     = "R"
	WRITE    = "W"
	EXECUTE  = "X"

	_PageSize = uint64(0x1000)
)

const (
	// These constants are made to match the ones in cmd/link/internal/ld/elf.go
	// This is an unmap flag
	U_VAL = uint8(0)
	X_VAL = uint8(1)
	W_VAL = uint8(1 << 1)
	R_VAL = uint8(1 << 2)
	S_VAL = uint8(1 << 3) // allows to separate a vma
	// Extra definitions that we require for seggregating pages.
	USER_VAL  = uint8(3 << 4)
	SUPER_VAL = uint8(1 << 4)
	P_VAL     = uint8(1 << 6)
	UNMAP_VAL = uint8(1 << 7)
	D_VAL     = R_VAL | W_VAL | X_VAL | USER_VAL // default set
	DEF_VAL   = R_VAL | W_VAL | X_VAL
	HEAP_VAL  = R_VAL | W_VAL | USER_VAL
)

func ParseMemoryView(memc string) ([]Entry, bool, error) {
	pristine := false
	mem, err := strconv.Unquote(memc)
	if err != nil {
		mem = memc
	}
	if len(mem) == 0 {
		return []Entry{}, false, nil
	}
	entries := strings.Split(mem, DELIMITER_PKGS)
	res := make([]Entry, 0)
	uniq := make(map[string]bool)
	for _, v := range entries {
		e, err := parseEntry(v)
		if err != nil {
			return res, false, err
		}
		if _, ok := uniq[e.Name]; ok {
			return nil, false, fmt.Errorf("Duplicated entry for %v\n", e.Name)
		}
		if e.Name == SELF_IDENTIFIER && e.Perm != P_VAL {
			return nil, false, fmt.Errorf("self can only be pristine %v\n", e.Perm)
		}

		if e.Name == SELF_IDENTIFIER {
			pristine = true
			continue
		}

		uniq[e.Name] = true
		res = append(res, e)
	}
	return res, pristine, nil
}

func parseEntry(entry string) (Entry, error) {
	split := strings.Split(entry, DELIMITER_ENTRY)
	if len(split) != 2 {
		return Entry{}, fmt.Errorf("Parsing error: expected 2 values, got %v: [%v]\n", len(split), entry)
	}
	name := strings.TrimSpace(split[0])
	if len(name) == 0 {
		return Entry{}, fmt.Errorf("Invalid package name of length 0\n")
	}
	perm, err := parsePerm(strings.TrimSpace(split[1]))
	if err != nil {
		return Entry{}, err
	}
	if perm == P_VAL && name != SELF_IDENTIFIER {
		return Entry{}, fmt.Errorf("Pristine applied to non self package")
	}
	return Entry{name, perm}, nil
}

func parsePerm(entry string) (uint8, error) {
	if len(entry) == 0 {
		return 0, fmt.Errorf("Unspecified permissions\n")
	}
	if len(entry) > 3 {
		return 0, fmt.Errorf("Invalid permission length %v\n", len(entry))
	}
	if entry == UNMAP {
		return U_VAL, nil
	}
	if entry == PRISTINE {
		return P_VAL, nil
	}
	perm := uint8(0)
	for i := 0; i < len(entry); i++ {
		char := string(entry[i])
		bit := uint8(0)
		switch char {
		case READ:
			bit = R_VAL
		case WRITE:
			bit = W_VAL
		case EXECUTE:
			bit = X_VAL
		default:
			return 0, fmt.Errorf("Invalid permission marker %v\n", char)
		}
		if (bit & perm) != 0 {
			return 0, fmt.Errorf("redundant permission marker %v in %v\n", char, entry)
		}
		perm |= bit
	}
	if (perm & R_VAL) == 0 {
		return 0, fmt.Errorf("Reading access right must be specified explicitly.\n")
	}
	return perm, nil
}

// Converts from mmap access rights to our own.
func ConvertRights(rights uintptr) uint8 {
	prots := USER_VAL
	if rights&syscall.PROT_READ != 0 {
		prots |= R_VAL
	}
	if rights&syscall.PROT_WRITE != 0 {
		prots |= W_VAL
	}
	if rights&syscall.PROT_EXEC != 0 {
		prots |= X_VAL
	}
	return prots
}

//go:nosplit
func Round(addr uint64, up bool) uint64 {
	res := addr - (addr % _PageSize)
	if up && (addr%_PageSize != 0) {
		res += _PageSize
	}
	return res
}
