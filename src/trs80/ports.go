// Copyright 2012 Lawrence Kesteloot

package main

import (
	"fmt"
	"log"
)

// http://www.trs-80.com/trs80-zaps-internals.htm#portsm3
// http://www.trs-80.com/trs80-zaps-internals.htm#ports
var ports = map[byte]string{
	0x1F: "Unknown",
	0x3F: "Unknown",
	0x84: "Model IV video page",
	0x85: "Model IV video page",
	0x86: "Model IV video page",
	0x87: "Model IV video page",
	0xE0: "maskable interrupt",
	0xE4: "NMI options/status",
	0xE5: "NMI options/status",
	0xE6: "NMI options/status",
	0xE7: "NMI options/status",
	0xE8: "UART modem",
	0xE9: "UART switches",
	0xEA: "UART status",
	0xEB: "UART data",
	0xEC: "various controls/timer",
	0xED: "various controls/timer",
	0xEE: "various controls/timer",
	0xEF: "various controls/timer",
	0xF0: "FDC command/status",
	0xF1: "FDC track",
	0xF2: "FDC sector",
	0xF3: "FDC data",
	0xF4: "select drive and options",
	0xF5: "select drive and options",
	0xF6: "select drive and options",
	0xF7: "select drive and options",
	0xF8: "printer status",
	0xF9: "printer status",
	0xFA: "printer status",
	0xFB: "printer status",
	0xFC: "Graphics/cassette",
	0xFD: "Graphics/cassette",
	0xFE: "Graphics/cassette",
	0xFF: "Graphics/cassette",
}

func (vm *vm) readPort(port byte) byte {
	/// log.Printf("Reading port %02X", port)
	switch port {
	case 0x3F:
		// Unmapped, don't crash.
		return 0xFF
	case 0xE0:
		// IRQ latch read.
		return ^vm.cpu.irqLatch
	case 0xE4:
		// NMI latch read.
		return ^vm.cpu.nmiLatch
	case 0xE8:
		// UART modem.
		return 0xFF
	case 0xE9:
		// UART switches.
		return 0xFF
	case 0xEA:
		// UART status.
		return 0xFF
	case 0xEB:
		// UART data.
		return 0xFF
	case 0xEC, 0xED, 0xEE, 0xEF:
		// Acknowledge timer.
		vm.cpu.timerInterrupt(false)
		return 0xFF
	case 0xF0:
		// Disk status.
		return vm.readDiskStatus()
	case 0xF1:
		// Disk track.
		return vm.readDiskTrack()
	case 0xF2:
		// Disk sector.
		return vm.readDiskSector()
	case 0xF3:
		// Disk data.
		return vm.readDiskData()
	case 0xF8:
		// Printer status. Printer selected, ready, with paper, not busy.
		return 0x30
	case 0xFF:
		// Cassette and various flags.
		cassetteStatus := byte(0)
		return (vm.modeImage & 0x7E) | cassetteStatus
	}

	panic(fmt.Sprintf("Can't read from unknown port %02X", port))
}

func (vm *vm) writePort(port byte, value byte) {
	/// log.Printf("Writing %02X to port %02X", value, port)
	switch port {
	case 0x84, 0x85, 0x86, 0x87:
		// Model 4 video page, etc. Ignore.
	case 0x1F:
		// Don't know. Don't crash.
	case 0xE0:
		// Set interrupt mask.
		vm.cpu.setIrqMask(value)
	case 0xE4, 0xE5, 0xE6, 0xE7:
		// NMI state.
		vm.cpu.setNmiMask(value)
	case 0xE8:
		// UART reset.
	case 0xE9:
		// UART baud.
	case 0xEA:
		// UART control.
	case 0xEB:
		// UART data.
	case 0xEC, 0xED, 0xEE, 0xEF:
		// Various controls.
		vm.modeImage = value
		/// trs_cassette_motor((value & 0x02) >> 1)
		/// trs_screen_expanded((value & 0x04) >> 2)
		/// trs_screen_alternate(!((value & 0x08) >> 3))
		/// trs_timer_speed((value & 0x40) >> 6)
	case 0xF0:
		// Disk command.
		vm.writeDiskCommand(value)
	case 0xF1:
		// Disk track.
		vm.writeDiskTrack(value)
	case 0xF2:
		// Disk sector.
		vm.writeDiskSector(value)
	case 0xF3:
		// Disk data.
		vm.writeDiskData(value)
	case 0xF4, 0xF5, 0xF6, 0xF7:
		// Disk select.
		vm.writeDiskSelect(value)
	case 0xF8, 0xF9, 0xFA, 0xFB:
		// Printer write.
		log.Printf("Writing %02X on printer", value)
	case 0xFC, 0xFD, 0xFE, 0xFF:
		if value&0x20 != 0 {
			// Model III Micro Labs graphics card.
			log.Printf("Sending %02X to Micro Labs graphics card", value)
		} else {
			// Do cassette emulation.
			log.Printf("Sending %02X to cassette", value)
			/// trs_cassette_out(value & 3);
		}
	default:
		panic(fmt.Sprintf("Can't write %02X to unknown port %02X", value, port))
	}
}
