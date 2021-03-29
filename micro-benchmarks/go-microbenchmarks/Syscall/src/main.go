package main

import (
	"fmt"
	"gosb"
	"gosb/backend"
	"gosb/vtx"
	"gosb/benchmark"
	"syscall"
	"runtime"
	"time"
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
)

func init() {
	back, bench, arg1, arg2 = benchmark.ParseBenchConfig()
	if bench {
		gosb.EnableBenchmarks()
	}
	gosb.Initialize(back)
}

func main() {
	durations = make([]time.Duration, arg1)
	cycles = make([]int64, arg1)
	//go:noinline
	call := sandbox["main:RWX", ""](round int) {
		for i := 0; i < arg2; i++ {
			_, _, _ = syscall.Syscall(syscall.SYS_GETUID, 0, 0, 0)
		}
	}
	for i := 0; i < arg1; i++ {
		if back == backend.VTX_BACKEND {
			vtx.VTXExit()
		}
		start := time.Now()
		cstart := runtime.GetCpuTicks()
		call(i)
		if back == backend.VTX_BACKEND {
			vtx.VTXExit()
		}
		cend := runtime.GetCpuTicks()
		durations[i] = time.Since(start)
		cycles[i] = cend - cstart
	}

	fmt.Println("Syscall: ", backend.BackendNames[back])
	fmt.Println(benchmark.ComputeMedian(durations, float64(arg2)))
	fmt.Println(benchmark.ComputeMedianCycles(cycles, float64(arg2)))
	if bench {
		gosb.DumpBenchmarks()
	}
}
