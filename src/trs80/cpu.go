package main

import (
	"fmt"
	"log"
	"time"
)

const (
	printDebug      = false
	historicalPcCount = 20
	ramBegin        = 0x4000
	cpuHz           = 2027520
	cpuPeriodNs     = 1000000000 / cpuHz
)

type cpu struct {
	memory  []byte
	romSize word

	// 8 bytes, each a bitfield of keys currently pressed.
	keyboard           [8]byte
	shiftForce         uint
	keyQueue           [16]keyActivity
	keyQueueSize       int
	keyProcessMinClock uint64

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

	// Interrupt flag.
	iff1 bool

	// Which IRQs should be handled.
	irqMask byte

	// Which IRQs have been requested by the hardware.
	irqLatch byte

	// Which NMIs should be handled.
	nmiMask byte

	// Which NMIs have been requested by the hardware.
	nmiLatch byte

	// Whether we've seen this NMI and handled it.
	nmiSeen bool

	// Various I/O settings.
	modeImage byte

	// Floppy disk controller and its disks.
	fdc fdc

	// Root of instruction tree.
	root *instruction

	// Channel to get updates from.
	cpuUpdateCh chan<- cpuUpdate

	// Queue of events to dispatch.
	events events

	// Breakpoints.
	breakpoints breakpoints

	// Keep last "historicalPcCount" PCs for debugging.
	historicalPc [historicalPcCount]word
	// Points to the most recent instruction added.
	historicalPcPtr int

	previousDumpTime    time.Time
	previousDumpClock   uint64
	sleptSinceDump      time.Duration
	previousYieldClock  uint64
	startTime           int64
	previousAdjustClock uint64
}

// Command to the CPU from the UI.
type cpuCommand struct {
	Cmd  string
	Addr int
	Data string
}

func (cpu *cpu) run(cpuCommandCh <-chan cpuCommand) {
	running := false
	shutdown := false

	timerCh := getTimerCh()

	handleCmd := func(msg cpuCommand) {
		switch msg.Cmd {
		case "boot":
			cpu.reset(true)
			running = true
		case "reset":
			cpu.reset(false)
		case "shutdown":
			shutdown = true
		case "press", "release":
			cpu.keyEvent(msg.Data, msg.Cmd == "press")
		case "add_breakpoint":
			cpu.breakpoints.add(breakpoint{pc: word(msg.Addr), active: true})
			log.Printf("Breakpoint added at %04X", msg.Addr)
		default:
			panic("Unknown CPU command " + msg.Cmd)
		}
	}

	for !shutdown {
		if running {
			select {
			case msg := <-cpuCommandCh:
				handleCmd(msg)
			case <-timerCh:
				cpu.handleTimer()
			default:
				// See if there's a breakpoint here.
				bp := cpu.breakpoints.find(cpu.pc)
				if bp != nil {
					if cpu.cpuUpdateCh != nil {
						cpu.cpuUpdateCh <- cpuUpdate{Cmd: "breakpoint", Addr: int(cpu.pc)}
					}
					log.Printf("Breakpoint at %04X", cpu.pc)
					for i := 0; i < historicalPcCount; i++ {
						pc := cpu.historicalPc[(cpu.historicalPcPtr+i+1)%historicalPcCount]
						line, _ := cpu.disasm(pc)
						log.Print(line)
					}
					running = false
				} else {
					cpu.step()
				}
			}
		} else {
			handleCmd(<-cpuCommandCh)
		}
	}

	log.Print("CPU shut down")

	// No more updates.
	close(cpu.cpuUpdateCh)
}

func (cpu *cpu) reset(powerOn bool) {
	/// trs_cassette_reset()
	/// trs_timer_speed(0)
	cpu.diskInit(powerOn)
	/// trs_hard_out(TRS_HARD_CONTROL, TRS_HARD_SOFTWARE_RESET|TRS_HARD_DEVICE_ENABLE)
	cpu.setIrqMask(0)
	cpu.setNmiMask(0)
	cpu.clearKeyboard()
	cpu.timerInterrupt(false)

	if powerOn {
		cpu.z80reset()
	} else {
		cpu.resetButtonInterrupt(true)
	}
}

func (cpu *cpu) z80reset() {
	cpu.pc = 0
	// cpu.i = 0
	cpu.iff1 = false
	// cpu.iff2 = false

	cpu.startTime = time.Now().UnixNano()
}

func (cpu *cpu) writeMem(addr word, b byte) {
	// xtrs:trs_memory.c
	// Check ROM writing. Harmless in real life, but may indicate a bug here.
	if addr < cpu.romSize {
		// ROM.
		panic(fmt.Sprintf("Tried to write %02X to ROM at %04X", b, addr))
	} else if addr >= ramBegin {
		// RAM.
		cpu.memory[addr] = b
	} else if addr >= screenBegin && addr < screenEnd {
		// Screen.
		cpu.memory[addr] = b
		if cpu.cpuUpdateCh != nil {
			cpu.cpuUpdateCh <- cpuUpdate{Cmd: "poke", Addr: int(addr), Data: int(b)}
		}
	} else if addr == 0x37E8 {
		// Printer. Ignore, but could print ASCII byte to display.
	} else {
		// Ignore write.
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
	if addr >= ramBegin {
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
		// Unmapped memory.
		b = 0xFF
	}

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
	fmt.Print(s)
}

func (cpu *cpu) logf(format string, arg ...interface{}) {
	fmt.Printf(format, arg...)
}

func (cpu *cpu) logln() {
	fmt.Println()
}
