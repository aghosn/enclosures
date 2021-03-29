package benchmark

import (
	"gosb/backend"
	c "gosb/commons"
)

func InitBenchWrapper(b *backend.BackendConfig) (*backend.BackendConfig, *Benchmark) {
	bench := &Benchmark{}
	config := &backend.BackendConfig{}
	config.Tpe = b.Tpe
	config.Init = func() {
		bench.BenchStartInit()
		b.Init()
		bench.BenchStopInit()
	}
	config.Prolog = func(id c.SandId) {
		bench.BenchProlog(id)
		b.Prolog(id)
	}
	config.Epilog = func(id c.SandId) {
		b.Epilog(id)
		bench.BenchEpilog(id)
	}
	config.Transfer = func(oldid, newid int, start, size uintptr) {
		bench.BenchEnterTransfer()
		b.Transfer(oldid, newid, start, size)
		//bench.BenchExitTransfer()
	}
	config.Register = func(id int, start, size uintptr) {
		bench.BenchEnterRegister()
		b.Register(id, start, size)
		bench.BenchExitRegister()
	}
	config.Execute = func(id c.SandId) {
		bench.BenchEnterExecute()
		b.Execute(id)
	}
	config.Mstart = func() {
		b.Mstart()
	}
	config.RuntimeGrowth = func(isheap bool, id int, start, size uintptr) {
		bench.growth++
		if b.RuntimeGrowth == nil {
			return
		}
		b.RuntimeGrowth(isheap, id, start, size)
	}
	config.Stats = func() {
		if b.Stats == nil {
			return
		}
		b.Stats()
	}

	return config, bench
}
