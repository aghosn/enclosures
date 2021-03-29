#include "textflag.h"

TEXT ·WritePKRU(SB),$0
	MOVQ prot+0(FP), AX
	XORQ CX, CX
    XORQ DX, DX
	BYTE $0x0f; BYTE $0x01; BYTE $0xef // WRPKRU
	RET
