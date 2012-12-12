package main

import (
	"fmt"
)

// http://www.zilog.com/docs/z80/um0080.pdf page 76
type flags byte

// Whether the nybble (0-15) has even parity.
var nybbleParity []bool = []bool{
	true,  // 0000
	false, // 0001
	false, // 0010
	true,  // 0011

	false, // 0100
	true,  // 0101
	true,  // 0110
	false, // 0111

	false, // 1000
	true,  // 1001
	true,  // 1010
	false, // 1011

	true,  // 1100
	false, // 1101
	false, // 1110
	true,  // 1111
}

// Get carry flag.
func (f flags) c() bool {
	return (f & 0x01) != 0
}

// Get add/subtract flag.
func (f flags) n() bool {
	panic("Not setting n flag yet")
	return (f & 0x02) != 0
}

// Get parity/overflow flag.
func (f flags) pv() bool {
	panic("Not setting pv flag yet")
	return (f & 0x04) != 0
}

// Get half carry flag.
func (f flags) h() bool {
	panic("Not setting h flag yet")
	return (f & 0x10) != 0
}

// Get zero flag.
func (f flags) z() bool {
	return (f & 0x40) != 0
}

// Get sign flag.
func (f flags) s() bool {
	return (f & 0x80) != 0
}

// Set carry flag.
func (f *flags) setC(c bool) {
	if c {
		*f |= 0x01
	} else {
		*f &^= 0x01
	}
}

// Set add/subtract flag.
func (f *flags) setN(n bool) {
	if n {
		*f |= 0x02
	} else {
		*f &^= 0x02
	}
}

// Set parity/overflow flag.
func (f *flags) setPv(pv bool) {
	if pv {
		*f |= 0x04
	} else {
		*f &^= 0x04
	}
}

// Set half carry flag.
func (f *flags) setH(h bool) {
	if h {
		*f |= 0x10
	} else {
		*f &^= 0x10
	}
}

// Set zero flag.
func (f *flags) setZ(z bool) {
	if z {
		*f |= 0x40
	} else {
		*f &^= 0x40
	}
}

// Set sign flag.
func (f *flags) setS(s bool) {
	if s {
		*f |= 0x80
	} else {
		*f &^= 0x80
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

	// P, set if parity is even; V, set if overflow.
	switch hints[2] {
	case '-':
		// Nothing.
	case 'P':
		f.setPv(isEvenParity(value))
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

func isEvenParity(b byte) bool {
	// Byte is even parity if both nybbles have same parity.
	return nybbleParity[b&0xF] == nybbleParity[b>>4]
}
