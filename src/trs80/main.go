package main

import (
	"io/ioutil"
	"runtime/pprof"
	"os"
	"log"
)

var profileFilename = "trs80.prof"

func main() {
	if true {
		cpu := createComputer(nil)

		f, err := os.Create(profileFilename)
		if err != nil {
			log.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()

		for cpu.clock < cpuHz*50 {
			cpu.step()
		}
	} else {
		serveWebsite()
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
		nmiMask:     resetNmiBit,
		modeImage:   0x80,
	}
	cpu.root.loadInstructions(instructionList)

	return cpu
}
