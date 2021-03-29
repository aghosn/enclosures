// Code generated by mkbuiltin.go. DO NOT EDIT.

package gc

import "cmd/compile/internal/types"

var runtimeDecls = [...]struct {
	name string
	tag  int
	typ  int
}{
	{"sandbox_prolog", funcTag, 1},
	{"sandbox_epilog", funcTag, 1},
	{"newobject", funcTag, 7},
	{"panicdivide", funcTag, 8},
	{"panicshift", funcTag, 8},
	{"panicmakeslicelen", funcTag, 8},
	{"throwinit", funcTag, 8},
	{"panicwrap", funcTag, 8},
	{"gopanic", funcTag, 10},
	{"gorecover", funcTag, 13},
	{"goschedguarded", funcTag, 8},
	{"goPanicIndex", funcTag, 14},
	{"goPanicIndexU", funcTag, 16},
	{"goPanicSliceAlen", funcTag, 14},
	{"goPanicSliceAlenU", funcTag, 16},
	{"goPanicSliceAcap", funcTag, 14},
	{"goPanicSliceAcapU", funcTag, 16},
	{"goPanicSliceB", funcTag, 14},
	{"goPanicSliceBU", funcTag, 16},
	{"goPanicSlice3Alen", funcTag, 14},
	{"goPanicSlice3AlenU", funcTag, 16},
	{"goPanicSlice3Acap", funcTag, 14},
	{"goPanicSlice3AcapU", funcTag, 16},
	{"goPanicSlice3B", funcTag, 14},
	{"goPanicSlice3BU", funcTag, 16},
	{"goPanicSlice3C", funcTag, 14},
	{"goPanicSlice3CU", funcTag, 16},
	{"printbool", funcTag, 18},
	{"printfloat", funcTag, 20},
	{"printint", funcTag, 22},
	{"printhex", funcTag, 24},
	{"printuint", funcTag, 24},
	{"printcomplex", funcTag, 26},
	{"printstring", funcTag, 27},
	{"printpointer", funcTag, 28},
	{"printiface", funcTag, 28},
	{"printeface", funcTag, 28},
	{"printslice", funcTag, 28},
	{"printnl", funcTag, 8},
	{"printsp", funcTag, 8},
	{"printlock", funcTag, 8},
	{"printunlock", funcTag, 8},
	{"concatstring2", funcTag, 31},
	{"concatstring3", funcTag, 32},
	{"concatstring4", funcTag, 33},
	{"concatstring5", funcTag, 34},
	{"concatstrings", funcTag, 36},
	{"cmpstring", funcTag, 37},
	{"intstring", funcTag, 40},
	{"slicebytetostring", funcTag, 42},
	{"slicebytetostringtmp", funcTag, 43},
	{"slicerunetostring", funcTag, 46},
	{"stringtoslicebyte", funcTag, 47},
	{"stringtoslicerune", funcTag, 50},
	{"slicecopy", funcTag, 52},
	{"slicestringcopy", funcTag, 53},
	{"decoderune", funcTag, 54},
	{"countrunes", funcTag, 55},
	{"convI2I", funcTag, 56},
	{"convT16", funcTag, 58},
	{"convT32", funcTag, 58},
	{"convT64", funcTag, 58},
	{"convTstring", funcTag, 58},
	{"convTslice", funcTag, 58},
	{"convT2E", funcTag, 59},
	{"convT2Enoptr", funcTag, 59},
	{"convT2I", funcTag, 59},
	{"convT2Inoptr", funcTag, 59},
	{"assertE2I", funcTag, 56},
	{"assertE2I2", funcTag, 60},
	{"assertI2I", funcTag, 56},
	{"assertI2I2", funcTag, 60},
	{"panicdottypeE", funcTag, 61},
	{"panicdottypeI", funcTag, 61},
	{"panicnildottype", funcTag, 62},
	{"ifaceeq", funcTag, 64},
	{"efaceeq", funcTag, 64},
	{"fastrand", funcTag, 66},
	{"makemap64", funcTag, 68},
	{"makemap", funcTag, 69},
	{"makemap_small", funcTag, 70},
	{"mapaccess1", funcTag, 71},
	{"mapaccess1_fast32", funcTag, 72},
	{"mapaccess1_fast64", funcTag, 72},
	{"mapaccess1_faststr", funcTag, 72},
	{"mapaccess1_fat", funcTag, 73},
	{"mapaccess2", funcTag, 74},
	{"mapaccess2_fast32", funcTag, 75},
	{"mapaccess2_fast64", funcTag, 75},
	{"mapaccess2_faststr", funcTag, 75},
	{"mapaccess2_fat", funcTag, 76},
	{"mapassign", funcTag, 71},
	{"mapassign_fast32", funcTag, 72},
	{"mapassign_fast32ptr", funcTag, 72},
	{"mapassign_fast64", funcTag, 72},
	{"mapassign_fast64ptr", funcTag, 72},
	{"mapassign_faststr", funcTag, 72},
	{"mapiterinit", funcTag, 77},
	{"mapdelete", funcTag, 77},
	{"mapdelete_fast32", funcTag, 78},
	{"mapdelete_fast64", funcTag, 78},
	{"mapdelete_faststr", funcTag, 78},
	{"mapiternext", funcTag, 79},
	{"mapclear", funcTag, 80},
	{"makechan64", funcTag, 82},
	{"makechan", funcTag, 83},
	{"chanrecv1", funcTag, 85},
	{"chanrecv2", funcTag, 86},
	{"chansend1", funcTag, 88},
	{"closechan", funcTag, 28},
	{"writeBarrier", varTag, 90},
	{"typedmemmove", funcTag, 91},
	{"typedmemclr", funcTag, 92},
	{"typedslicecopy", funcTag, 93},
	{"selectnbsend", funcTag, 94},
	{"selectnbrecv", funcTag, 95},
	{"selectnbrecv2", funcTag, 97},
	{"selectsetpc", funcTag, 62},
	{"selectgo", funcTag, 98},
	{"block", funcTag, 8},
	{"makeslice", funcTag, 99},
	{"makeslice64", funcTag, 100},
	{"growslice", funcTag, 102},
	{"memmove", funcTag, 103},
	{"memclrNoHeapPointers", funcTag, 104},
	{"memclrHasPointers", funcTag, 104},
	{"memequal", funcTag, 105},
	{"memequal0", funcTag, 106},
	{"memequal8", funcTag, 106},
	{"memequal16", funcTag, 106},
	{"memequal32", funcTag, 106},
	{"memequal64", funcTag, 106},
	{"memequal128", funcTag, 106},
	{"f32equal", funcTag, 107},
	{"f64equal", funcTag, 107},
	{"c64equal", funcTag, 107},
	{"c128equal", funcTag, 107},
	{"strequal", funcTag, 107},
	{"interequal", funcTag, 107},
	{"nilinterequal", funcTag, 107},
	{"memhash", funcTag, 108},
	{"memhash0", funcTag, 109},
	{"memhash8", funcTag, 109},
	{"memhash16", funcTag, 109},
	{"memhash32", funcTag, 109},
	{"memhash64", funcTag, 109},
	{"memhash128", funcTag, 109},
	{"f32hash", funcTag, 109},
	{"f64hash", funcTag, 109},
	{"c64hash", funcTag, 109},
	{"c128hash", funcTag, 109},
	{"strhash", funcTag, 109},
	{"interhash", funcTag, 109},
	{"nilinterhash", funcTag, 109},
	{"int64div", funcTag, 110},
	{"uint64div", funcTag, 111},
	{"int64mod", funcTag, 110},
	{"uint64mod", funcTag, 111},
	{"float64toint64", funcTag, 112},
	{"float64touint64", funcTag, 113},
	{"float64touint32", funcTag, 114},
	{"int64tofloat64", funcTag, 115},
	{"uint64tofloat64", funcTag, 116},
	{"uint32tofloat64", funcTag, 117},
	{"complex128div", funcTag, 118},
	{"racefuncenter", funcTag, 119},
	{"racefuncenterfp", funcTag, 8},
	{"racefuncexit", funcTag, 8},
	{"raceread", funcTag, 119},
	{"racewrite", funcTag, 119},
	{"racereadrange", funcTag, 120},
	{"racewriterange", funcTag, 120},
	{"msanread", funcTag, 120},
	{"msanwrite", funcTag, 120},
	{"checkptrAlignment", funcTag, 121},
	{"checkptrArithmetic", funcTag, 123},
	{"x86HasPOPCNT", varTag, 17},
	{"x86HasSSE41", varTag, 17},
	{"x86HasFMA", varTag, 17},
	{"armHasVFPv4", varTag, 17},
	{"arm64HasATOMICS", varTag, 17},
}

func runtimeTypes() []*types.Type {
	var typs [124]*types.Type
	typs[0] = types.Types[TSTRING]
	typs[1] = functype(nil, []*Node{anonfield(typs[0]), anonfield(typs[0]), anonfield(typs[0])}, nil)
	typs[2] = types.Bytetype
	typs[3] = types.NewPtr(typs[2])
	typs[4] = types.Types[TINT]
	typs[5] = types.Types[TANY]
	typs[6] = types.NewPtr(typs[5])
	typs[7] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[4])}, []*Node{anonfield(typs[6])})
	typs[8] = functype(nil, nil, nil)
	typs[9] = types.Types[TINTER]
	typs[10] = functype(nil, []*Node{anonfield(typs[9])}, nil)
	typs[11] = types.Types[TINT32]
	typs[12] = types.NewPtr(typs[11])
	typs[13] = functype(nil, []*Node{anonfield(typs[12])}, []*Node{anonfield(typs[9])})
	typs[14] = functype(nil, []*Node{anonfield(typs[4]), anonfield(typs[4])}, nil)
	typs[15] = types.Types[TUINT]
	typs[16] = functype(nil, []*Node{anonfield(typs[15]), anonfield(typs[4])}, nil)
	typs[17] = types.Types[TBOOL]
	typs[18] = functype(nil, []*Node{anonfield(typs[17])}, nil)
	typs[19] = types.Types[TFLOAT64]
	typs[20] = functype(nil, []*Node{anonfield(typs[19])}, nil)
	typs[21] = types.Types[TINT64]
	typs[22] = functype(nil, []*Node{anonfield(typs[21])}, nil)
	typs[23] = types.Types[TUINT64]
	typs[24] = functype(nil, []*Node{anonfield(typs[23])}, nil)
	typs[25] = types.Types[TCOMPLEX128]
	typs[26] = functype(nil, []*Node{anonfield(typs[25])}, nil)
	typs[27] = functype(nil, []*Node{anonfield(typs[0])}, nil)
	typs[28] = functype(nil, []*Node{anonfield(typs[5])}, nil)
	typs[29] = types.NewArray(typs[2], 32)
	typs[30] = types.NewPtr(typs[29])
	typs[31] = functype(nil, []*Node{anonfield(typs[30]), anonfield(typs[0]), anonfield(typs[0])}, []*Node{anonfield(typs[0])})
	typs[32] = functype(nil, []*Node{anonfield(typs[30]), anonfield(typs[0]), anonfield(typs[0]), anonfield(typs[0])}, []*Node{anonfield(typs[0])})
	typs[33] = functype(nil, []*Node{anonfield(typs[30]), anonfield(typs[0]), anonfield(typs[0]), anonfield(typs[0]), anonfield(typs[0])}, []*Node{anonfield(typs[0])})
	typs[34] = functype(nil, []*Node{anonfield(typs[30]), anonfield(typs[0]), anonfield(typs[0]), anonfield(typs[0]), anonfield(typs[0]), anonfield(typs[0])}, []*Node{anonfield(typs[0])})
	typs[35] = types.NewSlice(typs[0])
	typs[36] = functype(nil, []*Node{anonfield(typs[30]), anonfield(typs[35])}, []*Node{anonfield(typs[0])})
	typs[37] = functype(nil, []*Node{anonfield(typs[0]), anonfield(typs[0])}, []*Node{anonfield(typs[4])})
	typs[38] = types.NewArray(typs[2], 4)
	typs[39] = types.NewPtr(typs[38])
	typs[40] = functype(nil, []*Node{anonfield(typs[39]), anonfield(typs[21])}, []*Node{anonfield(typs[0])})
	typs[41] = types.NewSlice(typs[2])
	typs[42] = functype(nil, []*Node{anonfield(typs[30]), anonfield(typs[41])}, []*Node{anonfield(typs[0])})
	typs[43] = functype(nil, []*Node{anonfield(typs[41])}, []*Node{anonfield(typs[0])})
	typs[44] = types.Runetype
	typs[45] = types.NewSlice(typs[44])
	typs[46] = functype(nil, []*Node{anonfield(typs[30]), anonfield(typs[45])}, []*Node{anonfield(typs[0])})
	typs[47] = functype(nil, []*Node{anonfield(typs[30]), anonfield(typs[0])}, []*Node{anonfield(typs[41])})
	typs[48] = types.NewArray(typs[44], 32)
	typs[49] = types.NewPtr(typs[48])
	typs[50] = functype(nil, []*Node{anonfield(typs[49]), anonfield(typs[0])}, []*Node{anonfield(typs[45])})
	typs[51] = types.Types[TUINTPTR]
	typs[52] = functype(nil, []*Node{anonfield(typs[5]), anonfield(typs[5]), anonfield(typs[51])}, []*Node{anonfield(typs[4])})
	typs[53] = functype(nil, []*Node{anonfield(typs[5]), anonfield(typs[5])}, []*Node{anonfield(typs[4])})
	typs[54] = functype(nil, []*Node{anonfield(typs[0]), anonfield(typs[4])}, []*Node{anonfield(typs[44]), anonfield(typs[4])})
	typs[55] = functype(nil, []*Node{anonfield(typs[0])}, []*Node{anonfield(typs[4])})
	typs[56] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[5])}, []*Node{anonfield(typs[5])})
	typs[57] = types.Types[TUNSAFEPTR]
	typs[58] = functype(nil, []*Node{anonfield(typs[5])}, []*Node{anonfield(typs[57])})
	typs[59] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[6])}, []*Node{anonfield(typs[5])})
	typs[60] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[5])}, []*Node{anonfield(typs[5]), anonfield(typs[17])})
	typs[61] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[3]), anonfield(typs[3])}, nil)
	typs[62] = functype(nil, []*Node{anonfield(typs[3])}, nil)
	typs[63] = types.NewPtr(typs[51])
	typs[64] = functype(nil, []*Node{anonfield(typs[63]), anonfield(typs[57]), anonfield(typs[57])}, []*Node{anonfield(typs[17])})
	typs[65] = types.Types[TUINT32]
	typs[66] = functype(nil, nil, []*Node{anonfield(typs[65])})
	typs[67] = types.NewMap(typs[5], typs[5])
	typs[68] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[21]), anonfield(typs[6])}, []*Node{anonfield(typs[67])})
	typs[69] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[4]), anonfield(typs[6])}, []*Node{anonfield(typs[67])})
	typs[70] = functype(nil, nil, []*Node{anonfield(typs[67])})
	typs[71] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67]), anonfield(typs[6])}, []*Node{anonfield(typs[6])})
	typs[72] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67]), anonfield(typs[5])}, []*Node{anonfield(typs[6])})
	typs[73] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67]), anonfield(typs[6]), anonfield(typs[3])}, []*Node{anonfield(typs[6])})
	typs[74] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67]), anonfield(typs[6])}, []*Node{anonfield(typs[6]), anonfield(typs[17])})
	typs[75] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67]), anonfield(typs[5])}, []*Node{anonfield(typs[6]), anonfield(typs[17])})
	typs[76] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67]), anonfield(typs[6]), anonfield(typs[3])}, []*Node{anonfield(typs[6]), anonfield(typs[17])})
	typs[77] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67]), anonfield(typs[6])}, nil)
	typs[78] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67]), anonfield(typs[5])}, nil)
	typs[79] = functype(nil, []*Node{anonfield(typs[6])}, nil)
	typs[80] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[67])}, nil)
	typs[81] = types.NewChan(typs[5], types.Cboth)
	typs[82] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[21])}, []*Node{anonfield(typs[81])})
	typs[83] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[4])}, []*Node{anonfield(typs[81])})
	typs[84] = types.NewChan(typs[5], types.Crecv)
	typs[85] = functype(nil, []*Node{anonfield(typs[84]), anonfield(typs[6])}, nil)
	typs[86] = functype(nil, []*Node{anonfield(typs[84]), anonfield(typs[6])}, []*Node{anonfield(typs[17])})
	typs[87] = types.NewChan(typs[5], types.Csend)
	typs[88] = functype(nil, []*Node{anonfield(typs[87]), anonfield(typs[6])}, nil)
	typs[89] = types.NewArray(typs[2], 3)
	typs[90] = tostruct([]*Node{namedfield("enabled", typs[17]), namedfield("pad", typs[89]), namedfield("needed", typs[17]), namedfield("cgo", typs[17]), namedfield("alignme", typs[23])})
	typs[91] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[6]), anonfield(typs[6])}, nil)
	typs[92] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[6])}, nil)
	typs[93] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[5]), anonfield(typs[5])}, []*Node{anonfield(typs[4])})
	typs[94] = functype(nil, []*Node{anonfield(typs[87]), anonfield(typs[6])}, []*Node{anonfield(typs[17])})
	typs[95] = functype(nil, []*Node{anonfield(typs[6]), anonfield(typs[84])}, []*Node{anonfield(typs[17])})
	typs[96] = types.NewPtr(typs[17])
	typs[97] = functype(nil, []*Node{anonfield(typs[6]), anonfield(typs[96]), anonfield(typs[84])}, []*Node{anonfield(typs[17])})
	typs[98] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[3]), anonfield(typs[4])}, []*Node{anonfield(typs[4]), anonfield(typs[17])})
	typs[99] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[4]), anonfield(typs[4])}, []*Node{anonfield(typs[57])})
	typs[100] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[21]), anonfield(typs[21])}, []*Node{anonfield(typs[57])})
	typs[101] = types.NewSlice(typs[5])
	typs[102] = functype(nil, []*Node{anonfield(typs[3]), anonfield(typs[101]), anonfield(typs[4])}, []*Node{anonfield(typs[101])})
	typs[103] = functype(nil, []*Node{anonfield(typs[6]), anonfield(typs[6]), anonfield(typs[51])}, nil)
	typs[104] = functype(nil, []*Node{anonfield(typs[57]), anonfield(typs[51])}, nil)
	typs[105] = functype(nil, []*Node{anonfield(typs[6]), anonfield(typs[6]), anonfield(typs[51])}, []*Node{anonfield(typs[17])})
	typs[106] = functype(nil, []*Node{anonfield(typs[6]), anonfield(typs[6])}, []*Node{anonfield(typs[17])})
	typs[107] = functype(nil, []*Node{anonfield(typs[57]), anonfield(typs[57])}, []*Node{anonfield(typs[17])})
	typs[108] = functype(nil, []*Node{anonfield(typs[57]), anonfield(typs[51]), anonfield(typs[51])}, []*Node{anonfield(typs[51])})
	typs[109] = functype(nil, []*Node{anonfield(typs[57]), anonfield(typs[51])}, []*Node{anonfield(typs[51])})
	typs[110] = functype(nil, []*Node{anonfield(typs[21]), anonfield(typs[21])}, []*Node{anonfield(typs[21])})
	typs[111] = functype(nil, []*Node{anonfield(typs[23]), anonfield(typs[23])}, []*Node{anonfield(typs[23])})
	typs[112] = functype(nil, []*Node{anonfield(typs[19])}, []*Node{anonfield(typs[21])})
	typs[113] = functype(nil, []*Node{anonfield(typs[19])}, []*Node{anonfield(typs[23])})
	typs[114] = functype(nil, []*Node{anonfield(typs[19])}, []*Node{anonfield(typs[65])})
	typs[115] = functype(nil, []*Node{anonfield(typs[21])}, []*Node{anonfield(typs[19])})
	typs[116] = functype(nil, []*Node{anonfield(typs[23])}, []*Node{anonfield(typs[19])})
	typs[117] = functype(nil, []*Node{anonfield(typs[65])}, []*Node{anonfield(typs[19])})
	typs[118] = functype(nil, []*Node{anonfield(typs[25]), anonfield(typs[25])}, []*Node{anonfield(typs[25])})
	typs[119] = functype(nil, []*Node{anonfield(typs[51])}, nil)
	typs[120] = functype(nil, []*Node{anonfield(typs[51]), anonfield(typs[51])}, nil)
	typs[121] = functype(nil, []*Node{anonfield(typs[57]), anonfield(typs[3]), anonfield(typs[51])}, nil)
	typs[122] = types.NewSlice(typs[57])
	typs[123] = functype(nil, []*Node{anonfield(typs[57]), anonfield(typs[122])}, nil)
	return typs[:]
}