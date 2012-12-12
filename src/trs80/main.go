package main

import (
	"fmt"
	"io/ioutil"
	"time"
)

func main() {
	cmdCh := startComputer()
	serveWebsite(cmdCh)
}

func startComputer() chan<- interface{} {
	// Allocate memory.
	memorySize := 1024 * 64
	memory := make([]byte, memorySize)
	fmt.Printf("Memory has %d bytes.\n", len(memory))

	// Load ROM into memory.
	romFilename := "roms/model3.rom"
	rom, err := ioutil.ReadFile(romFilename)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ROM has %d bytes.\n", len(rom))

	// Copy ROM into memory.
	copy(memory, rom)

	// Make a CPU.
	cpu := &cpu{
		memory:   memory,
		romSize:  word(len(rom)),
		root:     &instruction{},
		updateCh: make(chan cpuUpdate, 128),
	}
	cpu.root.loadInstructions(instructionList)

	// Make it go.
	fmt.Println("Booting")
	go func() {
		if (true) {
			time.Sleep(3 * time.Second)
		}
		cpu.run()
	}()

	// Pull out updates.
	cmdCh := make(chan interface{})
	go dispatchUpdates(cpu.updateCh, cmdCh)

	return cmdCh
}
