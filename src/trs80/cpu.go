package main

import (
	"fmt"
)

type cpu struct {
	memory []byte
	romSize uint16

	// 8-bit registers:
	a, f, b, c, d, e, h, l, i, r byte

	// "prime" registers:
	ap, fp, bp, cp, dp, ep, hp, lp byte

	// 16-bit registers:
	sp, pc, ix, iy uint16

	// Not sure.
	iff byte

	imap instructionMap
}

func (cpu *cpu) run() {
	for {
		cpu.step2()
	}
}

func (cpu *cpu) step() {
	beginPc := cpu.pc
	endPc := beginPc
	opcode := cpu.fetchByte()

	switch opcode {
	case 0x00:
		cpu.log(beginPc, endPc, "NOP")
	case 0x01:
		word := cpu.fetchWord()
		cpu.log(beginPc, endPc, "LD BC,%04X", word)
		cpu.setBc(word)
	case 0x11:
		word := cpu.fetchWord()
		cpu.log(beginPc, endPc, "LD DE,%04X", word)
		cpu.setDe(word)
	case 0x20:
		index := cpu.fetchByte()
		cpu.log(beginPc, endPc, "JR NZ,%02X", index)
		/*
		if cpu.z == 0 {
			cpu.pc += index
		}
		*/
	case 0x21:
		word := cpu.fetchWord()
		cpu.log(beginPc, endPc, "LD HL,%04X", word)
		cpu.setHl(word)
	case 0x31:
		word := cpu.fetchWord()
		cpu.log(beginPc, endPc, "LD SP,%04X", word)
		cpu.sp = word
	case 0x3D:
		cpu.log(beginPc, endPc, "DEC A")
		cpu.a--
	case 0xF3:
		cpu.log(beginPc, endPc, "DI")
		cpu.iff = 0
	case 0xAF:
		cpu.log(beginPc, endPc, "XOR A")
		cpu.a = cpu.a ^ cpu.a
	case 0xC3:
		addr := cpu.fetchWord()
		cpu.log(beginPc, endPc, "JP %04X", addr)
		cpu.pc = addr
	case 0xD3:
		port := cpu.fetchByte()
		cpu.log(beginPc, endPc, "OUT (%02X),A", port)
		// XXX port
	case 0xED:
		opcode16 := uint16(opcode) << 8 | uint16(cpu.fetchByte())
		switch opcode16 {
		case 0xEDB0:
			cpu.log(beginPc, endPc, "LDIR (copy HL to DE for BC bytes)")
			// Not sure if this should be while or do while.
			for cpu.bc() != 0xFFFF {
				cpu.writeMem(cpu.de(), cpu.memory[cpu.hl()])
				cpu.setHl(cpu.hl() + 1)
				cpu.setDe(cpu.de() + 1)
				cpu.setBc(cpu.bc() - 1)
			}
		default:
			panic(fmt.Sprintf("Don't know how to handle opcode %04X at %04X", opcode16, beginPc))
		}
	default:
		panic(fmt.Sprintf("Don't know how to handle opcode %02X at %04X", opcode, beginPc))
	}
}

func (cpu *cpu) fetchByte() byte {
	value := cpu.memory[cpu.pc]
	cpu.pc++
	return value
}

func (cpu *cpu) fetchWord() uint16 {
	// Little endian.
	value := uint16(cpu.memory[cpu.pc]) + 256*uint16(cpu.memory[cpu.pc + 1])
	cpu.pc += 2
	return value
}

func (cpu *cpu) log(beginPc, endPc uint16, instFormat string, a ...interface{}) {
	fmt.Printf("%04X ", beginPc)
	for pc := beginPc; pc < endPc; pc++ {
		fmt.Printf("%02X ", cpu.memory[pc])
	}
	for pc := endPc; pc < beginPc + 4; pc++ {
		fmt.Print("   ")
	}
	fmt.Printf(instFormat, a...)
	fmt.Println()
}

func (cpu *cpu) writeMem(addr uint16, b byte) {
	if addr < cpu.romSize {
		panic(fmt.Sprintf("Tried to write %02X to ROM at %04X", b, addr))
	}

	cpu.memory[addr] = b
}

func (cpu *cpu) bc() uint16 {
	return uint16(cpu.b) << 8 | uint16(cpu.c)
}

func (cpu *cpu) setBc(word uint16) {
	cpu.b = byte(word >> 8)
	cpu.c = byte(word)
}

func (cpu *cpu) de() uint16 {
	return uint16(cpu.d) << 8 | uint16(cpu.e)
}

func (cpu *cpu) setDe(word uint16) {
	cpu.d = byte(word >> 8)
	cpu.e = byte(word)
}

func (cpu *cpu) hl() uint16 {
	return uint16(cpu.h) << 8 | uint16(cpu.l)
}

func (cpu *cpu) setHl(word uint16) {
	cpu.h = byte(word >> 8)
	cpu.l = byte(word)
}
