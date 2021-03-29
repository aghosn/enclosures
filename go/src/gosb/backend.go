package gosb

import (
	be "gosb/backend"
	"gosb/benchmark"
	"gosb/mpk"
	"gosb/sim"
	"gosb/vtx"
)

// Configurations
var (
	configBackends = [be.BACKEND_SIZE]be.BackendConfig{
		be.BackendConfig{be.SIM_BACKEND, sim.Init, sim.Prolog, sim.Epilog, sim.Transfer, sim.Register, sim.Execute, nil, sim.RuntimeGrowth, sim.Stats},
		be.BackendConfig{be.VTX_BACKEND, vtx.Init, vtx.Prolog, vtx.Epilog, vtx.Transfer, vtx.Register, vtx.Execute, nil, vtx.RuntimeGrowth, vtx.Stats},
		be.BackendConfig{be.MPK_BACKEND, mpk.Init, mpk.Prolog, mpk.Epilog, mpk.Transfer, mpk.Register, mpk.Execute, mpk.MStart, nil, nil},
		be.BackendConfig{be.DVTX_BACKEND, vtx.DInit, vtx.DProlog, vtx.DEpilog, vtx.DynTransfer, nil, nil, nil, vtx.DRuntimeGrowth, nil},
		be.BackendConfig{be.DMPK_BACKEND, mpk.DInit, mpk.DProlog, mpk.DEpilog, nil, nil, nil, nil, mpk.DRuntimeGrowth, nil},
	}
)

// The actual backend that we use in this session
var (
	currBackend  *be.BackendConfig
	benchmarking bool = false
)

func EnableBenchmarks() {
	benchmarking = true
}

func initBackend(b be.Backend) {
	currBackend = &configBackends[b]
	if benchmarking {
		currBackend, benchmark.Bench = benchmark.InitBenchWrapper(currBackend)
	}
	if currBackend.Init != nil {
		currBackend.Init()
	}
}

func DumpBenchmarks() {
	if benchmark.Bench == nil {
		return
	}
	benchmark.Bench.Dump()
	if currBackend.Stats != nil {
		currBackend.Stats()
	}
}

func ResetBenchmarks() {
	if benchmark.Bench == nil {
		return
	}
	benchmark.Bench.Reset()
}
