package main

// This file borrows heavily from the xtrs file trs_disk.c.

import (
	"fmt"
	"log"
	"io/ioutil"
)

// Type I status bits.
const (
	diskBusy = 1 << iota
	diskIndex
	diskTrkZero
	diskCrcErr
	diskSeekErr
	diskHeadEngd
	diskWritePrt
	diskNotRdy
)

// Select register bits.
const (
	diskSide = 0x10 << iota
	diskPrecomp
	diskWait
	diskMfm
)

const (
	// How long the disk motor stays on after drive selected (in seconds).
	motorTimeAfterSelect = 2

	// Width of the index hole as a fraction of the entire circumference.
	diskHoleWidth = 0.01

	// Speed of disk.
	diskRpm = 300
	clocksPerRevolution = cpuHz*60/diskRpm

	// Whether disks are write-protected.
	writeProtection = true
)

const (
	diskCommandMask = 0xF0

	// Type I commands: cccchvrr, where
	// cccc = command number
	// h = head load
	// v = verify (i.e., read next address to check we're on the right track)
	// rr = step rate:  00=6ms, 01=12ms, 10=20ms, 11=40ms
	diskRestore = 0x00
	diskSeek = 0x10
	diskStep = 0x20  // Doesn't update track register
	diskStepU = 0x30  // Updates track register
	diskStepIn = 0x40
	diskStepInU = 0x50
	diskStepOut = 0x60
	diskStepOutU = 0x70
	diskUMask = 0x10
	diskHMask = 0x08
	diskVMask = 0x04

	// Type II commands: ccccbecd, where
	// cccc = command number
	// e = delay for head engage (10ms)
	// 1771:
	//   b = 1=IBM format, 0=nonIBM format
	//   cd = select data address mark (writes only, 00 for reads):
	//        00=FB (normal), 01=FA, 10=F9, 11=F8 (deleted)
	//  1791:
	//   b = side expected
	//   c = side compare (0=disable, 1=enable)
	//   d = select data address mark (writes only, 0 for reads):
	//       0=FB (normal), 1=F8 (deleted)
	diskRead = 0x80  // Single sector
	diskReadM = 0x90  // Multiple sectors
	diskWrite = 0xa0
	diskWriteM = 0xb0
	diskMMask = 0x10
	diskBMask = 0x08
	diskEMask = 0x04
	diskCMask = 0x02
	diskDMask = 0x01

	// Type III commands: ccccxxxs (?), where
	// cccc = command number
	// xxx = ?? (usually 010)
	// s = 1=READTRK no synchronize; otherwise 0
	diskReadAdr = 0xc0
	diskReadTrk = 0xe0
	diskWriteTrk = 0xf0

	// Type IV command: cccciiii, where
	// cccc = command number
	// iiii = bitmask of events to terminate and interrupt on (unused on trs80).
	//        0000 for immediate terminate with no interrupt.
	diskForceInt = 0xd0
)

// Data about the disk controller.
type fdc struct {
	// Registers.
	status byte
	track byte
	sector byte
	data byte

	// Various state.
	currentCommand byte
	backSide bool
	doubleDensity bool
	currentDrive int
	motorTimeout uint64

	// Disks themselves.
	driveCount int
	disk disk
}

// Data about the floppy that has been inserted.
type disk struct {
	// Which physical track the head is on.
	physicalTrack int

	// Nil if no disk is inserted, or the contents of the disk.
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

func (cpu *cpu) diskInit(powerOn bool) {
	// Registers.
	cpu.fdc.status = diskNotRdy|diskTrkZero
	cpu.fdc.track = 0
	cpu.fdc.sector = 0
	cpu.fdc.data = 0

	// Various state.
	cpu.fdc.currentCommand = diskRestore
	cpu.fdc.backSide = false
	cpu.fdc.doubleDensity = false
	cpu.fdc.currentDrive = 0
	cpu.fdc.motorTimeout = 0

	/*
	cpu.fdc.lastdirection = 1
	cpu.fdc.bytecount = 0
	cpu.fdc.format = FMT_DONE
	cpu.fdc.format_bytecount = 0
	cpu.fdc.format_sec = 0
	cpu.fdc.controller = TRSDISK_P1791
	cpu.fdc.last_readadr = -1
	*/

	if powerOn {
		cpu.fdc.disk.physicalTrack = 0
	}

	// XXX Recognizes new disks inserted.
	// trs_disk_change_all()
	// XXX cancel any pending event.
	// trs_cancel_event()

	cpu.fdc.driveCount = 1
}

func (cpu *cpu) checkDiskMotorOff() bool {
	stopped := cpu.clock > cpu.fdc.motorTimeout

	if stopped {
		log.Print("Stopping motor")
		cpu.fdc.status |= diskNotRdy

		/*XXX
		cmdtype := commandType(cpu.fdc.currentCommand)
		if ((cmdtype == 2 || cmdtype == 3) && (cpu.fdc.status & TRSDISK_DRQ)) {
		  // Also end the command and set Lost Data for good measure
		  cpu.fdc.status = (cpu.fdc.status | TRSDISK_LOSTDATA) &
		~(TRSDISK_BUSY | TRSDISK_DRQ)
		  cpu.fdc.bytecount = 0
		}
		*/
	}

	return stopped
}

// Return a value in [0,1) indicating how far we've rotated
// from the leading edge of the index hole. For the first diskHoleWidth we're
// on the hole itself.
func (cpu *cpu) diskAngle() float32 {
  return float32(cpu.clock % clocksPerRevolution) / float32(clocksPerRevolution)
}

func commandType(cmd byte) int {
	switch cmd & diskCommandMask {
	case diskRestore, diskSeek, diskStep, diskStepU,
		diskStepIn, diskStepInU, diskStepOut, diskStepOutU:

		return 1
	case diskRead, diskReadM, diskWrite, diskWriteM:
		return 2
	case diskReadAdr, diskReadTrk, diskWriteTrk:
		return 3
	case diskForceInt:
		return 4
	}

	panic(fmt.Sprintf("Unknown type for command %02X", cmd))
}

func (cpu *cpu) updateDiskStatus() {
	switch commandType(cpu.fdc.currentCommand) {
	case 2, 3:
		return
	}

	if cpu.fdc.disk.data == nil {
		cpu.fdc.status |= diskIndex
	} else {
		if cpu.diskAngle() < diskHoleWidth {
			cpu.fdc.status |= diskIndex
		} else {
			cpu.fdc.status &^= diskIndex
		}

		if writeProtection {
			cpu.fdc.status |= diskWritePrt
		} else {
			cpu.fdc.status &^= diskWritePrt
		}
	}

	if cpu.fdc.disk.physicalTrack == 0 {
		cpu.fdc.status |= diskTrkZero
	} else {
		cpu.fdc.status &^= diskTrkZero
	}

	// RDY and HLT inputs are wired together on TRS-80 I/III/4/4P.
	if cpu.fdc.status & diskNotRdy != 0 {
		cpu.fdc.status &^= diskHeadEngd
	} else {
		cpu.fdc.status |= diskHeadEngd
	}
}

func (cpu *cpu) readDiskStatus() byte {
	if cpu.fdc.driveCount == 0 {
		return 0xFF
	}

	cpu.updateDiskStatus()
	if cpu.fdc.status & diskNotRdy == 0 {
		if cpu.clock > cpu.fdc.motorTimeout {
			// Motor stopped.
			cpu.fdc.status |= diskNotRdy
		}
	}

	cpu.diskIntrqInterrupt(false)

	return cpu.fdc.status
}

func (cpu *cpu) readDiskTrack() byte {
	return cpu.fdc.track
}

func (cpu *cpu) readDiskSector() byte {
	return cpu.fdc.sector
}

func (cpu *cpu) readDiskData() byte {
	panic("readDiskData")
}

func (cpu *cpu) writeDiskCommand(value byte) {
	panic("writeDiskCommand")
}

func (cpu *cpu) writeDiskTrack(value byte) {
	cpu.fdc.track = value
}

func (cpu *cpu) writeDiskSector(value byte) {
	cpu.fdc.sector = value
}

func (cpu *cpu) writeDiskData(value byte) {
	panic("writeDiskData")
}

func (cpu *cpu) writeDiskSelect(value byte) {
	cpu.fdc.status &^= diskNotRdy
	cpu.fdc.backSide = (value & diskSide) != 0
	cpu.fdc.doubleDensity = (value & diskMfm) != 0
	if value & diskWait != 0 {
		// If there was an event pending, simulate waiting until it was due.
		/* XXX
		if (trs_event_scheduled() != NULL &&
		trs_event_scheduled() != trs_disk_lostdata) {
			z80_state.t_count = z80_state.sched
			trs_do_event()
		}
		*/
	}

	switch value & 0x0F {
	case 0:
		cpu.fdc.status |= diskNotRdy
	case 1:
		cpu.fdc.currentDrive = 0
	default:
		// Could extend this if we wanted more than 2 disks. See fdc.driveCount.
		panic("Disk not handled")
	}

	// If a drive was selected, turn on its motor.
	if cpu.fdc.status & diskNotRdy == 0 {
		log.Print("Starting motor")
		cpu.fdc.motorTimeout = cpu.clock + motorTimeAfterSelect*cpuHz
		cpu.diskMotorOffInterrupt(false)
	}
}
