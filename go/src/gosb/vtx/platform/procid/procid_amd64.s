#include "textflag.h"

TEXT ·Current(SB),NOSPLIT,$0-8
	// The offset specified here is the m_procid offset for Go1.8+.
	// Changes to this offset should be caught by the tests, and major
	// version changes require an explicit tag change above.
	MOVQ TLS, AX
	MOVQ 0(AX)(TLS*1), AX
	MOVQ 48(AX), AX // g_m (may change in future versions)
	MOVQ 72(AX), AX // m_procid (may change in future versions)
	MOVQ AX, ret+0(FP)
	RET
