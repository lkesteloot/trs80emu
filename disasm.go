// Copyright 2012 Lawrence Kesteloot

package main

import (
	"fmt"
	/// "github.com/remogatto/z80"
	"github.com/lkesteloot/z80"
)

// Disassemble the instruction at the given pc and return the address,
// machine language, and instruction. Return the PC of the following
// instruction in nextPc.
func (vm *vm) disasm(pc uint16) (line string, nextPc uint16) {
	var asm string

	shift := 0

	// Disassemble the instruction.
	for {
		asm, nextPc, shift = z80.Disassemble(vm, pc, shift)

		// Keep going as long as shift != 0. This is for extended instructions like 0xCB.
		if shift == 0 {
			break
		}
	}

	// Address.
	line = fmt.Sprintf("%04X ", pc)

	// Machine language.
	for addr := pc; addr < pc+4; addr++ {
		if addr < nextPc {
			line += fmt.Sprintf("%02X ", vm.memory[addr])
		} else {
			line += fmt.Sprint("   ")
		}
	}

	// Instruction.
	line += asm
	return
}
