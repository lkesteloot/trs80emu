package main

import (
	"fmt"
	"strings"
)

func (cpu *cpu) disasm(pc word) (line string, nextPc word) {
	instPc := pc
	inst, byteData, wordData := cpu.lookUpInst(&pc)
	nextPc = pc

	line = fmt.Sprintf("%04X ", instPc)
	for pc = instPc; pc < instPc+4; pc++ {
		if pc < nextPc {
			line += fmt.Sprintf("%02X ", cpu.memory[pc])
		} else {
			line += fmt.Sprint("   ")
		}
	}

	// Substitute N and NN. Not worth a complete tokenizing, just hack it.
	asm := inst.asm
	asm = strings.Replace(asm, "NC", "!1", -1)
	asm = strings.Replace(asm, "NZ", "!2", -1)
	asm = strings.Replace(asm, "NN", fmt.Sprintf("%04X", wordData), -1)
	asm = strings.Replace(asm, "N", fmt.Sprintf("%02X", byteData), -1)
	asm = strings.Replace(asm, "!1", "NC", -1)
	asm = strings.Replace(asm, "!2", "NZ", -1)

	line += asm
	return
}
