// Copyright 2012 Lawrence Kesteloot

package main

// Memory simulator. This includes ROM, RAM, and memory-mapped I/O.

import (
	"fmt"
	"log"
)

const (
	// True RAM begins at this address.
	ramBegin = 0x4000
)

// Write a byte to an address in memory.
func (vm *vm) writeMem(addr word, b byte) {
	// xtrs:trs_memory.c
	// Check ROM writing. Harmless in real life, but may indicate a bug here.
	if addr < vm.romSize {
		// ROM.
		if crashOnRomWrite || logOnRomWrite {
			msg := fmt.Sprintf("Warning: Tried to write %02X to ROM at %04X", b, addr)
			vm.logHistoricalPc()
			if crashOnRomWrite {
				panic(msg)
			} else {
				log.Print(msg)
			}
		}
	} else if addr >= ramBegin {
		// RAM.
		vm.memory[addr] = b
		vm.memInit[addr] = true
	} else if addr >= screenBegin && addr < screenEnd {
		// Screen.
		vm.memory[addr] = b
		if vm.vmUpdateCh != nil {
			vm.vmUpdateCh <- vmUpdate{Cmd: "poke", Addr: int(addr), Msg: string(b)}
		}
	} else if addr == 0x37E8 {
		// Printer. Ignore, but could print ASCII byte to display.
	} else {
		// Ignore write anywhere else.
	}
}

// Write a word to memory, little endian.
func (vm *vm) writeMemWord(addr word, w word) {
	// Little endian.
	vm.writeMem(addr, w.l())
	vm.writeMem(addr+1, w.h())
}

// Read a byte from memory.
func (vm *vm) readMem(addr word) (b byte) {
	// Memory-mapped I/O.
	// http://www.trs-80.com/trs80-zaps-internals.htm#memmapio
	// xtrs:trs_memory.c
	if addr < vm.romSize {
		// ROM.
		b = vm.memory[addr]
	} else if addr >= ramBegin {
		// RAM.
		if warnUninitMemRead && !vm.memInit[addr] {
			log.Printf("Warning: Uninitialized read of RAM at %04X", addr)
		}
		b = vm.memory[addr]
	} else if addr == 0x37E8 {
		// Printer. 0x30 = Printer selected, ready, with paper, not busy.
		b = 0x30
	} else if addr >= screenBegin && addr < screenEnd {
		// Screen.
		b = vm.memory[addr]
	} else if addr >= keyboardBegin && addr < keyboardEnd {
		// Keyboard.
		b = vm.readKeyboard(addr)
	} else {
		// Unmapped memory.
		b = 0xFF
	}

	return
}

// Read a word from memory, little endian.
func (vm *vm) readMemWord(addr word) (w word) {
	w.setL(vm.readMem(addr))
	w.setH(vm.readMem(addr + 1))

	return
}

// Push a byte onto the stack.
func (vm *vm) pushByte(b byte) {
	vm.cpu.sp--
	vm.writeMem(vm.cpu.sp, b)
}

// Push a word onto the stack, little endian.
func (vm *vm) pushWord(w word) {
	vm.pushByte(w.h())
	vm.pushByte(w.l())
}

// Pop a byte off the stack.
func (vm *vm) popByte() byte {
	vm.cpu.sp++
	return vm.readMem(vm.cpu.sp - 1)
}

// Pop a word off the stack, little endian.
func (vm *vm) popWord() word {
	var w word

	w.setL(vm.popByte())
	w.setH(vm.popByte())

	return w
}

// Get a byte from the specified reference, which could be a register or memory location.
func (vm *vm) getByteValue(operand int, byteData byte, wordData word) byte {
	cpu := &vm.cpu

	switch operand {
	case operandA:
		return cpu.a
	case operandB:
		return cpu.bc.h()
	case operandC:
		return cpu.bc.l()
	case operandD:
		return cpu.de.h()
	case operandE:
		return cpu.de.l()
	case operandH:
		return cpu.hl.h()
	case operandL:
		return cpu.hl.l()
	case operandParensBc:
		if printDebug {
			vm.msg += fmt.Sprintf("(BC = %04X) ", cpu.bc)
		}
		return vm.readMem(cpu.bc)
	case operandParensDe:
		if printDebug {
			vm.msg += fmt.Sprintf("(DE = %04X) ", cpu.de)
		}
		return vm.readMem(cpu.de)
	case operandParensHl:
		if printDebug {
			vm.msg += fmt.Sprintf("(HL = %04X) ", cpu.hl)
		}
		return vm.readMem(cpu.hl)
	case operandParensIxPlusN:
		addr := cpu.ix + signExtend(byteData)
		if printDebug {
			vm.msg += fmt.Sprintf("(IX = %04X + %02X = %04X) ", cpu.ix, byteData, addr)
		}
		return vm.readMem(addr)
	case operandParensIyPlusN:
		addr := cpu.iy + signExtend(byteData)
		if printDebug {
			vm.msg += fmt.Sprintf("(IY = %04X + %02X = %04X) ", cpu.iy, byteData, addr)
		}
		return vm.readMem(addr)
	case operandN:
		return byteData
	case operandParensNn:
		if printDebug {
			vm.msg += fmt.Sprintf("(NN = %04X) ", wordData)
		}
		return vm.readMem(wordData)
	}

	panic(fmt.Sprintf("We don't yet handle addressing mode %d", operand))
}

// Get a word from the specified reference, which could be a register or memory location.
func (vm *vm) getWordValue(operand int, byteData byte, wordData word) word {
	cpu := &vm.cpu

	switch operand {
	case operandAf:
		var w word
		w.setH(cpu.a)
		w.setL(byte(cpu.f))
		return w
	case operandAfp:
		var w word
		w.setH(cpu.ap)
		w.setL(byte(cpu.fp))
		return w
	case operandBc:
		return cpu.bc
	case operandDe:
		return cpu.de
	case operandHl:
		return cpu.hl
	case operandIx:
		return cpu.ix
	case operandIy:
		return cpu.iy
	case operandSp:
		return cpu.sp
	case operandNn:
		return wordData
	case operandParensNn:
		if printDebug {
			vm.msg += fmt.Sprintf("(NN = %04X) ", wordData)
		}
		return vm.readMemWord(wordData)
	case operandParensHl:
		if printDebug {
			vm.msg += fmt.Sprintf("(HL = %04X) ", cpu.hl)
		}
		return vm.readMemWord(cpu.hl)
	case operandParensSp:
		if printDebug {
			vm.msg += fmt.Sprintf("(SP = %04X) ", cpu.sp)
		}
		return vm.readMemWord(cpu.sp)
	}

	panic(fmt.Sprintf("We don't yet handle addressing mode %d", operand))
}

// Write a byte to the specified reference, which could be a register or memory location.
func (vm *vm) setByte(operand int, value byte, byteData byte, wordData word) {
	cpu := &vm.cpu

	switch operand {
	case operandA:
		cpu.a = value
	case operandB:
		cpu.bc.setH(value)
	case operandC:
		cpu.bc.setL(value)
	case operandD:
		cpu.de.setH(value)
	case operandE:
		cpu.de.setL(value)
	case operandH:
		cpu.hl.setH(value)
	case operandL:
		cpu.hl.setL(value)
	case operandLx:
		cpu.ix.setL(value)
	case operandHx:
		cpu.ix.setH(value)
	case operandLy:
		cpu.iy.setL(value)
	case operandHy:
		cpu.iy.setH(value)
	case operandParensBc:
		vm.writeMem(cpu.bc, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(BC = %04X) ", cpu.bc)
		}
	case operandParensDe:
		vm.writeMem(cpu.de, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(DE = %04X) ", cpu.de)
		}
	case operandParensHl:
		vm.writeMem(cpu.hl, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(HL = %04X) ", cpu.hl)
		}
	case operandParensIxPlusN:
		addr := cpu.ix + signExtend(byteData)
		vm.writeMem(addr, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(IX = %04X + %02X = %04X) ", cpu.ix, byteData, addr)
		}
	case operandParensIyPlusN:
		addr := cpu.iy + signExtend(byteData)
		vm.writeMem(addr, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(IY = %04X + %02X = %04X) ", cpu.iy, byteData, addr)
		}
	case operandParensNn:
		vm.writeMem(wordData, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(NN = %04X) ", wordData)
		}
	default:
		panic(fmt.Sprintf("Can't handle destination of %d", operand))
	}
}

// Write a word to the specified reference, which could be a register or memory location.
func (vm *vm) setWord(operand int, value word, byteData byte, wordData word) {
	cpu := &vm.cpu

	switch operand {
	case operandAf:
		cpu.a = value.h()
		cpu.f = flags(value.l())
	case operandAfp:
		cpu.ap = value.h()
		cpu.fp = flags(value.l())
	case operandBc:
		cpu.bc = value
	case operandDe:
		cpu.de = value
	case operandHl:
		cpu.hl = value
	case operandSp:
		cpu.sp = value
	case operandIx:
		cpu.ix = value
	case operandIy:
		cpu.iy = value
	case operandParensNn:
		vm.writeMemWord(wordData, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(NN = %04X) ", wordData)
		}
	case operandParensSp:
		vm.writeMemWord(cpu.sp, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(SP = %04X) ", cpu.sp)
		}
	default:
		panic(fmt.Sprintf("Can't handle destination of %d", operand))
	}
}
