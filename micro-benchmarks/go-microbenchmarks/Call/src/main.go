package main

import (
	"fmt"
	"gosb"
	"gosb/vtx"
	"gosb/backend"
	"gosb/benchmark"
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
	call := sandbox["main:RWX", ""]() {
		counter++
	}
	for i := 0; i < arg1; i++ {
		if back == backend.VTX_BACKEND {
			vtx.VTXExit()
		}
		start := time.Now()
		cstart := runtime.GetCpuTicks()
		for j := 0; j < arg2; j++ {
			call()
		}
		if back == backend.VTX_BACKEND {
			vtx.VTXExit()
		}
		cend := runtime.GetCpuTicks()
		durations[i] = time.Since(start)
		cycles[i] = cend - cstart
	}

	if counter != uint64(arg1*arg2) {
		panic("urrgh")
	}
	fmt.Println("Call: ", backend.BackendNames[back])
	fmt.Println(benchmark.ComputeMedian(durations, float64(arg2)))
	fmt.Println(benchmark.ComputeMedianCycles(cycles, float64(arg2)))
	if bench {
		gosb.DumpBenchmarks()
	}
}
