// Copyright 2012 Lawrence Kesteloot

package main

// The VM (Virtual Machine) represents the entire machine.

import (
	"flag"
	/// "github.com/remogatto/z80"
	"github.com/lkesteloot/z80"
	"io/ioutil"
	"log"
	"time"
)

const (
	// How many instructions to keep around in a queue so that we can display
	// the last historicalPcCount instructions when a problem happens.
	historicalPcCount = 20
)

// The vm structure (Virtual Machine) represents the entire emulated machine.
// That includes the CPU, memory, disk, cassette, keyboard, display, and other
// parts like the clock interrupt hardware.
type vm struct {
	// The CPU state.
	z80 *z80.Z80

	// All of addressable memory, including ROM, RAM, and memory-mapped I/O
	// such as the display.
	memory []byte

	// Whether each byte of RAM has been initialized. This is useful for
	// finding bugs in the emulator. If a program reads too many uninitialized
	// locations, then it has probably gone off the rails.
	memInit []bool

	// Size of ROM, starting at memory location zero.
	romSize uint16

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

	// Simulated keyboard.
	keyboard keyboard

	// Floppy disk controller.
	fdc fdc

	// Cassette controller..
	cc cassetteController

	// Breakpoints.
	breakpoints breakpoints

	// Queued up events.
	events events

	// Clock from boot, in cycles.
	clock uint64

	// Various I/O settings.
	modeImage byte

	// Channel to get updates from. The VM will send updates (screen
	// writes, diagnostic messages, etc.) to this channel.
	vmUpdateCh chan<- vmUpdate

	// Keep last "historicalPcCount" PCs for debugging.
	historicalPc [historicalPcCount]uint16
	// Points to the most recent instruction added.
	historicalPcPtr int

	// Debug message. This is constructed during instruction execution
	// and logged after the instruction is done.
	msg string

	// Various fields to periodically debug or adjust the VM.
	previousDumpTime    time.Time
	previousDumpClock   uint64
	sleptSinceDump      time.Duration
	startTime           int64
	previousAdjustClock uint64
	previousTimerClock  uint64
}

// Command to the VM from the UI, such as keyboard presses or boot.
type vmCommand struct {
	Cmd  string
	Addr int
	Data string
}

// Creates a new virtual machine. Updates will be sent to vmUpdateCh.
func createVm(vmUpdateCh chan<- vmUpdate) *vm {
	// Allocate memory.
	memorySize := 1024 * 64
	memory := make([]byte, memorySize)
	memInit := make([]bool, memorySize)
	log.Printf("Memory has %d bytes", len(memory))

	// Load ROM into memory.
	romFilename := "roms/model3.rom"
	rom, err := ioutil.ReadFile(romFilename)
	if err != nil {
		panic(err)
	}
	log.Printf("ROM has %d bytes", len(rom))

	// Copy ROM into memory.
	copy(memory, rom)

	// Make a CPU.
	vm := &vm{
		z80:        nil, // Set below.
		memory:     memory,
		memInit:    memInit,
		romSize:    uint16(len(rom)),
		vmUpdateCh: vmUpdateCh,
		modeImage:  0x80,
	}
	vm.z80 = z80.NewZ80(vm, vm)
	vm.z80.Reset()

	// Specify the disks in the drive.
	for drive, filename := range flag.Args() {
		err = vm.loadDisk(drive, filename)
		if err != nil {
			panic(err)
		}
	}

	return vm
}

// Starts a VM. This doesn't boot the machine. It needs to get the
// boot command from the command channel, specified in vmCommandCh.
// The command channel also includes keyboard updates.
func (vm *vm) run(vmCommandCh <-chan vmCommand) {
	running := false
	shutdown := false

	// Handle a command from the UI.
	handleCmd := func(msg vmCommand) {
		switch msg.Cmd {
		case "boot":
			vm.reset(true)
			running = true
		case "reset":
			vm.reset(false)
		case "shutdown":
			shutdown = true
		case "press", "release":
			vm.keyboard.keyEvent(msg.Data, msg.Cmd == "press")
		case "add_breakpoint":
			vm.breakpoints.add(breakpoint{pc: uint16(msg.Addr), active: true})
			log.Printf("Breakpoint added at %04X", msg.Addr)
		case "tron":
			printDebug = !printDebug
			if vm.vmUpdateCh != nil {
				if printDebug {
					vm.vmUpdateCh <- vmUpdate{Cmd: "message", Msg: "Trace is on"}
				} else {
					vm.vmUpdateCh <- vmUpdate{Cmd: "message", Msg: "Trace is off"}
				}
			}
		case "set_disk0":
			log.Printf("Loading diskette %s into drive %d", msg.Data, 0)
			err := vm.loadDisk(0, "disks/"+msg.Data)
			if err != nil {
				panic(err)
			}
		case "set_disk1":
			log.Printf("Loading diskette %s into drive %d", msg.Data, 1)
			err := vm.loadDisk(1, "disks/"+msg.Data)
			if err != nil {
				panic(err)
			}
		case "set_cassette":
			log.Printf("Loading cassette %s", msg.Data)
			vm.cc.filename = msg.Data
		default:
			panic("Unknown VM command " + msg.Cmd)
		}
	}

	for !shutdown {
		if running {
			select {
			case msg := <-vmCommandCh:
				handleCmd(msg)
			default:
				// See if there's a breakpoint here.
				bp := vm.breakpoints.find(vm.z80.PC())
				if bp != nil {
					if vm.vmUpdateCh != nil {
						vm.vmUpdateCh <- vmUpdate{Cmd: "breakpoint", Addr: int(vm.z80.PC())}
					}
					log.Printf("Breakpoint at %04X", vm.z80.PC())
					vm.logHistoricalPc()
					running = false
				} else {
					vm.step()
				}
			}
		} else {
			handleCmd(<-vmCommandCh)
		}
	}

	log.Print("VM shut down")

	if vm.vmUpdateCh != nil {
		vm.vmUpdateCh <- vmUpdate{Cmd: "shutdown"}
	}

	// No more updates.
	close(vm.vmUpdateCh)
	vm.vmUpdateCh = nil
}

// Log the last historicalPcCount assembly instructions that we executed.
func (vm *vm) logHistoricalPc() {
	for i := 0; i < historicalPcCount; i++ {
		pc := vm.historicalPc[(vm.historicalPcPtr+i+1)%historicalPcCount]
		line, _ := vm.disasm(pc)
		log.Print(line)
	}
}

// Reset the virtual machine, optionally to power-on state.
func (vm *vm) reset(powerOn bool) {
	vm.resetCassette()
	vm.diskInit(powerOn)
	vm.setIrqMask(0)
	vm.setNmiMask(0)
	vm.keyboard.clearKeyboard()
	vm.timerInterrupt(false)

	if powerOn {
		vm.z80.Reset()
		vm.startTime = time.Now().UnixNano()
	} else {
		vm.resetButtonInterrupt(true)
	}
}
