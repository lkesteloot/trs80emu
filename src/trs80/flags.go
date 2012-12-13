package main

import (
	"fmt"
)

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
	panic("Not setting n flag yet")
	return (f & subtractMask) != 0
}

// Get parity/overflow flag.
func (f flags) pv() bool {
	panic("Not setting pv flag yet")
	return (f & parityOverflowMask) != 0
}

// Get half carry flag.
func (f flags) h() bool {
	panic("Not setting h flag yet")
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

// Update all flags based on result of operation. The "hints" string is
// the fourth column from the z80.txt files.
func (f *flags) updateFromByte(value byte, hints string) {
	// C, set if carry.
	switch hints[0] {
	case '-':
		// Nothing.
	case '+':
		// XXX.
	case '0':
		f.setC(false)
	case '1':
		f.setC(true)
	default:
		panic(fmt.Sprintf("Can't handle flag hint %c", hints[0]))
	}

	// N, ???
	switch hints[1] {
	case '-':
		// Nothing.
	case '+':
		// XXX.
	case '0':
		f.setN(false)
	case '1':
		f.setN(true)
	default:
		panic(fmt.Sprintf("Can't handle flag hint %c", hints[1]))
	}

	// P, set if parity is even. V, set if overflow.
	switch hints[2] {
	case '-':
		// Nothing.
	case 'P':
	case 'V':
		// XXX need more data.
	case '0':
		f.setPv(false)
	case '1':
		f.setPv(true)
	default:
		panic(fmt.Sprintf("Can't handle flag hint %c", hints[2]))
	}

	// H, XXX
	switch hints[3] {
	case '-':
		// Nothing.
	case '+':
		// XXX
	case '0':
		f.setH(false)
	case '1':
		f.setH(true)
	default:
		panic(fmt.Sprintf("Can't handle flag hint %c", hints[3]))
	}

	// Z, set if zero.
	switch hints[4] {
	case '-':
		// Nothing.
	case '+':
		f.setZ(value == 0)
	case '0':
		f.setZ(false)
	case '1':
		f.setZ(true)
	default:
		panic(fmt.Sprintf("Can't handle flag hint %c", hints[4]))
	}

	// S, set if negative.
	switch hints[5] {
	case '-':
		// Nothing.
	case '+':
		f.setS(value&0x80 != 0)
	case '0':
		f.setS(false)
	case '1':
		f.setS(true)
	default:
		panic(fmt.Sprintf("Can't handle flag hint %c", hints[5]))
	}
}

// Update all flags based on result of operation. The "hints" string is
// the fourth column from the z80.txt files.
func (f *flags) updateFromWord(value word, hints string) {
	// Z
	switch hints[4] {
	case '+':
		f.setZ(value == 0)
	case '0':
		f.setZ(false)
	case '1':
		f.setZ(true)
	}

	// S, set if negative.
	switch hints[5] {
	case '+':
		f.setS(value&0x8000 != 0)
	case '0':
		f.setS(false)
	case '1':
		f.setS(true)
	}
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
	if result & 0x0F == 0x0F {
		*f |= halfCarryMask
	}
	if result == 0 {
		*f |= zeroMask
	}
	if result & 0x80 != 0 {
		*f |= signMask
	}

    *f |= flags(result & undocMasks)
}

func (f *flags) updateFromIncByte(result byte) {
	*f &= carryMask

	if result == 0x80 {
		*f |= parityOverflowMask
	}
	if result & 0x0F == 0 {
		*f |= halfCarryMask
	}
	if result == 0 {
		*f |= zeroMask
	}
	if result & 0x80 != 0 {
		*f |= signMask
	}

    *f |= flags(result & undocMasks)
}

func (f *flags) updateFromInByte(result byte) {
	*f &^= signMask | zeroMask | halfCarryMask | parityOverflowMask | subtractMask

	if result & 0x80 != 0 {
		*f |= signMask
	}
	if result == 0 {
		*f |= zeroMask
	}
	*f |= flags(parityTable[result] << parityOverflowShift)
}
