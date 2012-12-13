package main

import (
	"fmt"
	"io/ioutil"
)

func main() {
	updateCmdCh, cpuCmdCh := startComputer()
	serveWebsite(updateCmdCh, cpuCmdCh)
}

func startComputer() (chan<- interface{}, chan<- cpuCommand) {
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

	// Various channels to communicate with the CPU.
	cpuCmdCh := make(chan cpuCommand)
	timerCh := getTimerCh()
	cpuUpdateCh := make(chan cpuUpdate)

	// Make a CPU.
	cpu := &cpu{
		memory:   memory,
		romSize:  word(len(rom)),
		root:     &instruction{},
		updateCh: cpuUpdateCh,
		nmiMask:  resetNmiBit,
		modeImage: 0x80,
	}
	cpu.root.loadInstructions(instructionList)

	// Make it go.
	fmt.Println("Booting")
	go cpu.run(cpuCmdCh, timerCh)

	// Pull out updates.
	updateCmdCh := make(chan interface{})
	go dispatchUpdates(cpu.updateCh, updateCmdCh)

	return updateCmdCh, cpuCmdCh
}
