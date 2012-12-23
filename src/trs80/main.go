package main

import (
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"
)

const profiling = false
const profileFilename = "trs80.prof"

func main() {
	if profiling {
		profileSystem()
	} else {
		serveWebsite()
	}
}

func profileSystem() {
	cpu := createComputer(nil)

	f, err := os.Create(profileFilename)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	cpu.reset(true)
	for cpu.clock < cpuHz*50 {
		cpu.step()
	}
}

func createComputer(cpuUpdateCh chan<- cpuUpdate) *cpu {
	// Allocate memory.
	memorySize := 1024 * 64
	memory := make([]byte, memorySize)
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
	cpu := &cpu{
		memory:      memory,
		romSize:     word(len(rom)),
		root:        &instruction{},
		cpuUpdateCh: cpuUpdateCh,
		modeImage:   0x80,
	}
	cpu.root.loadInstructions(instructionList)

	/// err = cpu.loadDisk("disks/aigames1.dsk")
	// err = cpu.loadDisk("disks/tdos13a.dsk")
	// err = cpu.loadDisk("disks/lescrp3.dsk")
	// err = cpu.loadDisk("disks/ldos513.dsk")
	err = cpu.loadDisk("disks/LDOS-DOT.DSK")
	if err != nil {
		panic(err)
	}

	return cpu
}
