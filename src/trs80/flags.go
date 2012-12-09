package main

// http://www.zilog.com/docs/z80/um0080.pdf page 76
type flags byte

// Get carry flag.
func (f flags) c() bool {
	return (f & 0x01) != 0
}

// Get add/subtract flag.
func (f flags) n() bool {
	return (f & 0x02) != 0
}

// Get parity/overflow flag.
func (f flags) pv() bool {
	return (f & 0x04) != 0
}

// Get half carry flag.
func (f flags) h() bool {
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
	// Z
	switch hints[4] {
	case '+':
		f.setZ(value == 0)
	case '0':
		f.setZ(false)
	case '1':
		f.setZ(true)
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
}
