package runtime

const (
	INCR_LVL   = 1
	CALLER_LVL = 2
	SKIPO1_LVL = 3
	MAX_LEVEL  = 4
)

// Exposing runtime locks to the outside.
type GosbMutex struct {
	m mutex
	s sbspinlock
}

//go:nosplit
func (g *GosbMutex) Lock() {
	if isDVTX {
		return
	}
	lock(&g.m)
	//slock(&g.s)
}

//go:nosplit
func (g *GosbMutex) Unlock() {
	if isDVTX {
		return
	}
	unlock(&g.m)
	//sunlock(&g.s)
}

func getpackageid(level int) int {
	if !bloatInitDone {
		return -1
	}
	if level <= 0 || level > MAX_LEVEL {
		throw("Invalid level in getpackageid")
	}
	sp := getcallersp()
	pc := getcallerpc()
	gp := getg()
	var n int
	var pcbuf [MAX_LEVEL]uintptr
	systemstack(func() {
		n = gentraceback(pc, sp, 0, gp, 0, &pcbuf[0], level, nil, nil, 0)
	})

	if n != level {
		panic("Unable to unwind the stack")
	}
	id := pcToPkg(pcbuf[n-1])
	if id == -1 && gp.sbid != _OUT_MODE {
		throw("What the fuck")
	}
	return id
}

func filterPkgId(id int) int {
	if !bloatInitDone {
		if !mainInitDone || id == 0 {
			return 0
		}
		return -1
	}
	if _, ok := idToPkg[id]; ok {
		return id
	}
	return -1
}

func gosbInterpose(lvl int) int {
	if !bloatInitDone {
		return 0
	}
	id := -1
	mp := acquirem()
	if mp.tracingAlloc == 1 {
		throw("Oups")
		goto cleanup
	}
	mp.tracingAlloc = 1
	id = filterPkgId(getpackageid(lvl + INCR_LVL))
cleanup:
	mp.tracingAlloc = 0
	releasem(mp)
	return id
}
