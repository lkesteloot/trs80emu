package main

import (
	"fmt"
)

const printDebug = false

type cpu struct {
	memory  []byte
	romSize word

	// Clock from boot, in cycles.
	clock uint64

	// Registers:
	a, i, r    byte
	f          flags
	bc, de, hl word

	// "prime" registers:
	ap            byte
	fp            flags
	bcp, dep, hlp word

	// 16-bit registers:
	sp, pc, ix, iy word

	// Interrupt flag?
	iff bool

	// Root of instruction tree.
	root *instruction

	// Channel to get updates from.
	updateCh chan cpuUpdate
}

// Command to the CPU from the UI.
type cpuCommand struct {
	Cmd string
	Data int
}

func (cpu *cpu) run(cpuCmdCh <-chan cpuCommand) {
	running := false

	handleCmd := func (msg cpuCommand) {
		switch msg.Cmd {
		case "boot":
			running = true
		case "press", "release":
			keyEvent(msg.Data, msg.Cmd == "press")
		default:
			panic("Unknown CPU command " + msg.Cmd)
		}
	}

	for {
		if (running) {
			select {
			case msg := <-cpuCmdCh:
				handleCmd(msg)
			default:
				cpu.step2()
			}
		} else {
			handleCmd(<-cpuCmdCh)
		}
	}
}

func (cpu *cpu) step() {
	/*
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
			if cpu.z == 0 {
				cpu.pc += index
			}
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
			opcode16 := word(opcode) << 8 | word(cpu.fetchByte())
			switch opcode16 {
			case 0xEDB0:
				cpu.log(beginPc, endPc, "LDIR (copy HL to DE for BC bytes)")
				// Not sure if this should be while or do while.
				for cpu.bc() != 0xFFFF {
					cpu.writeMem(cpu.de(), cpu.readMem(cpu.hl()))
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
	*/
}

func (cpu *cpu) fetchByte() byte {
	value := cpu.readMem(cpu.pc)
	cpu.pc++
	return value
}

func (cpu *cpu) fetchWord() (w word) {
	// Little endian.
	w.setL(cpu.fetchByte())
	w.setH(cpu.fetchByte())

	return
}

func (cpu *cpu) writeMem(addr word, b byte) {
	// Check ROM writing. Harmless in real life, but may indicate a bug here.
	if addr < cpu.romSize {
		panic(fmt.Sprintf("Tried to write %02X to ROM at %04X", b, addr))
	}

	// Memory-mapped I/O.
	// http://www.trs-80.com/trs80-zaps-internals.htm#memmapio
	if addr >= 0x37E0 && addr <= 0x37FF {
		panic(fmt.Sprintf("Tried to write %02X to cassette/disk at %04X", b, addr))
	} else if addr >= 0x3801 && addr <= 0x3880 {
		panic(fmt.Sprintf("Tried to write %02X to keyboard at %04X", b, addr))
	} else if addr >= 0x3C00 && addr <= 0x3FFF {
		cpu.memory[addr] = b
		cpu.updateCh <- cpuUpdate{Cmd: "poke", Addr: int(addr), Data: int(b)}
	} else {
		cpu.memory[addr] = b
	}
}

func (cpu *cpu) writeMemWord(addr word, w word) {
	// Little endian.
	cpu.writeMem(addr, w.l())
	cpu.writeMem(addr+1, w.h())
}

func (cpu *cpu) readMem(addr word) (b byte) {
	// Memory-mapped I/O.
	// http://www.trs-80.com/trs80-zaps-internals.htm#memmapio
	// xtrs:trs_memory.c
	if addr >= 0x4000 {
		// RAM.
		b = cpu.memory[addr]
	} else if addr == 0x37E8 {
		// Printer. 0x30 = Printer selected, ready, with paper, not busy.
		b = 0x30
	} else if addr < cpu.romSize {
		// ROM.
		b = cpu.memory[addr]
	} else if addr >= screenBegin && addr < screenEnd {
		// Screen.
		b = cpu.memory[addr]
	} else if addr >= keyboardBegin && addr < keyboardEnd {
		// Keyboard.
		b = cpu.readKeyboard(addr)
	} else {
		// ?? From trs_memory.c. What about all the memory-mapped I/O?
		b = 0xFF
	}

	// } else if addr >= 0x37E0 && addr <= 0x37FF {
		// b = cpu.readDisk(addr)
	// }

	return
}

func (cpu *cpu) readMemWord(addr word) (w word) {
	w.setL(cpu.readMem(addr))
	w.setH(cpu.readMem(addr + 1))

	return
}

func (cpu *cpu) pushByte(b byte) {
	cpu.sp--
	cpu.writeMem(cpu.sp, b)
}

func (cpu *cpu) pushWord(w word) {
	cpu.pushByte(w.h())
	cpu.pushByte(w.l())
}

func (cpu *cpu) popByte() byte {
	cpu.sp++
	return cpu.readMem(cpu.sp - 1)
}

func (cpu *cpu) popWord() word {
	var w word

	w.setL(cpu.popByte())
	w.setH(cpu.popByte())

	return w
}

func (cpu *cpu) log(s string) {
	if (printDebug) {
		fmt.Print(s)
	}
}

func (cpu *cpu) logf(format string, arg ...interface{}) {
	if (printDebug) {
		fmt.Printf(format, arg...)
	}
}

func (cpu *cpu) logln() {
	if (printDebug) {
		fmt.Println()
	}
}
