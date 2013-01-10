// Copyright 2012 Lawrence Kesteloot

package main

// 8-bit flag register with helper functions.
// More info here: http://www.zilog.com/docs/z80/um0080.pdf page 76
type flags byte

// Flags and masks.
const (
	carryShift, carryMask = iota, 1 << iota
	subtractShift, subtractMask
	parityOverflowShift, parityOverflowMask
	undoc3Shift, undoc3Mask
	halfCarryShift, halfCarryMask
	undoc5Shift, undoc5Mask
	zeroShift, zeroMask
	signShift, signMask

	// These two bits are undocumented but we handle them properly just in case.
	undocMasks = undoc3Mask | undoc5Mask
)

// Look-up table to make setting the sign, carry, and overflow flags quickly.
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

// Look-up table to make setting the half-carry flag quickly.
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

// Look-up table to make setting the sign, carry, and overflow flags quickly when
// doing subtractions.
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

// Look-up table to make setting the half-carry flag quickly when doing subtractions.
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

// Specifies parity of each value of an 8-bit byte. 1 = even parity, 0 = odd parity.
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

// Set the undoc flags from a byte. The bits are just copied from
// the corresponding bits of the specified byte.
func (f *flags) setUndoc(v byte) {
	*f = (*f &^ undocMasks) | (flags(v) & undocMasks)
}

// Update simple flags (S, Z, P, and undoc) based on result of operation.
// Carry, half carry, and subtract are unaffected.
func (f *flags) updateFromByte(value byte) {
	f.setS(value&0x80 != 0)
	f.setZ(value == 0)
	f.setPv(parityTable[value] == 1)
	f.setUndoc(value)
}

// Update flags after value1 + value2 was placed in result.
func (f *flags) updateFromAddByte(value1, value2, result byte) {
	index := (value1&0x88)>>1 | (value2&0x88)>>2 | (result&0x88)>>3
	*f = halfCarryTable[index&7] |
		signCarryOverflowTable[index>>4] |
		flags(result&undocMasks)

	if result == 0 {
		*f |= zeroMask
	}
}

// Update flags after value1 + value2 was placed in result.
func (f *flags) updateFromAddWord(value1, value2, result word) {
	index := (value1&0x8800)>>9 | (value2&0x8800)>>10 | (result&0x8800)>>11
	*f = halfCarryTable[index&7] |
		(signCarryOverflowTable[index>>4] & carryMask) |
		(*f & (zeroMask | parityOverflowMask | signMask)) |
		flags(result.h()&undocMasks)
}

// Update flags after value1 + value2 + carry was placed in result.
func (f *flags) updateFromAdcWord(value1, value2, result word) {
	index := (value1&0x8800)>>9 | (value2&0x8800)>>10 | (result&0x8800)>>11
	*f = halfCarryTable[index&7] |
		signCarryOverflowTable[index>>4] |
		flags(result.h()&undocMasks)

	if result == 0 {
		*f |= zeroMask
	}
}

// Update flags after value1 - value2 was placed in result.
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

// Update flags after value1 - value2 - carry was placed in result.
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

// Update from a logical operation with result as a result. Specify
// true for isAnd if this was an AND operation, false if it was
// an OR or XOR.
func (f *flags) updateFromLogicByte(result byte, isAnd bool) {
	if isAnd {
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

	f.setUndoc(result)
}

// Update from a decrement operation that results in result.
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

	f.setUndoc(result)
}

// Update from an increment operation that results in result.
func (f *flags) updateFromIncByte(result byte) {
	*f &= carryMask
	f.setPv(result == 0x80)
	f.setH(result&0x0F == 0)
	f.setZ(result == 0)
	f.setS(result&0x80 != 0)
	f.setUndoc(result)
}

// Update from an in-port operation that results in result.
func (f *flags) updateFromInByte(result byte) {
	*f &^= signMask | zeroMask | halfCarryMask | parityOverflowMask | subtractMask

	if result&0x80 != 0 {
		*f |= signMask
	}
	if result == 0 {
		*f |= zeroMask
	}
	*f |= flags(parityTable[result]) << parityOverflowShift
}
