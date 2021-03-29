package runtime

import (
	"runtime/internal/atomic"
	"unsafe"
)

var (
	MRTRuntimeVals [1000]uintptr
	MRTRuntimeIdx  uint32 = 0
	MRTId          int64  = -1
	MRTBaddy       int    = 0
	Lock           GosbMutex

	// The value
	SchedLock uintptr = uintptr(unsafe.Pointer(&sched.lock))
)

var (
	gcMarkAddr     uintptr = 0
	timerProcAddr  uintptr = 0
	bgsweepAddr    uintptr = 0
	bgscavengeAddr uintptr = 0
)

//go:nosplit
func TakeValue(a uintptr) {
	idx := atomic.Xadd(&MRTRuntimeIdx, 1)
	if int(idx) < len(MRTRuntimeVals) {
		MRTRuntimeVals[idx] = a
	}
}

//go:nosplit
func Reset() {
	MRTBaddy = 0
}

//go:nosplit
func StartCapture() {
	_g_ := getg()
	MRTId = _g_.goid
	Reset()
}

//go:nosplit
func TakeValueTrace(a uintptr) {
	_g_ := getg()
	if _g_ == nil {
		return
	}
	if _g_.goid == MRTId {
		TakeValue(a)
	}
}

func GiveGoid() int64 {
	_g_ := getg()
	return _g_.goid
}

//go:nosplit
func FindAllSpans(add uintptr) {
	var count = 0
	for i := 0; i < len(mheap_.allspans); i++ {
		if mheap_.allspans[i].base() == add {
			count++
		}
	}
	if count != 1 {
		throw("Many of them")
	}
}
