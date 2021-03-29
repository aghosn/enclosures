package usermem

import "encoding/binary"

const (
	// PageSize is the system page size.
	PageSize = 1 << PageShift

	// HugePageSize is the system huge page size.
	HugePageSize = 1 << HugePageShift

	// PageShift is the binary log of the system page size.
	PageShift = 12

	// HugePageShift is the binary log of the system huge page size.
	HugePageShift = 21
)

var (
	// ByteOrder is the native byte order (little endian).
	ByteOrder = binary.LittleEndian
)
