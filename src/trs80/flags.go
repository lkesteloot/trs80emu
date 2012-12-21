package main

// http://www.zilog.com/docs/z80/um0080.pdf page 76
type flags byte

const (
	carryShift, carryMask = iota, 1 << iota
	subtractShift, subtractMask
	parityOverflowShift, parityOverflowMask
	undoc3Shift, undoc3Mask
	halfCarryShift, halfCarryMask
	undoc5Shift, undoc5Mask
	zeroShift, zeroMask
	signShift, signMask

	undocMasks = undoc3Mask | undoc5Mask
)

var signCarryOverflowTable = []flags{
	0,
	parityOverflowMask | signMask,
	carryMask,
	signMask,
	carryMask,
	signMask,
	carryMask | parityOverflowMask,
	carryMask | signMask,
}

var halfCarryTable = []flags{
	0,
	0,
	halfCarryMask,
	0,
	halfCarryMask,
	0,
	halfCarryMask,
	halfCarryMask,
}

var subtractSignCarryOverflowTable = []flags{
	0,
	carryMask | signMask,
	carryMask,
	parityOverflowMask | carryMask | signMask,
	parityOverflowMask,
	signMask,
	0,
	carryMask | signMask,
}

var subtractHalfCarryTable = []flags{
	0,
	halfCarryMask,
	halfCarryMask,
	halfCarryMask,
	0,
	0,
	0,
	halfCarryMask,
}

// For parity flag. 1 = even parity, 0 = odd parity.
var parityTable = [...]byte{
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
}

// Get carry flag.
func (f flags) c() bool {
	return (f & carryMask) != 0
}

// Get add/subtract flag.
func (f flags) n() bool {
	return (f & subtractMask) != 0
}

// Get parity/overflow flag.
func (f flags) pv() bool {
	return (f & parityOverflowMask) != 0
}

// Get half carry flag.
func (f flags) h() bool {
	return (f & halfCarryMask) != 0
}

// Get zero flag.
func (f flags) z() bool {
	return (f & zeroMask) != 0
}

// Get sign flag.
func (f flags) s() bool {
	return (f & signMask) != 0
}

// Set carry flag.
func (f *flags) setC(c bool) {
	if c {
		*f |= carryMask
	} else {
		*f &^= carryMask
	}
}

// Set add/subtract flag.
func (f *flags) setN(n bool) {
	if n {
		*f |= subtractMask
	} else {
		*f &^= subtractMask
	}
}

// Set parity/overflow flag.
func (f *flags) setPv(pv bool) {
	if pv {
		*f |= parityOverflowMask
	} else {
		*f &^= parityOverflowMask
	}
}

// Set half carry flag.
func (f *flags) setH(h bool) {
	if h {
		*f |= halfCarryMask
	} else {
		*f &^= halfCarryMask
	}
}

// Set zero flag.
func (f *flags) setZ(z bool) {
	if z {
		*f |= zeroMask
	} else {
		*f &^= zeroMask
	}
}

// Set sign flag.
func (f *flags) setS(s bool) {
	if s {
		*f |= signMask
	} else {
		*f &^= signMask
	}
}

// Update simple flags (S, Z, P, and undoc) based on result of operation.
// Carry is unaffected.
func (f *flags) updateFromByte(value byte) {
	*f &= carryMask
	f.setS(value&0x80 != 0)
	f.setZ(value == 0)
	f.setPv(parityTable[value] == 1)
	*f |= flags(value & undocMasks)
}

func (f *flags) updateFromAddByte(value1, value2, result byte) {
	index := (value1&0x88)>>1 | (value2&0x88)>>2 | (result&0x88)>>3
	*f = halfCarryTable[index&7] |
		signCarryOverflowTable[index>>4] |
		flags(result&undocMasks)

	if result == 0 {
		*f |= zeroMask
	}
}

func (f *flags) updateFromAddWord(value1, value2, result word) {
	index := (value1&0x8800)>>9 | (value2&0x8800)>>10 | (result&0x8800)>>11
	*f = halfCarryTable[index&7] |
		(signCarryOverflowTable[index>>4] & carryMask) |
		(*f & (zeroMask | parityOverflowMask | signMask)) |
		flags(result.h()&undocMasks)
}

func (f *flags) updateFromSubByte(value1, value2, result byte) {
	index := (value1&0x88)>>1 | (value2&0x88)>>2 | (result&0x88)>>3
	*f = subtractMask |
		subtractHalfCarryTable[index&7] |
		subtractSignCarryOverflowTable[index>>4] |
		flags(result&undocMasks)

	if result == 0 {
		*f |= zeroMask
	}
}

func (f *flags) updateFromSbcWord(value1, value2, result word) {
	index := (value1&0x8800)>>9 | (value2&0x8800)>>10 | (result&0x8800)>>11
	*f = subtractMask |
		subtractHalfCarryTable[index&7] |
		subtractSignCarryOverflowTable[index>>4] |
		flags(result.h()&undocMasks)

	if result == 0 {
		*f |= zeroMask
	}
}

func (f *flags) updateFromLogicByte(result byte, isAdd bool) {
	if isAdd {
		*f = halfCarryMask
	} else {
		*f = 0
	}

	*f |= flags(parityTable[result] << parityOverflowShift)
	if result == 0 {
		*f |= zeroMask
	}
	if result&0x80 != 0 {
		*f |= signMask
	}

	*f |= flags(result & undocMasks)
}

func (f *flags) updateFromDecByte(result byte) {
	*f = (*f & carryMask) | subtractMask

	if result == 0x7F {
		*f |= parityOverflowMask
	}
	if result&0x0F == 0x0F {
		*f |= halfCarryMask
	}
	if result == 0 {
		*f |= zeroMask
	}
	if result&0x80 != 0 {
		*f |= signMask
	}

	*f |= flags(result & undocMasks)
}

func (f *flags) updateFromIncByte(result byte) {
	*f &= carryMask

	if result == 0x80 {
		*f |= parityOverflowMask
	}
	if result&0x0F == 0 {
		*f |= halfCarryMask
	}
	if result == 0 {
		*f |= zeroMask
	}
	if result&0x80 != 0 {
		*f |= signMask
	}

	*f |= flags(result & undocMasks)
}

func (f *flags) updateFromInByte(result byte) {
	*f &^= signMask | zeroMask | halfCarryMask | parityOverflowMask | subtractMask

	if result&0x80 != 0 {
		*f |= signMask
	}
	if result == 0 {
		*f |= zeroMask
	}
	*f |= flags(parityTable[result] << parityOverflowShift)
}
