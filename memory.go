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
func (vm *vm) writeMem(addr uint16, b byte, protectRom bool) {
	// xtrs:trs_memory.c
	// Check ROM writing. Harmless in real life, but may indicate a bug here.
	if addr < vm.romSize {
		// ROM.
		if protectRom {
			if crashOnRomWrite || logOnRomWrite {
				msg := fmt.Sprintf("Warning: Tried to write %02X to ROM at %04X", b, addr)
				vm.logHistoricalPc()
				if crashOnRomWrite {
					panic(msg)
				} else {
					log.Print(msg)
				}
			}
		} else {
			vm.memory[addr] = b
			vm.memInit[addr] = true
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

// Read a byte from memory.
func (vm *vm) readMem(addr uint16) (b byte) {
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

// The rest of the file is to satisfy the z80.MemoryAccessor interface, which the
// z80 uses.
func (vm *vm) ReadByte(address uint16) byte {
	vm.clock += 3
	return vm.ReadByteInternal(address)
}

func (vm *vm) ReadByteInternal(address uint16) byte {
	return vm.readMem(address)
}

func (vm *vm) WriteByte(address uint16, value byte) {
	vm.clock += 3
	vm.WriteByteInternal(address, value)
}

func (vm *vm) WriteByteInternal(address uint16, value byte) {
	vm.writeMem(address, value, true)
}

func (vm *vm) ContendRead(address uint16, time int) {
	vm.clock += uint64(time)
}

func (vm *vm) ContendReadNoMreq(address uint16, time int) {
	vm.clock += uint64(time)
}

func (vm *vm) ContendReadNoMreq_loop(address uint16, time int, count uint) {
	vm.clock += uint64(time * int(count))
}

func (vm *vm) ContendWriteNoMreq(address uint16, time int) {
	vm.clock += uint64(time)
}

func (vm *vm) ContendWriteNoMreq_loop(address uint16, time int, count uint) {
	vm.clock += uint64(time * int(count))
}

func (vm *vm) Read(address uint16) byte {
	// Not sure.
	return vm.readMem(address)
}

func (vm *vm) Write(address uint16, value byte, protectROM bool) {
	// Not sure.
	vm.writeMem(address, value, protectROM)
}

func (vm *vm) Data() []byte {
	return vm.memory
}
