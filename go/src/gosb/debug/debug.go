package debug

import (
	"fmt"
	"sync/atomic"
)

// This file implements a very simple debugging library that allows to take small
// time stamps to see where the code goes. Voila voila.

var (
	MRTValues  [1000]uintptr
	MRTIndex   int32 = -1
	MRTValues2 [1000]uintptr
	MRTValues3 [1000]uintptr
	MRTValues4 [1000]uintptr
	MRTIndex2  int
	MRTMarkers [15]int
	MRTUpdates [50]uintptr
	MRTUIdx    int = 0
)

// Reset the debugging tags
//
//go:nosplit
func Reset() {
	MRTValues = [1000]uintptr{}
	MRTIndex = -1
}

//go:nosplit
func TakeValue(a uintptr) int32 {
	idx := atomic.AddInt32(&MRTIndex, 1)
	if int(idx) < len(MRTValues) {
		MRTValues[idx] = a
	}
	return idx
}

//go:nosplit
func TakeValue2(i int, idx int32, a uintptr) {
	if int(idx) < len(MRTValues2) {
		switch i {
		case 0:
			MRTValues2[idx] = a
		case 1:
			MRTValues3[idx] = a
		case 2:
			MRTValues4[idx] = a
		default:
			panic("Out of bounds")
		}
	}
}

//go:nosplit
func DoneAdding(a uintptr) {
	if MRTUIdx < len(MRTUpdates) {
		MRTUpdates[MRTUIdx] = a
		MRTUIdx++
	}
}

//go:nosplit
func TakeInc(a int) {
	if a >= len(MRTMarkers) {
		return
	}
	MRTMarkers[a]++
}

func DumpValues() {
	fmt.Printf("Dumping values: (%v)\n", MRTIndex)
	for i := 0; i < int(MRTIndex); i++ {
		fmt.Printf("%v: %x\n", i, MRTValues[i])
	}
}
