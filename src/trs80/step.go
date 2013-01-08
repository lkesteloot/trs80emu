// Copyright 2012 Lawrence Kesteloot

package main

// The main code that emulates the Z80.

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Steps through one instruction.
func (vm *vm) step() {
	cpu := &vm.cpu

	// Log PC for retroactive disassembly.
	if historicalPcCount > 0 {
		vm.historicalPcPtr = (vm.historicalPcPtr + 1) % historicalPcCount
		vm.historicalPc[vm.historicalPcPtr] = cpu.pc
	}

	// Look up the instruction in the tree.
	instPc := cpu.pc
	inst, byteData, wordData := vm.lookUpInst(&cpu.pc)
	if inst == nil {
		vm.logHistoricalPc()
		panic("Don't know how to handle opcode")
	}
	nextInstPc := cpu.pc
	avoidHandlingIrq := false
	isHalting := false

	// Put together a message for debugging this instruction.
	vm.msg = ""
	if printDebug {
		vm.explainLine(instPc, cpu.hl, cpu.a)
		vm.msg += fmt.Sprintf("%10d %04X ", vm.clock, instPc)
		for pc := instPc; pc < instPc+4; pc++ {
			if pc < nextInstPc {
				vm.msg += fmt.Sprintf("%02X ", vm.memory[pc])
			} else {
				vm.msg += "   "
			}
		}
		vm.msg += fmt.Sprintf("%-15s ", substituteData(inst.asm, byteData, wordData))
	}

	// Dispatch on instruction.
	subfields := inst.subfields
	switch inst.instInt {
	case instAdc:
		// Add with carry.
		if isWordOperand(subfields[0]) || isWordOperand(subfields[1]) {
			value1 := vm.getWordValue(subfields[0], byteData, wordData)
			value2 := vm.getWordValue(subfields[1], byteData, wordData)
			result := value1 + value2
			if cpu.f.c() {
				result++
			}
			vm.setWord(subfields[0], result, byteData, wordData)
			if printDebug {
				vm.msg += fmt.Sprintf("%04X + %04X + %v = %04X", value1, value2, cpu.f.c(), result)
			}
			cpu.f.updateFromAdcWord(value1, value2, result)
		} else {
			value1 := vm.getByteValue(subfields[0], byteData, wordData)
			value2 := vm.getByteValue(subfields[1], byteData, wordData)
			result := value1 + value2
			if cpu.f.c() {
				result++
			}
			vm.setByte(subfields[0], result, byteData, wordData)
			if printDebug {
				vm.msg += fmt.Sprintf("%02X + %02X + %v = %02X", value1, value2, cpu.f.c(), result)
			}
			cpu.f.updateFromAddByte(value1, value2, result)
		}
	case instAdd:
		if isWordOperand(subfields[0]) || isWordOperand(subfields[1]) {
			value1 := vm.getWordValue(subfields[0], byteData, wordData)
			value2 := vm.getWordValue(subfields[1], byteData, wordData)
			result := value1 + value2
			vm.setWord(subfields[0], result, byteData, wordData)
			if printDebug {
				vm.msg += fmt.Sprintf("%04X + %04X = %04X", value1, value2, result)
			}
			cpu.f.updateFromAddWord(value1, value2, result)
		} else {
			value1 := vm.getByteValue(subfields[0], byteData, wordData)
			value2 := vm.getByteValue(subfields[1], byteData, wordData)
			result := value1 + value2
			vm.setByte(subfields[0], result, byteData, wordData)
			if printDebug {
				vm.msg += fmt.Sprintf("%02X + %02X = %02X", value1, value2, result)
			}
			cpu.f.updateFromAddByte(value1, value2, result)
		}
	case instAnd, instXor, instOr:
		value := vm.getByteValue(subfields[0], byteData, wordData)
		before := cpu.a
		var symbol string
		switch inst.instInt {
		case instAnd:
			cpu.a &= value
			symbol = "&"
		case instXor:
			cpu.a ^= value
			symbol = "^"
		case instOr:
			cpu.a |= value
			symbol = "|"
		}
		cpu.f.updateFromLogicByte(cpu.a, inst.instInt == instAnd)
		if printDebug {
			vm.msg += fmt.Sprintf("%02X %s %02X = %02X", before, symbol, value, cpu.a)
		}
	case instBit:
		// Test bit.
		b, _ := strconv.ParseUint(subfields[0], 10, 8)
		value := vm.getByteValue(subfields[1], byteData, wordData)
		result := byte(1<<b) & value
		cpu.f = (cpu.f & carryMask) | halfCarryMask | (flags(result) & signMask)
		if result == 0 {
			cpu.f |= parityOverflowMask | zeroMask
		}
		if subfields[1] != "(HL)" {
			cpu.f.setUndoc(value)
		}
	case instCcf:
		// Complement carry.
		carry := cpu.f.c()
		cpu.f.setH(carry)
		cpu.f.setN(false)
		cpu.f.setC(!carry)
		cpu.f.setUndoc(cpu.a)
		if printDebug {
			vm.msg += fmt.Sprintf("Carry flipped from %s to %s", carry, !carry)
		}
	case instCp:
		value := vm.getByteValue(subfields[0], byteData, wordData)
		result := cpu.a - value
		cpu.f.updateFromSubByte(cpu.a, value, result)
		if printDebug {
			vm.msg += fmt.Sprintf("%02X - %02X = %02X", cpu.a, value, result)
		}
	case instCpi, instCpir, instCpd, instCpdr:
		// Look for A at (HL) for at most BC bytes or until \0.
		oldCarry := cpu.f.c()
		value := vm.readMem(cpu.hl)
		result := cpu.a - value
		switch inst.instInt {
		case instCpi, instCpir:
			cpu.hl++
		case instCpd, instCpdr:
			cpu.hl--
		default:
			panic("Logic error")
		}
		cpu.bc--
		switch inst.instInt {
		case instCpir:
		case instCpdr:
			if cpu.bc != 0 && result != 0 {
				// Start instructions again.
				cpu.pc -= 2
			}
		}
		cpu.f.updateFromSubByte(cpu.a, value, result)
		cpu.f = (cpu.f &^ undoc5Mask) |
			(((flags(result) - ((cpu.f & halfCarryMask) >> halfCarryShift)) & 2) << 4)
		cpu.f.setC(oldCarry)
		cpu.f.setPv(cpu.bc != 0)
		if result&15 == 8 && cpu.f.h() {
			cpu.f &^= undoc3Mask
		}
	case instCpl:
		// Complement A.
		a := cpu.a
		cpu.a = ^a
		cpu.f.setH(true)
		cpu.f.setN(true)
		cpu.f.setUndoc(cpu.a)
		if printDebug {
			vm.msg += fmt.Sprintf("A complemented from %02X to %02X", a, cpu.a)
		}
	case instDaa:
		// BCD add/subtract.
		a := int(cpu.a)
		f := cpu.f
		aLow := a & 0x0F
		carry := f.c()
		halfCarry := f.h()
		if f.n() {
			// Subtract.
			hd := carry || a > 0x99
			if halfCarry || aLow > 9 {
				if aLow > 5 {
					halfCarry = false
				}
				a = (a - 6) & 0xFF
			}
			if hd {
				a -= 0x160
			}
		} else {
			// Add.
			if halfCarry || aLow > 9 {
				halfCarry = aLow > 9
				a += 6
			}
			if carry || (a&0x1F0) > 0x90 {
				a += 0x60
			}
		}
		if a&0x100 != 0 {
			carry = true
		}
		cpu.a = byte(a)
		cpu.f.updateFromByte(cpu.a)
		cpu.f.setH(halfCarry)
		cpu.f.setC(carry)
	case instDec:
		if isWordOperand(subfields[0]) {
			value := vm.getWordValue(subfields[0], byteData, wordData)
			result := value - 1
			if printDebug {
				vm.msg += fmt.Sprintf("%04X - 1 = %04X", value, result)
			}
			vm.setWord(subfields[0], result, byteData, wordData)
			// Flags are not affected.
		} else {
			value := vm.getByteValue(subfields[0], byteData, wordData)
			result := value - 1
			if printDebug {
				vm.msg += fmt.Sprintf("%02X - 1 = %02X", value, result)
			}
			vm.setByte(subfields[0], result, byteData, wordData)
			cpu.f.updateFromDecByte(result)
		}
	case instDi:
		cpu.iff1 = false
		cpu.iff2 = false
	case instDjnz:
		rel := signExtend(byteData)
		cpu.bc.setH(cpu.bc.h() - 1)
		if cpu.bc.h() != 0 {
			cpu.pc += rel
			if printDebug {
				vm.msg += fmt.Sprintf("%04X (%d), b = %02X", cpu.pc, int16(rel), cpu.bc.h())
			}
		} else {
			if printDebug {
				vm.msg += "jump skipped"
			}
		}
	case instEi:
		cpu.iff1 = true
		cpu.iff2 = true
		avoidHandlingIrq = true
	case instEx:
		value1 := vm.getWordValue(subfields[0], byteData, wordData)
		value2 := vm.getWordValue(subfields[1], byteData, wordData)
		vm.setWord(subfields[0], value2, byteData, wordData)
		vm.setWord(subfields[1], value1, byteData, wordData)
		if printDebug {
			vm.msg += fmt.Sprintf("%04X <--> %04X", value1, value2)
		}
	case instExx:
		cpu.bc, cpu.bcp = cpu.bcp, cpu.bc
		cpu.de, cpu.dep = cpu.dep, cpu.de
		cpu.hl, cpu.hlp = cpu.hlp, cpu.hl
	case instHalt:
		// Wait for interrupt. The real Z80 executes NOPs internally until an
		// interrupt is triggered, but here we copy xtrs's behavior and just
		// back up to execute the HALT again. If an interrupt does come in, we
		// move the PC past the HALT (see use of isHalting below).
		cpu.pc--
		isHalting = true
	case instIm:
		// Interrupt mode.
		if subfields[0] != "1" {
			panic("We only support interrupt mode 1")
		}
	case instIn:
		var port byte
		source := subfields[len(subfields)-1]
		affectFlags := false
		switch source {
		case "(C)":
			port = cpu.bc.l()
			affectFlags = true
		case "(N)":
			port = byteData
		default:
			panic("Unknown IN source " + source)
		}
		value := vm.readPort(port)
		if len(subfields) == 2 {
			vm.setByte(subfields[0], value, byteData, wordData)
		}
		if affectFlags {
			cpu.f.updateFromInByte(value)
		}
		if printDebug {
			portDescription, ok := ports[port]
			if !ok {
				panic(fmt.Sprintf("Unknown port %02X", port))
			}
			vm.msg += fmt.Sprintf("%02X <- %02X (%s)", value, port, portDescription)
		}
	case instInc:
		if isWordOperand(subfields[0]) {
			value := vm.getWordValue(subfields[0], byteData, wordData)
			result := value + 1
			if printDebug {
				vm.msg += fmt.Sprintf("%04X + 1 = %04X", value, result)
			}
			vm.setWord(subfields[0], result, byteData, wordData)
			// Flags are not affected.
		} else {
			value := vm.getByteValue(subfields[0], byteData, wordData)
			result := value + 1
			if printDebug {
				vm.msg += fmt.Sprintf("%02X + 1 = %02X", value, result)
			}
			vm.setByte(subfields[0], result, byteData, wordData)
			cpu.f.updateFromIncByte(result)
		}
	case instIni, instInir, instInd, instIndr:
		// Input from port C, store in (HL), increment/decrement HL, and decrement B.
		// If repeating and B != 0, loop.
		value := vm.readPort(cpu.bc.l())
		vm.writeMem(cpu.hl, value)
		switch inst.instInt {
		case instIni, instInir:
			cpu.hl++
		case instInd, instIndr:
			cpu.hl--
		}
		// Decrement B.
		b := cpu.bc.h() - 1
		cpu.bc.setH(b)
		// Repeat.
		switch inst.instInt {
		case instInir, instIndr:
			if b != 0 {
				cpu.pc -= 2
			}
		}
		cpu.f.setZ(b == 0)
		cpu.f.setN(true)
	case instJp, instCall:
		addr := vm.getWordValue(subfields[len(subfields)-1], byteData, wordData)
		if len(subfields) == 1 || cpu.conditionSatisfied(subfields[0]) {
			if inst.instInt == instCall {
				vm.pushWord(cpu.pc)
			}
			cpu.pc = addr
			if printDebug {
				vm.msg += fmt.Sprintf("%04X", addr)
			}
		} else {
			if printDebug {
				vm.msg += "jump skipped"
			}
		}
	case instJr:
		if subfields[len(subfields)-1] != "N+2" {
			panic("Can only handle relative jumps to N, not " + subfields[len(subfields)-1])
		}
		// Relative jump is signed.
		rel := signExtend(byteData)
		if len(subfields) == 1 || cpu.conditionSatisfied(subfields[0]) {
			cpu.pc += rel
			if printDebug {
				vm.msg += fmt.Sprintf("%04X (%d)", cpu.pc, int16(rel))
			}
		} else {
			if printDebug {
				vm.msg += "jump skipped"
			}
		}
	case instLd:
		if isWordOperand(subfields[0]) || isWordOperand(subfields[1]) {
			value := vm.getWordValue(subfields[1], byteData, wordData)
			vm.setWord(subfields[0], value, byteData, wordData)
			if printDebug {
				vm.msg += fmt.Sprintf("%04X", value)
			}
		} else {
			// This is a horrific hack, but two instructions have two byte operands.
			// These are LD (IX+N),N and LD (IY+N),N. We store the first N into
			// byteData and the second N into the high byte of wordData when reading
			// the instructions. We have to hack this in here otherwise the getByteValue()
			// function will re-use the byte data. Note that the disassembled instruction
			// will be wrong for that.
			var value byte
			if strings.HasSuffix(inst.fields[1], "N),N") {
				value = byte(wordData >> 8)
			} else {
				value = vm.getByteValue(subfields[1], byteData, wordData)
			}
			vm.setByte(subfields[0], value, byteData, wordData)
			if printDebug {
				vm.msg += fmt.Sprintf("%02X", value)
			}
		}
	case instLdi, instLdir, instLdd, instLddr:
		// Copy (HL) to (DE), increment/decrement both, and decrement BC. If
		// repeating and BC != 0, loop.
		value := vm.readMem(cpu.hl)
		vm.writeMem(cpu.de, value)
		switch inst.instInt {
		case instLdi, instLdir:
			cpu.hl++
			cpu.de++
		case instLdd, instLddr:
			cpu.hl--
			cpu.de--
		}
		cpu.bc--
		switch inst.instInt {
		case instLdir, instLddr:
			if cpu.bc != 0 {
				cpu.pc -= 2
			}
		}

		// Carry, zero, and sign are unaffected.
		cpu.f.setPv(cpu.bc != 0)
		cpu.f.setH(false)
		cpu.f.setN(false)

		// Undoc craziness.
		undoc := flags(cpu.a + value)
		cpu.f = (cpu.f &^ undocMasks) | (undoc & undoc3Mask) | ((undoc & 2) << 3)
	case instNeg:
		value := cpu.a
		cpu.a = -value
		cpu.f.updateFromSubByte(0, value, cpu.a)
	case instNop:
		// Nothing to do!
	case instOut:
		var port byte
		value := vm.getByteValue(subfields[1], byteData, wordData)
		switch subfields[0] {
		case "(C)":
			port = cpu.bc.l()
		case "(N)":
			port = byteData
		default:
			panic("Unknown OUT destination " + subfields[0])
		}
		vm.writePort(port, value)
		if printDebug {
			portDescription, ok := ports[port]
			if !ok {
				panic(fmt.Sprintf("Unknown port %02X", port))
			}
			vm.msg += fmt.Sprintf("%02X (%s) <- %02X", port, portDescription, value)
		}
	case instOutdr, instOutir, instOutd, instOuti:
		// Send (HL) to port C, increment/decrement HL, decrement B. If
		// repeating, loop if B != 0.
		value := vm.readMem(cpu.hl)
		port := cpu.bc.l()
		vm.writePort(port, value)

		switch inst.instInt {
		case instOutd, instOutdr:
			cpu.hl--
		case instOuti, instOutir:
			cpu.hl++
		}

		// Decrement B.
		b := cpu.bc.h() - 1
		cpu.bc.setH(b)

		switch inst.instInt {
		case instOutdr, instOutir:
			if b != 0 {
				cpu.pc -= 2
			}
		}

		cpu.f.setZ(b == 0)
		cpu.f.setN(true)
	case instPop:
		value := vm.popWord()
		vm.setWord(subfields[0], value, byteData, wordData)
		if printDebug {
			vm.msg += fmt.Sprintf("%04X", value)
		}
	case instPush:
		value := vm.getWordValue(subfields[0], byteData, wordData)
		vm.pushWord(value)
		if printDebug {
			vm.msg += fmt.Sprintf("%04X", value)
		}
	case instRes:
		// Reset bit.
		b, _ := strconv.ParseUint(subfields[0], 10, 8)
		origValue := vm.getByteValue(subfields[1], byteData, wordData)
		value := origValue &^ (1 << b)
		vm.setByte(subfields[1], value, byteData, wordData)
		if printDebug {
			vm.msg += fmt.Sprintf("%02X &^ %02X = %02X", origValue, 1<<b, value)
		}
	case instRet:
		if subfields == nil || cpu.conditionSatisfied(subfields[0]) {
			cpu.pc = vm.popWord()
			if printDebug {
				vm.msg += fmt.Sprintf("%04X", cpu.pc)
			}
		} else {
			if printDebug {
				vm.msg += "return skipped"
			}
		}
	case instReti:
		// Return from IRQ.  We're supposed to signal I/O devices that we're
		// done with handling their interrupt, but I don't know how to do that,
		// the Z80 manual doesn't give specifics, and xtrs does nothing too.
		cpu.pc = vm.popWord()
		if printDebug {
			vm.msg += fmt.Sprintf("%04X", cpu.pc)
		}
	case instRetn:
		// Return from NMI.
		cpu.pc = vm.popWord()
		// Restore the IFF state.
		cpu.iff1 = cpu.iff2
		if printDebug {
			vm.msg += fmt.Sprintf("%04X", cpu.pc)
		}
	case instRl:
		// Rotate left through carry.
		value := vm.getByteValue(subfields[0], byteData, wordData)
		result := value << 1
		if cpu.f.c() {
			result |= 0x01
		}
		cpu.f.updateFromByte(result)
		cpu.f.setC(value&0x80 != 0)
		vm.setByte(subfields[0], result, byteData, wordData)
	case instRla:
		// Left rotate A through carry.
		value := cpu.a
		result := value << 1
		if cpu.f.c() {
			result |= 1
		}
		if printDebug {
			vm.msg += fmt.Sprintf("%02X << 1 (%v) = %02X", value, cpu.f.c(), result)
		}
		cpu.a = result
		cpu.f.setC(value&0x80 != 0)
		cpu.f.setH(false)
		cpu.f.setN(false)
		cpu.f.setUndoc(result)
	case instRlc:
		// Left rotate. We can't combine this with RLCA because the resulting condition
		// bits are different.
		value := vm.getByteValue(subfields[0], byteData, wordData)
		leftBit := value >> 7
		result := (value << 1) | leftBit
		vm.setByte(subfields[0], result, byteData, wordData)
		cpu.f.updateFromByte(result)
		cpu.f.setC(leftBit == 1)
		cpu.f.setH(false)
		cpu.f.setN(false)
		if printDebug {
			vm.msg += fmt.Sprintf("%02X << 1 = %02X", value, result)
		}
	case instRlca:
		// Left rotate.
		value := cpu.a
		leftBit := value >> 7
		cpu.a = (value << 1) | leftBit
		cpu.f.setH(false)
		cpu.f.setN(false)
		cpu.f.setC(leftBit == 1)
		cpu.f.setUndoc(cpu.a)
		if printDebug {
			vm.msg += fmt.Sprintf("%02X << 1 = %02X", value, cpu.a)
		}
	case instRld:
		// Left rotate decimal.
		origValue := vm.readMem(cpu.hl)

		// Left-shift old value, add lower bits of A.
		newValue := (origValue << 4) | (cpu.a & 0x0F)

		// Rotate high bits of old value into low bits of A.
		cpu.a = (cpu.a & 0xF0) | (origValue >> 4)

		cpu.f.updateFromByte(cpu.a)
		cpu.f.setN(false)
		cpu.f.setH(false)
		cpu.f.setUndoc(cpu.a)
		vm.writeMem(cpu.hl, newValue)
	case instRr:
		// Rotate right through carry.
		value := vm.getByteValue(subfields[0], byteData, wordData)
		result := value >> 1
		if cpu.f.c() {
			result |= 0x80
		}
		cpu.f.updateFromByte(result)
		cpu.f.setC(value&0x01 != 0)
		cpu.f.setN(false)
		cpu.f.setH(false)
		cpu.f.setUndoc(result)
		vm.setByte(subfields[0], result, byteData, wordData)
	case instRra:
		// Right rotate A through carry.
		value := cpu.a
		result := value >> 1
		if cpu.f.c() {
			result |= 0x80
		}
		if printDebug {
			vm.msg += fmt.Sprintf("%02X >> 1 (%v) = %02X", value, cpu.f.c(), result)
		}
		cpu.a = result
		cpu.f.setC(value&1 != 0)
		cpu.f.setH(false)
		cpu.f.setN(false)
		cpu.f.setUndoc(cpu.a)
	case instRrc:
		// Rotate right.
		value := vm.getByteValue(subfields[0], byteData, wordData)
		result := value >> 1
		if value&0x01 != 0 {
			result |= 0x80
		} else {
		}
		cpu.f.updateFromByte(result)
		cpu.f.setH(false)
		cpu.f.setN(false)
		cpu.f.setC(result&0x80 != 0)
		vm.setByte(subfields[0], result, byteData, wordData)
	case instRrca:
		// Rotate right.
		value := cpu.a
		rightBit := value & 1
		cpu.a = (value >> 1) | (rightBit << 7)
		cpu.f.setH(false)
		cpu.f.setN(false)
		cpu.f.setC(rightBit == 1)
		cpu.f.setUndoc(cpu.a)
		if printDebug {
			vm.msg += fmt.Sprintf("%02X >> 1 = %02X", value, cpu.a)
		}
	case instRrd:
		// Rotate right decimal.
		value := vm.readMem(cpu.hl)

		// Right-shift old value, add lower bits of A.
		result := (value >> 4) | ((cpu.a & 0x0F) << 4)

		// Rotate low bits of old value into low bits of A.
		cpu.a = (cpu.a & 0xF0) | (value & 0x0F)

		cpu.f.updateFromByte(cpu.a)
		cpu.f.setH(false)
		cpu.f.setN(false)
		vm.writeMem(cpu.hl, result)
	case instRst:
		addr := parseByte(subfields[0])
		vm.pushWord(cpu.pc)
		cpu.pc.setH(0)
		cpu.pc.setL(addr)
		if printDebug {
			vm.msg += fmt.Sprintf("%04X", cpu.pc)
		}
	case instScf:
		// Set carry.
		cpu.f.setH(false)
		cpu.f.setN(false)
		cpu.f.setC(true)
		cpu.f.setUndoc(cpu.a)
		if printDebug {
			vm.msg += fmt.Sprintf("Carry set")
		}
	case instSet:
		// Set bit.
		b, _ := strconv.ParseUint(subfields[0], 10, 8)
		value := vm.getByteValue(subfields[1], byteData, wordData)
		result := value | (1 << b)
		vm.setByte(subfields[1], result, byteData, wordData)
		if printDebug {
			vm.msg += fmt.Sprintf("%02X | %02X = %02X", value, 1<<b, result)
		}
	case instSbc:
		// Subtract with carry.
		if len(subfields) == 1 {
			panic("Can't handle SBC with one parameter")
		}
		if isWordOperand(subfields[0]) {
			before := vm.getWordValue(subfields[0], byteData, wordData)
			value := vm.getWordValue(subfields[1], byteData, wordData)
			result := before - value
			if cpu.f.c() {
				result--
			}
			if printDebug {
				vm.msg += fmt.Sprintf("%04X - %04X - %v = %04X", before, value, cpu.f.c(), result)
			}
			cpu.f.updateFromSbcWord(before, value, result)
			vm.setWord(subfields[0], result, byteData, wordData)
		} else {
			before := vm.getByteValue(subfields[0], byteData, wordData)
			value := vm.getByteValue(subfields[1], byteData, wordData)
			result := before - value
			if cpu.f.c() {
				result--
			}
			if printDebug {
				vm.msg += fmt.Sprintf("%02X - %02X - %v = %02X", before, value, cpu.f.c(), result)
			}
			cpu.f.updateFromSubByte(before, value, result)
			vm.setByte(subfields[0], result, byteData, wordData)
		}
	case instSla:
		// Shift left into carry.
		value := vm.getByteValue(subfields[0], byteData, wordData)
		result := value << 1
		cpu.f.updateFromByte(result)
		cpu.f.setH(false)
		cpu.f.setN(false)
		cpu.f.setC(value&0x80 != 0)
		vm.setByte(subfields[0], result, byteData, wordData)
	case instSll:
		// Shift left and increment. xtrs calls this SLIA and says that it's undocumented.
		value := vm.getByteValue(subfields[0], byteData, wordData)
		result := (value << 1) | 1
		cpu.f.updateFromByte(result)
		cpu.f.setC(value&0x80 != 0)
		cpu.f.setH(false)
		cpu.f.setN(false)
		vm.setByte(subfields[0], result, byteData, wordData)
	case instSra:
		// Shift right arithmetic.
		value := vm.getByteValue(subfields[0], byteData, wordData)
		result := byte(int8(value) >> 1)
		cpu.f.updateFromByte(result)
		cpu.f.setC(value&0x01 != 0)
		cpu.f.setH(false)
		cpu.f.setN(false)
		vm.setByte(subfields[0], result, byteData, wordData)
	case instSrl:
		// Shift right.
		value := vm.getByteValue(subfields[0], byteData, wordData)
		result := value >> 1
		cpu.f.updateFromByte(result)
		cpu.f.setC(value&0x01 != 0)
		cpu.f.setH(false)
		cpu.f.setN(false)
		vm.setByte(subfields[0], result, byteData, wordData)
	case instSub:
		// Always 8-bit, always to accumulator.
		before := cpu.a
		value := vm.getByteValue(subfields[0], byteData, wordData)
		cpu.a -= value
		if printDebug {
			vm.msg += fmt.Sprintf("%02X - %02X = %02X", before, value, cpu.a)
		}
		cpu.f.updateFromSubByte(before, value, cpu.a)
	default:
		panic(fmt.Sprintf("Don't know how to handle %s (at %04X)",
			inst.asm, instPc))
	}

	if vm.msg != "" {
		log.Print(vm.msg)
	}

	// Dispatch scheduled events.
	vm.events.dispatch(vm.clock)

	// Handle non-maskable interrupts.
	if (cpu.nmiLatch&cpu.nmiMask) != 0 && !cpu.nmiSeen {
		if isHalting {
			// Skip past HALT.
			cpu.pc++
		}
		vm.handleNmi()
		cpu.nmiSeen = true

		// Simulate the reset button being released.
		vm.cpu.resetButtonInterrupt(false)
	}

	// Handle interrupts.
	if (cpu.irqLatch&cpu.irqMask) != 0 && cpu.iff1 && !avoidHandlingIrq {
		if isHalting {
			// Skip past HALT.
			cpu.pc++
		}
		vm.handleIrq()
	}

	vm.clock += inst.cycles
	if cpu.pc != nextInstPc {
		// If we jumped, pay the penalty.
		vm.clock += inst.jumpPenalty
	}

	if vm.clock > vm.previousDumpClock+cpuHz {
		now := time.Now()
		if vm.previousDumpClock > 0 {
			elapsed := now.Sub(vm.previousDumpTime)
			computerTime := float64(vm.clock-vm.previousDumpClock) / float64(cpuHz)
			log.Printf("Computer time: %.1fs, elapsed: %.1fs, mult: %.1f, slept: %dms",
				computerTime, elapsed.Seconds(), computerTime/elapsed.Seconds(),
				vm.sleptSinceDump/time.Millisecond)
			vm.sleptSinceDump = 0
		}
		vm.previousDumpTime = now
		vm.previousDumpClock = vm.clock
	}

	// Slow down CPU if we're going too fast.
	if !*profiling && vm.clock > vm.previousAdjustClock+1000 {
		now := time.Now().UnixNano()
		elapsedReal := time.Duration(now - vm.startTime)
		elapsedFake := time.Duration(vm.clock * cpuPeriodNs)
		aheadNs := elapsedFake - elapsedReal
		if aheadNs > 0 {
			time.Sleep(aheadNs)
			vm.sleptSinceDump += aheadNs
		} else {
			// Yield periodically so that we can get messages from other
			// goroutines like the one sending us commands.
			runtime.Gosched()
		}
		vm.previousAdjustClock = vm.clock
	}

	// Set off a timer interrupt.
	if vm.clock > vm.previousTimerClock+timerCycles {
		vm.handleTimer()
		vm.previousTimerClock = vm.clock
	}

	// Update cassette state.
	vm.updateCassette()
}
