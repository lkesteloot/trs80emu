package main

import (
	"fmt"
	"regexp"
)

// Look for N and NN on word boundaries.
var nRegExp = regexp.MustCompile(`\bN\b`)
var nnRegExp = regexp.MustCompile(`\bNN\b`)

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

	// Substitute N and NN.
	asm := inst.asm
	asm = nRegExp.ReplaceAllLiteralString(asm, fmt.Sprintf("%02X", byteData))
	asm = nnRegExp.ReplaceAllLiteralString(asm, fmt.Sprintf("%04X", wordData))

	line += asm
	return
}
