package main

import (
	"fmt"
	"gosb"
	"gosb/backend"
	"gosb/benchmark"
	"runtime"
	"time"
	"unsafe"
)

type _unsafeP = unsafe.Pointer

const (
	_size = 1 * 0x1000
)

var (
	bench     bool            = false
	arg1      int             = 10
	arg2      int             = 10
	back      backend.Backend = backend.VTX_BACKEND
	durations []time.Duration
	cycles []int64
	write     bool = true
	counter   uint64
	transfer func (oldid, newid int, start, size uintptr) = nil
)

func init() {
	back, bench, arg1, arg2 = benchmark.ParseBenchConfig()
	if bench {
		gosb.EnableBenchmarks()
	}
	gosb.Initialize(back)
	switch back {
	case backend.VTX_BACKEND:
		transfer = transferVTX
	case backend.SIM_BACKEND:
		transfer = transferSIM
	case backend.MPK_BACKEND:
		transfer = transferMPK
	}

}

//go:linkname transferSIM gosb/sim.Transfer
func transferSIM(oldid, newid int, start, size uintptr)

//go:linkname transferVTX gosb/vtx.Transfer
func transferVTX(oldid, newid int, start, size uintptr)

//go:linkname transferMPK gosb/mpk.Transfer
func transferMPK(oldid, newid int, start, size uintptr)

func main() {
	durations = make([]time.Duration, arg1)
	cycles = make([]int64, arg1)

	// Sandbox with a dependency on runtime.
	//go:noinline
	x := sandbox["main:R",""]() string {
		return "coucou"
	}
	x()

	array := make([]byte, _size)
	ptr := uintptr(unsafe.Pointer(&array[0]))
	addr, npages, _ := runtime.GosbSpanOf(ptr)
	fmt.Printf("pages: %v %x %v\n", npages, addr, transfer == nil)

	for i := 0; i < arg1; i++ {
		start := time.Now()
		cstart := runtime.GetCpuTicks()
		for j := 0; j < arg2; j++ {
			if j%2 == 0 {
				transfer(-1, 0, addr, _size)
			} else {
				transfer(0, -1, addr, _size)
			}
		}
		cend := runtime.GetCpuTicks()
		durations[i] = time.Since(start)
		cycles[i] = cend - cstart

	}
	fmt.Println("Transfer: ", backend.BackendNames[back])
	fmt.Println(benchmark.ComputeMedian(durations, float64(arg2)))
	fmt.Println(benchmark.ComputeMedianCycles(cycles, float64(arg2)))
	if bench {
		gosb.DumpBenchmarks()
	}
}
