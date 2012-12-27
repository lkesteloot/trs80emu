// Copyright 2012 Lawrence Kesteloot

package main

import (
	"flag"
	"io/ioutil"
	"log"
	"time"
)

const (
	historicalPcCount = 20
)

type vm struct {
	// The CPU state.
	cpu cpu

	// RAM.
	memory []byte

	// Whether each byte of RAM has been initialized.
	memInit []bool

	// Size of ROM.
	romSize word

	// Simulated keyboard.
	keyboard keyboard

	// Floppy disk controller.
	fdc fdc

	// Breakpoints.
	breakpoints breakpoints

	// Queued up events.
	events events

	// Clock from boot, in cycles.
	clock uint64

	// Various I/O settings.
	modeImage byte

	// Channel to get updates from.
	vmUpdateCh chan<- vmUpdate

	// Keep last "historicalPcCount" PCs for debugging.
	historicalPc [historicalPcCount]word
	// Points to the most recent instruction added.
	historicalPcPtr int

	// Debug message.
	msg string

	previousDumpTime    time.Time
	previousDumpClock   uint64
	sleptSinceDump      time.Duration
	startTime           int64
	previousAdjustClock uint64
	previousTimerClock  uint64
}

// Command to the VM from the UI.
type vmCommand struct {
	Cmd  string
	Addr int
	Data string
}

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
		memory:     memory,
		memInit:    memInit,
		romSize:    word(len(rom)),
		vmUpdateCh: vmUpdateCh,
		modeImage:  0x80,
	}
	vm.cpu.initialize()

	// Specify the disks in the drive.
	for drive, filename := range flag.Args() {
		err = vm.loadDisk(drive, filename)
		if err != nil {
			panic(err)
		}
	}

	return vm
}

func (vm *vm) run(vmCommandCh <-chan vmCommand) {
	running := false
	shutdown := false

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
			vm.breakpoints.add(breakpoint{pc: word(msg.Addr), active: true})
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
		default:
			panic("Unknown CPU command " + msg.Cmd)
		}
	}

	for !shutdown {
		if running {
			select {
			case msg := <-vmCommandCh:
				handleCmd(msg)
			default:
				// See if there's a breakpoint here.
				bp := vm.breakpoints.find(vm.cpu.pc)
				if bp != nil {
					if vm.vmUpdateCh != nil {
						vm.vmUpdateCh <- vmUpdate{Cmd: "breakpoint", Addr: int(vm.cpu.pc)}
					}
					log.Printf("Breakpoint at %04X", vm.cpu.pc)
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

func (vm *vm) reset(powerOn bool) {
	/// trs_cassette_reset()
	/// trs_timer_speed(0)
	vm.diskInit(powerOn)
	/// trs_hard_out(TRS_HARD_CONTROL, TRS_HARD_SOFTWARE_RESET|TRS_HARD_DEVICE_ENABLE)
	vm.cpu.setIrqMask(0)
	vm.cpu.setNmiMask(0)
	vm.keyboard.clearKeyboard()
	vm.cpu.timerInterrupt(false)

	if powerOn {
		vm.cpu.reset()
		vm.startTime = time.Now().UnixNano()
	} else {
		vm.cpu.resetButtonInterrupt(true)
	}
}
