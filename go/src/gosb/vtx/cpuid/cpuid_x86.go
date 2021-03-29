package cpuid

import (
	"fmt"
)

// Common references for CPUID leaves and bits:
//
// Intel:
//   * Intel SDM Volume 2, Chapter 3.2 "CPUID" (more up-to-date)
//   * Intel Application Note 485 (more detailed)
//
// AMD:
//   * AMD64 APM Volume 3, Appendix 3 "Obtaining Processor Information ..."

// block is a collection of 32 Feature bits.
type block int

const blockSize = 32

// Feature bits are numbered according to "blocks". Each block is 32 bits, and
// feature bits from the same source (cpuid leaf/level) are in the same block.
func featureID(b block, bit int) Feature {
	return Feature(32*int(b) + bit)
}

// Block 0 constants are all of the "basic" feature bits returned by a cpuid in
// ecx with eax=1.
const (
	X86FeatureSSE3 Feature = iota
	X86FeaturePCLMULDQ
	X86FeatureDTES64
	X86FeatureMONITOR
	X86FeatureDSCPL
	X86FeatureVMX
	X86FeatureSMX
	X86FeatureEST
	X86FeatureTM2
	X86FeatureSSSE3 // Not a typo, "supplemental" SSE3.
	X86FeatureCNXTID
	X86FeatureSDBG
	X86FeatureFMA
	X86FeatureCX16
	X86FeatureXTPR
	X86FeaturePDCM
	_ // ecx bit 16 is reserved.
	X86FeaturePCID
	X86FeatureDCA
	X86FeatureSSE4_1
	X86FeatureSSE4_2
	X86FeatureX2APIC
	X86FeatureMOVBE
	X86FeaturePOPCNT
	X86FeatureTSCD
	X86FeatureAES
	X86FeatureXSAVE
	X86FeatureOSXSAVE
	X86FeatureAVX
	X86FeatureF16C
	X86FeatureRDRAND
	_ // ecx bit 31 is reserved.
)

// Block 1 constants are all of the "basic" feature bits returned by a cpuid in
// edx with eax=1.
const (
	X86FeatureFPU Feature = 32 + iota
	X86FeatureVME
	X86FeatureDE
	X86FeaturePSE
	X86FeatureTSC
	X86FeatureMSR
	X86FeaturePAE
	X86FeatureMCE
	X86FeatureCX8
	X86FeatureAPIC
	_ // edx bit 10 is reserved.
	X86FeatureSEP
	X86FeatureMTRR
	X86FeaturePGE
	X86FeatureMCA
	X86FeatureCMOV
	X86FeaturePAT
	X86FeaturePSE36
	X86FeaturePSN
	X86FeatureCLFSH
	_ // edx bit 20 is reserved.
	X86FeatureDS
	X86FeatureACPI
	X86FeatureMMX
	X86FeatureFXSR
	X86FeatureSSE
	X86FeatureSSE2
	X86FeatureSS
	X86FeatureHTT
	X86FeatureTM
	X86FeatureIA64
	X86FeaturePBE
)

// Block 2 bits are the "structured extended" features returned in ebx for
// eax=7, ecx=0.
const (
	X86FeatureFSGSBase Feature = 2*32 + iota
	X86FeatureTSC_ADJUST
	_ // ebx bit 2 is reserved.
	X86FeatureBMI1
	X86FeatureHLE
	X86FeatureAVX2
	X86FeatureFDP_EXCPTN_ONLY
	X86FeatureSMEP
	X86FeatureBMI2
	X86FeatureERMS
	X86FeatureINVPCID
	X86FeatureRTM
	X86FeatureCQM
	X86FeatureFPCSDS
	X86FeatureMPX
	X86FeatureRDT
	X86FeatureAVX512F
	X86FeatureAVX512DQ
	X86FeatureRDSEED
	X86FeatureADX
	X86FeatureSMAP
	X86FeatureAVX512IFMA
	X86FeaturePCOMMIT
	X86FeatureCLFLUSHOPT
	X86FeatureCLWB
	X86FeatureIPT // Intel processor trace.
	X86FeatureAVX512PF
	X86FeatureAVX512ER
	X86FeatureAVX512CD
	X86FeatureSHA
	X86FeatureAVX512BW
	X86FeatureAVX512VL
)

// Block 4 constants are for xsave capabilities in CPUID.(EAX=0DH,ECX=01H):EAX.
// The CPUID leaf is available only if 'X86FeatureXSAVE' is present.
const (
	X86FeatureXSAVEOPT Feature = 4*32 + iota
	X86FeatureXSAVEC
	X86FeatureXGETBV1
	X86FeatureXSAVES
	// EAX[31:4] are reserved.
)

// Block 5 constants are the extended feature bits in
// CPUID.(EAX=0x80000001):ECX.
const (
	X86FeatureLAHF64 Feature = 5*32 + iota
	X86FeatureCMP_LEGACY
	X86FeatureSVM
	X86FeatureEXTAPIC
	X86FeatureCR8_LEGACY
	X86FeatureLZCNT
	X86FeatureSSE4A
	X86FeatureMISALIGNSSE
	X86FeaturePREFETCHW
	X86FeatureOSVW
	X86FeatureIBS
	X86FeatureXOP
	X86FeatureSKINIT
	X86FeatureWDT
	_ // ecx bit 14 is reserved.
	X86FeatureLWP
	X86FeatureFMA4
	X86FeatureTCE
	_ // ecx bit 18 is reserved.
	_ // ecx bit 19 is reserved.
	_ // ecx bit 20 is reserved.
	X86FeatureTBM
	X86FeatureTOPOLOGY
	X86FeaturePERFCTR_CORE
	X86FeaturePERFCTR_NB
	_ // ecx bit 25 is reserved.
	X86FeatureBPEXT
	X86FeaturePERFCTR_TSC
	X86FeaturePERFCTR_LLC
	X86FeatureMWAITX
	// ECX[31:30] are reserved.
)

// CacheType describes the type of a cache, as returned in eax[4:0] for eax=4.
type CacheType uint8

const (
	amdVendorID   = "AuthenticAMD"
	intelVendorID = "GenuineIntel"
)

const (
	// cacheNull indicates that there are no more entries.
	cacheNull CacheType = iota

	// CacheData is a data cache.
	CacheData

	// CacheInstruction is an instruction cache.
	CacheInstruction

	// CacheUnified is a unified instruction and data cache.
	CacheUnified
)

// Cache describes the parameters of a single cache on the system.
//
// +stateify savable
type Cache struct {
	// Level is the hierarchical level of this cache (L1, L2, etc).
	Level uint32

	// Type is the type of cache.
	Type CacheType

	// FullyAssociative indicates that entries may be placed in any block.
	FullyAssociative bool

	// Partitions is the number of physical partitions in the cache.
	Partitions uint32

	// Ways is the number of ways of associativity in the cache.
	Ways uint32

	// Sets is the number of sets in the cache.
	Sets uint32

	// InvalidateHierarchical indicates that WBINVD/INVD from threads
	// sharing this cache acts upon lower level caches for threads sharing
	// this cache.
	InvalidateHierarchical bool

	// Inclusive indicates that this cache is inclusive of lower cache
	// levels.
	Inclusive bool

	// DirectMapped indicates that this cache is directly mapped from
	// address, rather than using a hash function.
	DirectMapped bool
}

// Feature is a unique identifier for a particular cpu feature. We just use an
// int as a feature number on x86 and arm64.
//
// On x86, features are numbered according to "blocks". Each block is 32 bits, and
// feature bits from the same source (cpuid leaf/level) are in the same block.
//
// On arm64, features are numbered according to the ELF HWCAP definition.
// arch/arm64/include/uapi/asm/hwcap.h
type Feature int

// FeatureSet is a set of Features for a CPU.
//
// +stateify savable
type FeatureSet struct {
	// Set is the set of features that are enabled in this FeatureSet.
	Set map[Feature]bool

	// VendorID is the 12-char string returned in ebx:edx:ecx for eax=0.
	VendorID string

	// ExtendedFamily is part of the processor signature.
	ExtendedFamily uint8

	// ExtendedModel is part of the processor signature.
	ExtendedModel uint8

	// ProcessorType is part of the processor signature.
	ProcessorType uint8

	// Family is part of the processor signature.
	Family uint8

	// Model is part of the processor signature.
	Model uint8

	// SteppingID is part of the processor signature.
	SteppingID uint8

	// Caches describes the caches on the CPU.
	Caches []Cache

	// CacheLine is the size of a cache line in bytes.
	//
	// All caches use the same line size. This is not enforced in the CPUID
	// encoding, but is true on all known x86 processors.
	CacheLine uint32
}

// HostFeatureSet uses cpuid to get host values and construct a feature set
// that matches that of the host machine. Note that there are several places
// where there appear to be some unnecessary assignments between register names
// (ax, bx, cx, or dx) and featureBlockN variables. This is to explicitly show
// where the different feature blocks come from, to make the code easier to
// inspect and read.
func HostFeatureSet() *FeatureSet {
	// eax=0 gets max supported feature and vendor ID.
	_, bx, cx, dx := HostID(0, 0)
	vendorID := vendorIDFromRegs(bx, cx, dx)

	// eax=1 gets basic features in ecx:edx.
	ax, bx, cx, dx := HostID(1, 0)
	featureBlock0 := cx
	featureBlock1 := dx
	ef, em, pt, f, m, sid := signatureSplit(ax)
	cacheLine := 8 * (bx >> 8) & 0xff

	// eax=4, ecx=i gets details about cache index i. Only supported on Intel.
	var caches []Cache
	if vendorID == intelVendorID {
		// ecx selects the cache index until a null type is returned.
		for i := uint32(0); ; i++ {
			ax, bx, cx, dx := HostID(4, i)
			t := CacheType(ax & 0xf)
			if t == cacheNull {
				break
			}

			lineSize := (bx & 0xfff) + 1
			if lineSize != cacheLine {
				panic(fmt.Sprintf("Mismatched cache line size: %d vs %d", lineSize, cacheLine))
			}

			caches = append(caches, Cache{
				Type:                   t,
				Level:                  (ax >> 5) & 0x7,
				FullyAssociative:       ((ax >> 9) & 1) == 1,
				Partitions:             ((bx >> 12) & 0x3ff) + 1,
				Ways:                   ((bx >> 22) & 0x3ff) + 1,
				Sets:                   cx + 1,
				InvalidateHierarchical: (dx & 1) == 0,
				Inclusive:              ((dx >> 1) & 1) == 1,
				DirectMapped:           ((dx >> 2) & 1) == 0,
			})
		}
	}

	// eax=7, ecx=0 gets extended features in ecx:ebx.
	_, bx, cx, _ = HostID(7, 0)
	featureBlock2 := bx
	featureBlock3 := cx

	// Leaf 0xd is supported only if CPUID.1:ECX.XSAVE[bit 26] is set.
	var featureBlock4 uint32
	if (featureBlock0 & (1 << 26)) != 0 {
		featureBlock4, _, _, _ = HostID(uint32(xSaveInfo), 1)
	}

	// eax=0x80000000 gets supported extended levels. We use this to
	// determine if there are any non-zero block 4 or block 6 bits to find.
	var featureBlock5, featureBlock6 uint32
	if ax, _, _, _ := HostID(uint32(extendedFunctionInfo), 0); ax >= uint32(extendedFeatures) {
		// eax=0x80000001 gets AMD added feature bits.
		_, _, cx, dx = HostID(uint32(extendedFeatures), 0)
		featureBlock5 = cx
		// Ignore features duplicated from block 1 on AMD. These bits
		// are reserved on Intel.
		featureBlock6 = dx &^ block6DuplicateMask
	}

	set := setFromBlockMasks(featureBlock0, featureBlock1, featureBlock2, featureBlock3, featureBlock4, featureBlock5, featureBlock6)
	return &FeatureSet{
		Set:            set,
		VendorID:       vendorID,
		ExtendedFamily: ef,
		ExtendedModel:  em,
		ProcessorType:  pt,
		Family:         f,
		Model:          m,
		SteppingID:     sid,
		CacheLine:      cacheLine,
		Caches:         caches,
	}
}

// UseXsave returns the choice of fp state saving instruction.
func (fs *FeatureSet) UseXsave() bool {
	return fs.HasFeature(X86FeatureXSAVE) && fs.HasFeature(X86FeatureOSXSAVE)
}

// UseXsaveopt returns true if 'fs' supports the "xsaveopt" instruction.
func (fs *FeatureSet) UseXsaveopt() bool {
	return fs.UseXsave() && fs.HasFeature(X86FeatureXSAVEOPT)
}

// HostID executes a native CPUID instruction.
func HostID(axArg, cxArg uint32) (ax, bx, cx, dx uint32)

// Helper to deconstruct signature dword.
func signatureSplit(v uint32) (ef, em, pt, f, m, sid uint8) {
	sid = uint8(v & 0xf)
	m = uint8(v>>4) & 0xf
	f = uint8(v>>8) & 0xf
	pt = uint8(v>>12) & 0x3
	em = uint8(v>>16) & 0xf
	ef = uint8(v >> 20)
	return
}

// Helper to convert 3 regs into 12-byte vendor ID.
func vendorIDFromRegs(bx, cx, dx uint32) string {
	bytes := make([]byte, 0, 12)
	for i := uint(0); i < 4; i++ {
		b := byte(bx >> (i * 8))
		bytes = append(bytes, b)
	}

	for i := uint(0); i < 4; i++ {
		b := byte(dx >> (i * 8))
		bytes = append(bytes, b)
	}

	for i := uint(0); i < 4; i++ {
		b := byte(cx >> (i * 8))
		bytes = append(bytes, b)
	}
	return string(bytes)
}

// Just a way to wrap cpuid function numbers.
type cpuidFunction uint32

// The constants below are the lower or "standard" cpuid functions, ordered as
// defined by the hardware.
const (
	vendorID                      cpuidFunction = iota // Returns vendor ID and largest standard function.
	featureInfo                                        // Returns basic feature bits and processor signature.
	intelCacheDescriptors                              // Returns list of cache descriptors. Intel only.
	intelSerialNumber                                  // Returns processor serial number (obsolete on new hardware). Intel only.
	intelDeterministicCacheParams                      // Returns deterministic cache information. Intel only.
	monitorMwaitParams                                 // Returns information about monitor/mwait instructions.
	powerParams                                        // Returns information about power management and thermal sensors.
	extendedFeatureInfo                                // Returns extended feature bits.
	_                                                  // Function 0x8 is reserved.
	intelDCAParams                                     // Returns direct cache access information. Intel only.
	intelPMCInfo                                       // Returns information about performance monitoring features. Intel only.
	intelX2APICInfo                                    // Returns core/logical processor topology. Intel only.
	_                                                  // Function 0xc is reserved.
	xSaveInfo                                          // Returns information about extended state management.
)

// The "extended" functions start at 0x80000000.
const (
	extendedFunctionInfo cpuidFunction = 0x80000000 + iota // Returns highest available extended function in eax.
	extendedFeatures                                       // Returns some extended feature bits in edx and ecx.
)

// Block 6 constants are the extended feature bits in
// CPUID.(EAX=0x80000001):EDX.
//
// These are sparse, and so the bit positions are assigned manually.
const (
	// On AMD, EDX[24:23] | EDX[17:12] | EDX[9:0] are duplicate features
	// also defined in block 1 (in identical bit positions). Those features
	// are not listed here.
	block6DuplicateMask = 0x183f3ff

	X86FeatureSYSCALL  Feature = 6*32 + 11
	X86FeatureNX       Feature = 6*32 + 20
	X86FeatureMMXEXT   Feature = 6*32 + 22
	X86FeatureFXSR_OPT Feature = 6*32 + 25
	X86FeatureGBPAGES  Feature = 6*32 + 26
	X86FeatureRDTSCP   Feature = 6*32 + 27
	X86FeatureLM       Feature = 6*32 + 29
	X86Feature3DNOWEXT Feature = 6*32 + 30
	X86Feature3DNOW    Feature = 6*32 + 31
)

// Helper to convert blockwise feature bit masks into a set of features. Masks
// must be provided in order for each block, without skipping them. If a block
// does not matter for this feature set, 0 is specified.
func setFromBlockMasks(blocks ...uint32) map[Feature]bool {
	s := make(map[Feature]bool)
	for b, blockMask := range blocks {
		for i := 0; i < blockSize; i++ {
			if blockMask&1 != 0 {
				s[featureID(block(b), i)] = true
			}
			blockMask >>= 1
		}
	}
	return s
}

// HasFeature tests whether or not a feature is in the given feature set.
func (fs *FeatureSet) HasFeature(feature Feature) bool {
	return fs.Set[feature]
}

var maxXsaveSize = func() uint32 {
	// Leaf 0 of xsaveinfo function returns the size for currently
	// enabled xsave features in ebx, the maximum size if all valid
	// features are saved with xsave in ecx, and valid XCR0 bits in
	// edx:eax.
	//
	// If xSaveInfo isn't supported, cpuid will not fault but will
	// return bogus values.
	_, _, maxXsaveSize, _ := HostID(uint32(xSaveInfo), 0)
	return maxXsaveSize
}()

// ExtendedStateSize returns the number of bytes needed to save the "extended
// state" for this processor and the boundary it must be aligned to. Extended
// state includes floating point registers, and other cpu state that's not
// associated with the normal task context.
//
// Note: We can save some space here with an optimization where we use a
// smaller chunk of memory depending on features that are actually enabled.
// Currently we just use the largest possible size for simplicity (which is
// about 2.5K worst case, with avx512).
func (fs *FeatureSet) ExtendedStateSize() (size, align uint) {
	if fs.UseXsave() {
		return uint(maxXsaveSize), 64
	}

	// If we don't support xsave, we fall back to fxsave, which requires
	// 512 bytes aligned to 16 bytes.
	return 512, 16
}

// ValidXCR0Mask returns the bits that may be set to 1 in control register
// XCR0.
func (fs *FeatureSet) ValidXCR0Mask() uint64 {
	if !fs.UseXsave() {
		return 0
	}
	eax, _, _, edx := HostID(uint32(xSaveInfo), 0)
	return uint64(edx)<<32 | uint64(eax)
}

// These are the extended floating point state features. They are used to
// enumerate floating point features in XCR0, XSTATE_BV, etc.
const (
	XSAVEFeatureX87         = 1 << 0
	XSAVEFeatureSSE         = 1 << 1
	XSAVEFeatureAVX         = 1 << 2
	XSAVEFeatureBNDREGS     = 1 << 3
	XSAVEFeatureBNDCSR      = 1 << 4
	XSAVEFeatureAVX512op    = 1 << 5
	XSAVEFeatureAVX512zmm0  = 1 << 6
	XSAVEFeatureAVX512zmm16 = 1 << 7
	XSAVEFeaturePKRU        = 1 << 9
)
