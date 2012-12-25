// Copyright 2012 Lawrence Kesteloot

package main

const (
	cpuHz       = 2027520
	cpuPeriodNs = 1000000000 / cpuHz
)

type cpu struct {
	// Registers:
	a          byte
	f          flags
	bc, de, hl word

	// "prime" registers:
	ap            byte
	fp            flags
	bcp, dep, hlp word

	// 16-bit registers:
	sp, pc, ix, iy word

	// Interrupt flags.
	iff1 bool
	iff2 bool

	// Which IRQs should be handled.
	irqMask byte

	// Which IRQs have been requested by the hardware.
	irqLatch byte

	// Which NMIs should be handled.
	nmiMask byte

	// Which NMIs have been requested by the hardware.
	nmiLatch byte

	// Whether we've seen this NMI and handled it.
	nmiSeen bool

	// Root of instruction tree.
	root *instruction
}

func (cpu *cpu) initialize() {
	cpu.root = &instruction{}
	cpu.root.loadInstructions(instructionList)
}

func (cpu *cpu) reset() {
	cpu.pc = 0
	cpu.iff1 = false
	cpu.iff2 = false
}

func (cpu *cpu) conditionSatisfied(cond string) bool {
	switch cond {
	case "C":
		return cpu.f.c()
	case "NC":
		return !cpu.f.c()
	case "Z":
		return cpu.f.z()
	case "NZ":
		return !cpu.f.z()
	case "M": // Negative (minus).
		return cpu.f.s()
	case "P": // Positive (plus).
		return !cpu.f.s()
	case "PE":
		return cpu.f.pv()
	case "PO":
		return !cpu.f.pv()
	}

	panic("Unknown condition " + cond)
}

func isWordOperand(op string) bool {
	switch op {
	case "BC", "DE", "HL", "NN", "SP", "IX", "IY":
		return true
	}

	return false
}

func signExtend(b byte) word {
	return word(int8(b))
}
