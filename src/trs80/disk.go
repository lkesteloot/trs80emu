package main

import (
	"log"
	"io/ioutil"
)

// Data about the disk controller.
type fdc struct {
	disk disk
}

// Data about the floppy that has been inserted.
type disk struct {
	data []byte
}

func (cpu *cpu) loadDisk(filename string) error {
	return cpu.fdc.disk.load(filename)
}

func (disk *disk) load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	log.Printf("Loaded disk \"%s\" (%d bytes)", filename, len(data))
	disk.data = data

	return nil
}

func (cpu *cpu) readDiskCommand() byte {
	return 0xFF
}

func (cpu *cpu) readDiskTrack() byte {
	return 0xFF
}

func (cpu *cpu) readDiskSector() byte {
	return 0xFF
}

func (cpu *cpu) readDiskData() byte {
	return 0xFF
}

func (cpu *cpu) writeDiskCommand(value byte) {
}

func (cpu *cpu) writeDiskTrack(value byte) {
}

func (cpu *cpu) writeDiskSector(value byte) {
}

func (cpu *cpu) writeDiskData(value byte) {
}

func (cpu *cpu) writeDiskSelect(value byte) {
}
