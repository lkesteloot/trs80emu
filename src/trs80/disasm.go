// Copyright 2012 Lawrence Kesteloot

package main

import (
	"fmt"
	"regexp"
)

// Look for N and NN on word boundaries.
var nRegExp = regexp.MustCompile(`\bN\b`)
var nnRegExp = regexp.MustCompile(`\bNN\b`)

func (vm *vm) disasm(pc word) (line string, nextPc word) {
	instPc := pc
	inst, byteData, wordData := vm.lookUpInst(&pc)
	nextPc = pc

	line = fmt.Sprintf("%04X ", instPc)
	for pc = instPc; pc < instPc+4; pc++ {
		if pc < nextPc {
			line += fmt.Sprintf("%02X ", vm.memory[pc])
		} else {
			line += fmt.Sprint("   ")
		}
	}

	if inst == nil {
		line += "Unknown instruction"
	} else {
		// Substitute N and NN.
		line += substituteData(inst.asm, byteData, wordData)
	}
	return
}

// Fills the N and NN parts of assembly instructions with their real value.
func substituteData(asm string, byteData byte, wordData word) string {
	// This does the wrong thing when the instruction has two byte N parameters.
	// See instLd for more info.
	asm = nRegExp.ReplaceAllLiteralString(asm, fmt.Sprintf("%02X", byteData))
	asm = nnRegExp.ReplaceAllLiteralString(asm, fmt.Sprintf("%04X", wordData))

	return asm
}
