// Copyright 2012 Lawrence Kesteloot

package main

// Memory simulator. This includes ROM, RAM, and memory-mapped I/O.

import (
	"fmt"
	"log"
)

const (
	// True RAM begins at this address.
	ramBegin        = 0x4000

	crashOnRomWrite = false
)

// Write a byte to an address in memory.
func (vm *vm) writeMem(addr word, b byte) {
	// xtrs:trs_memory.c
	// Check ROM writing. Harmless in real life, but may indicate a bug here.
	if addr < vm.romSize {
		// ROM.
		msg := fmt.Sprintf("Warning: Tried to write %02X to ROM at %04X", b, addr)
		vm.logHistoricalPc()
		if crashOnRomWrite {
			panic(msg)
		} else {
			log.Print(msg)
		}
	} else if addr >= ramBegin {
		// RAM.
		vm.memory[addr] = b
		vm.memInit[addr] = true
	} else if addr >= screenBegin && addr < screenEnd {
		// Screen.
		vm.memory[addr] = b
		if vm.vmUpdateCh != nil {
			vm.vmUpdateCh <- vmUpdate{Cmd: "poke", Addr: int(addr), Data: int(b)}
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
	if addr >= ramBegin {
		// RAM.
		if warnUninitMemRead && !vm.memInit[addr] {
			log.Printf("Warning: Uninitialized read of RAM at %04X", addr)
		}
		b = vm.memory[addr]
	} else if addr == 0x37E8 {
		// Printer. 0x30 = Printer selected, ready, with paper, not busy.
		b = 0x30
	} else if addr < vm.romSize {
		// ROM.
		b = vm.memory[addr]
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
func (vm *vm) getByteValue(ref string, byteData byte, wordData word) byte {
	cpu := &vm.cpu

	switch ref {
	case "A":
		return cpu.a
	case "B":
		return cpu.bc.h()
	case "C":
		return cpu.bc.l()
	case "D":
		return cpu.de.h()
	case "E":
		return cpu.de.l()
	case "H":
		return cpu.hl.h()
	case "L":
		return cpu.hl.l()
	case "(BC)":
		if printDebug {
			vm.msg += fmt.Sprintf("(BC = %04X) ", cpu.bc)
		}
		return vm.readMem(cpu.bc)
	case "(DE)":
		if printDebug {
			vm.msg += fmt.Sprintf("(DE = %04X) ", cpu.de)
		}
		return vm.readMem(cpu.de)
	case "(HL)":
		if printDebug {
			vm.msg += fmt.Sprintf("(HL = %04X) ", cpu.hl)
		}
		return vm.readMem(cpu.hl)
	case "(IX+N)":
		addr := cpu.ix + signExtend(byteData)
		if printDebug {
			vm.msg += fmt.Sprintf("(IX = %04X + %02X = %04X) ", cpu.ix, byteData, addr)
		}
		return vm.readMem(addr)
	case "(IY+N)":
		addr := cpu.iy + signExtend(byteData)
		if printDebug {
			vm.msg += fmt.Sprintf("(IY = %04X + %02X = %04X) ", cpu.iy, byteData, addr)
		}
		return vm.readMem(addr)
	case "N":
		return byteData
	case "(NN)":
		if printDebug {
			vm.msg += fmt.Sprintf("(NN = %04X) ", wordData)
		}
		return vm.readMem(wordData)
	}

	panic("We don't yet handle addressing mode " + ref)
}

// Get a word from the specified reference, which could be a register or memory location.
func (vm *vm) getWordValue(ref string, byteData byte, wordData word) word {
	cpu := &vm.cpu

	switch ref {
	case "AF":
		var w word
		w.setH(cpu.a)
		w.setL(byte(cpu.f))
		return w
	case "AF'":
		var w word
		w.setH(cpu.ap)
		w.setL(byte(cpu.fp))
		return w
	case "BC":
		return cpu.bc
	case "DE":
		return cpu.de
	case "HL":
		return cpu.hl
	case "IX":
		return cpu.ix
	case "IY":
		return cpu.iy
	case "SP":
		return cpu.sp
	case "NN":
		return wordData
	case "(NN)":
		if printDebug {
			vm.msg += fmt.Sprintf("(NN = %04X) ", wordData)
		}
		return vm.readMemWord(wordData)
	case "(HL)":
		if printDebug {
			vm.msg += fmt.Sprintf("(HL = %04X) ", cpu.hl)
		}
		return vm.readMemWord(cpu.hl)
	case "(SP)":
		if printDebug {
			vm.msg += fmt.Sprintf("(SP = %04X) ", cpu.sp)
		}
		return vm.readMemWord(cpu.sp)
	}

	panic("We don't yet handle addressing mode " + ref)
}

// Write a byte to the specified reference, which could be a register or memory location.
func (vm *vm) setByte(ref string, value byte, byteData byte, wordData word) {
	cpu := &vm.cpu

	switch ref {
	case "A":
		cpu.a = value
	case "B":
		cpu.bc.setH(value)
	case "C":
		cpu.bc.setL(value)
	case "D":
		cpu.de.setH(value)
	case "E":
		cpu.de.setL(value)
	case "H":
		cpu.hl.setH(value)
	case "L":
		cpu.hl.setL(value)
	case "LX":
		cpu.ix.setL(value)
	case "HX":
		cpu.ix.setH(value)
	case "LY":
		cpu.iy.setL(value)
	case "HY":
		cpu.iy.setH(value)
	case "(BC)":
		vm.writeMem(cpu.bc, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(BC = %04X) ", cpu.bc)
		}
	case "(DE)":
		vm.writeMem(cpu.de, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(DE = %04X) ", cpu.de)
		}
	case "(HL)":
		vm.writeMem(cpu.hl, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(HL = %04X) ", cpu.hl)
		}
	case "(IX+N)":
		addr := cpu.ix + signExtend(byteData)
		vm.writeMem(addr, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(IX = %04X + %02X = %04X) ", cpu.ix, byteData, addr)
		}
	case "(IY+N)":
		addr := cpu.iy + signExtend(byteData)
		vm.writeMem(addr, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(IY = %04X + %02X = %04X) ", cpu.iy, byteData, addr)
		}
	case "(NN)":
		vm.writeMem(wordData, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(NN = %04X) ", wordData)
		}
	default:
		panic("Can't handle destination of " + ref)
	}
}

// Write a word to the specified reference, which could be a register or memory location.
func (vm *vm) setWord(ref string, value word, byteData byte, wordData word) {
	cpu := &vm.cpu

	switch ref {
	case "AF":
		cpu.a = value.h()
		cpu.f = flags(value.l())
	case "AF'":
		cpu.ap = value.h()
		cpu.fp = flags(value.l())
	case "BC":
		cpu.bc = value
	case "DE":
		cpu.de = value
	case "HL":
		cpu.hl = value
	case "SP":
		cpu.sp = value
	case "IX":
		cpu.ix = value
	case "IY":
		cpu.iy = value
	case "(NN)":
		vm.writeMemWord(wordData, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(NN = %04X) ", wordData)
		}
	case "(SP)":
		vm.writeMemWord(cpu.sp, value)
		if printDebug {
			vm.msg += fmt.Sprintf("(SP = %04X) ", cpu.sp)
		}
	default:
		panic("Can't handle destination of " + ref)
	}
}
