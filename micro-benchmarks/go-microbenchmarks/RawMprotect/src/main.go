package main

import (
	"fmt"
	"gosb"
	"gosb/backend"
	"gosb/benchmark"
	"gosb/mpk"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const (
	_size = 1 * 0x1000
)

var (
	bench     bool            = false
	arg1      int             = 1000
	arg2      int             = 10000
	back      backend.Backend = backend.VTX_BACKEND
	durations []time.Duration
	cycles    []int64
	write     bool = true
)

func main() {
	durations = make([]time.Duration, arg1)
	cycles = make([]int64, arg1)
	array := make([]byte, _size)
	ptr := uintptr(unsafe.Pointer(&array[0]))
	addr, npages, _ := runtime.GosbSpanOf(ptr)
	fmt.Printf("pages: %v %v\n", npages, addr)
	key1, err1 := mpk.PkeyAlloc()
	key2, err2 := mpk.PkeyAlloc()
	prot := syscall.PROT_READ | syscall.PROT_WRITE
	if err1 != nil || err2 != nil {
		panic(err1)
	}
	for i := 0; i < arg1; i++ {
		cstart := runtime.GetCpuTicks()
		start := time.Now()
		for j := 0; j < arg2; j++ {
			if j%2 == 0 {
				mpk.PkeyMprotect(addr, _size, prot, key1)
			} else {
				mpk.PkeyMprotect(addr, _size, prot, key2)
			}
		}
		durations[i] = time.Since(start)
		cend := runtime.GetCpuTicks()
		cycles[i] = cend - cstart

	}
	fmt.Println("PkeyProt: ", backend.BackendNames[back])
	fmt.Println(benchmark.ComputeMedian(durations, float64(arg2)))
	fmt.Println(benchmark.ComputeMedianCycles(cycles, float64(arg2)))
	if bench {
		gosb.DumpBenchmarks()
	}
}
