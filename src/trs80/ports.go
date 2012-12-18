package main

import (
	"fmt"
	"log"
)

// http://www.trs-80.com/trs80-zaps-internals.htm#portsm3
// http://www.trs-80.com/trs80-zaps-internals.htm#ports
var ports = map[byte]string{
	0xE0: "maskable interrupt",
	0xE4: "NMI options/status",
	0xEC: "various controls",
	0xF0: "FDC command/status",
	0xF4: "select drive and options",
	0xFF: "cassette port",
}

func (cpu *cpu) readPort(port byte) byte {
	log.Printf("Reading port %02X", port)
	switch port {
	case 0xE0:
		// Figure out which interrupts were requested.
		return ^cpu.irqLatch
	case 0xE4:
		// NMI latch read.
		return ^cpu.nmiLatch
	case 0xEC, 0xED, 0xEE, 0xEF:
		// Acknowledge timer.
		cpu.timerInterrupt(false)
		return 0xFF
	case 0xF0:
		// Disk command.
		return readDiskCommand()
	case 0xF1:
		// Disk track.
		return readDiskTrack()
	case 0xF2:
		// Disk sector.
		return readDiskSector()
	case 0xF3:
		// Disk data.
		return readDiskData()
	case 0xFF:
		// Cassette and various flags.
		cassetteStatus := byte(0)
		return (cpu.modeImage & 0x7E) | cassetteStatus
	}

	panic(fmt.Sprintf("Can't read from unknown port %02X", port))
}

func (cpu *cpu) writePort(port byte, value byte) {
	log.Printf("Writing %02X to port %02X", value, port)
	switch port {
	case 0xE0:
		// Set interrupt mask.
		cpu.setIrqMask(value)
	case 0xE4, 0xE5, 0xE6, 0xE7:
		// NMI state.
		cpu.setNmiMask(value)
	case 0xEC, 0xED, 0xEE, 0xEF:
		// Various controls.
		cpu.modeImage = value
		/// trs_cassette_motor((value & 0x02) >> 1)
		/// trs_screen_expanded((value & 0x04) >> 2)
		/// trs_screen_alternate(!((value & 0x08) >> 3))
		/// trs_timer_speed((value & 0x40) >> 6)
	case 0xF0:
		// Disk command.
		writeDiskCommand(value)
	case 0xF1:
		// Disk track.
		writeDiskTrack(value)
	case 0xF2:
		// Disk sector.
		writeDiskSector(value)
	case 0xF3:
		// Disk data.
		writeDiskData(value)
	case 0xF4, 0xF5, 0xF6, 0xF7:
		// Disk select.
		writeDiskSelect(value)
	default:
		panic(fmt.Sprintf("Can't write %02X to unknown port %02X", value, port))
	}
}
