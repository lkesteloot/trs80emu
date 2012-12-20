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

// Read status bits.
const (
	diskDrq = 0x02
	diskLostData = 0x04
	diskNotFound = 0x10
	diskRecType = 0x60
	disk1791FB = 0x00
	disk1791F8 = 0x20
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

	// Never have more than this many tracks.
	maxTracks = 255

	// JV1 info.
	jv1BytesPerSector = 256
	jv1SectorsPerTrack = 10
)

const (
	diskCommandMask = 0xF0

	// Type I commands: cccchvrr, where
	//     cccc = command number
	//     h = head load
	//     v = verify (i.e., read next address to check we're on the right track)
	//     rr = step rate:  00=6ms, 01=12ms, 10=20ms, 11=40ms
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
	//     cccc = command number
	//     e = delay for head engage (10ms)
	//     b = side expected
	//     c = side compare (0=disable, 1=enable)
	//     d = select data address mark (writes only, 0 for reads):
	//         0=FB (normal), 1=F8 (deleted)
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
	//     cccc = command number
	//     xxx = ?? (usually 010)
	//     s = 1=READTRK no synchronize; otherwise 0
	diskReadAdr = 0xc0
	diskReadTrk = 0xe0
	diskWriteTrk = 0xf0

	// Type IV command: cccciiii, where
	//     cccc = command number
	//     iiii = bitmask of events to terminate and interrupt on (unused on trs80).
	//            0000 for immediate terminate with no interrupt.
	diskForceInt = 0xd0
)

// -1 = unspecified
// 0 = front
// 1 = back
type side int

// Data about the disk controller. We only emulate the WD1791/93, not the
// Model I's WD1771.
type fdc struct {
	// Registers.
	status byte
	track byte
	sector byte
	data byte

	// Various state.
	currentCommand byte
	byteCount int  // Bytes left to transfer this command.
	side side
	doubleDensity bool
	currentDrive int
	motorIsOn bool
	motorTimeout uint64
	lastReadAdr int // Id index found by last readadr.

	// Disks themselves.
	driveCount int
	disk disk
}

// Data about the floppy that has been inserted.
type disk struct {
	// Which physical track the head is on.
	physicalTrack byte

	// Where we're pointing to within the data.
	dataOffset int

	// Nil if no disk is inserted, or the contents of the disk.
	data []byte
}

// Sets the side variable based on the boolean, which uses true for back and
// false for front.
func (side *side) setFromBoolean(value bool) {
	if value {
		*side = 1
	} else {
		*side = 0
	}
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
	cpu.fdc.byteCount = 0
	cpu.fdc.side = 0
	cpu.fdc.doubleDensity = false
	cpu.fdc.currentDrive = 0
	cpu.fdc.motorIsOn = false
	cpu.fdc.motorTimeout = 0
	cpu.fdc.lastReadAdr = -1

	if powerOn {
		cpu.fdc.disk.physicalTrack = 0
	}

	// XXX Recognizes new disks inserted.
	// trs_disk_change_all()

	// Cancel any pending disk event.
	cpu.events.cancelDiskEvents()

	cpu.fdc.driveCount = 1
}

// Event used for delayed command completion.  Clears BUSY,
// sets any additional bits specified, and generates a command
// completion interrupt.
func (cpu *cpu) diskDone(bits byte) {
	cpu.fdc.status &^= diskBusy
	cpu.fdc.status |= bits
	cpu.diskIntrqInterrupt(true)
}

// Event to abort the last command with LOSTDATA if it is
// still in progress.
func (cpu *cpu) diskLostData(cmd byte) {
	if (cpu.fdc.currentCommand == cmd) {
		cpu.fdc.status &^= diskBusy
		cpu.fdc.status |= diskLostData
		cpu.fdc.byteCount = 0
		cpu.diskIntrqInterrupt(true)
	}
}

// Event used as a delayed command start. Sets DRQ, generates a DRQ interrupt,
// sets any additional bits specified, and schedules a diskLostData() event.
func (cpu *cpu) diskFirstDrq(bits byte) {
	cpu.fdc.status |= diskDrq | bits
	cpu.diskDrqInterrupt(true)
	// Evaluate this now, not when the callback is run.
	currentCommand := cpu.fdc.currentCommand
	cpu.addEvent(eventDiskLostData, func () { cpu.diskLostData(currentCommand) }, cpuHz/2)
}

func (cpu *cpu) checkDiskMotorOff() bool {
	stopped := cpu.clock > cpu.fdc.motorTimeout
	if stopped {
		cpu.setDiskMotor(false)
		cpu.fdc.status |= diskNotRdy

		if isReadWriteCommand(cpu.fdc.currentCommand) && (cpu.fdc.status & diskDrq) != 0 {
			// Also end the command and set Lost Data for good measure
			cpu.fdc.status = (cpu.fdc.status | diskLostData) & ^byte(diskBusy | diskDrq)
			cpu.fdc.byteCount = 0
		}
	}

	return stopped
}

func (cpu *cpu) setDiskMotor(value bool) {
	if cpu.fdc.motorIsOn != value {
		var intValue int
		if value {
			log.Print("Starting motor")
			intValue = 1
		} else {
			log.Print("Stopping motor")
			intValue = 0
		}
		// Update UI.
		if cpu.cpuUpdateCh != nil {
			cpu.cpuUpdateCh <- cpuUpdate{Cmd: "motor", Data: intValue}
		}
		cpu.fdc.motorIsOn = value
	}
}

// Return a value in [0,1) indicating how far we've rotated
// from the leading edge of the index hole. For the first diskHoleWidth we're
// on the hole itself.
func (cpu *cpu) diskAngle() float32 {
  return float32(cpu.clock % clocksPerRevolution) / float32(clocksPerRevolution)
}

func isReadWriteCommand(cmd byte) bool {
	cmdType := commandType(cmd)
	return cmdType == 2 || cmdType == 3
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
	if isReadWriteCommand(cpu.fdc.currentCommand) {
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
			cpu.setDiskMotor(false)
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

func (cpu *cpu) writeDiskCommand(cmd byte) {
	// Cancel any ongoing command.
	cpu.events.cancelEvents(eventDiskLostData)

	cpu.diskIntrqInterrupt(false)
	cpu.fdc.byteCount = 0
	cpu.fdc.currentCommand = cmd

	switch cmd & diskCommandMask {
	case diskRestore:
		cpu.fdc.lastReadAdr = -1
		cpu.fdc.disk.physicalTrack = 0
		cpu.fdc.track = 0
		cpu.fdc.status = diskTrkZero|diskBusy
		// We don't implement verification. XXX
		/// if (cmd & TRSDISK_VBIT) verify()
		cpu.addEvent(eventDiskDone, func () { cpu.diskDone(0) }, 2000)
	case diskSeek:
		panic("Don't handle diskSeek")
	case diskStep:
		panic("Don't handle diskStep")
	case diskStepU:
		panic("Don't handle diskStepU")
	case diskStepIn:
		panic("Don't handle diskStepIn")
	case diskStepInU:
		panic("Don't handle diskStepInU")
	case diskStepOut:
		panic("Don't handle diskStepOut")
	case diskStepOutU:
		panic("Don't handle diskStepOutU")
	case diskRead:
		cpu.fdc.lastReadAdr = -1
		cpu.fdc.status = 0
		goalSide := side(-1)
		if cmd & diskCMask != 0 {
			goalSide.setFromBoolean((cmd & diskBMask) != 0)
		}
		sectorIndex := cpu.searchSector(int(cpu.fdc.sector), goalSide)
		if sectorIndex == -1 {
			cpu.fdc.status |= diskBusy
			cpu.addEvent(eventDiskDone, func () { cpu.diskDone(0) }, 512)
		} else {
			disk := &cpu.fdc.disk
			var newStatus byte = 0
			if disk.physicalTrack == 17 {
				newStatus = disk1791F8
			}
			cpu.fdc.byteCount = jv1BytesPerSector
			disk.dataOffset = cpu.dataOffset(sectorIndex)
			cpu.fdc.status |= diskBusy
			cpu.addEvent(eventDiskFirstDrq, func () { cpu.diskFirstDrq(newStatus) }, 64)
		}
	case diskReadM:
		panic("Don't handle diskReadM")
	case diskWrite:
		panic("Don't handle diskWrite")
	case diskWriteM:
		panic("Don't handle diskWriteM")
	case diskReadAdr:
		panic("Don't handle diskReadAdr")
	case diskReadTrk:
		panic("Don't handle diskReadTrk")
	case diskWriteTrk:
		panic("Don't handle diskWriteTrk")
	case diskForceInt:
		// Stop whatever is going on and forget it.
		cpu.events.cancelDiskEvents()
		cpu.fdc.status = 0
		cpu.updateDiskStatus()
		if (cmd & 0x07) != 0 {
			panic("Conditional interrupt features not implemented")
		} else if (cmd & 0x08) != 0 {
			// Immediate interrupt.
			cpu.diskIntrqInterrupt(true)
		} else {
			cpu.diskIntrqInterrupt(false)
		}
	default:
		panic(fmt.Sprintf("Unknown disk command %02X", cmd))
	}
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
	cpu.fdc.side.setFromBoolean((value & diskSide) != 0)
	cpu.fdc.doubleDensity = (value & diskMfm) != 0
	if value & diskWait != 0 {
		// If there was an event pending, simulate waiting until it was due.
		event := cpu.events.getFirstEvent(eventDiskLostData)
		if event != nil {
			cpu.clock = event.clock
			cpu.events.dispatch(cpu.clock)
		}
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
		cpu.setDiskMotor(true)
		cpu.fdc.motorTimeout = cpu.clock + motorTimeAfterSelect*cpuHz
		cpu.diskMotorOffInterrupt(false)
	}
}

// Search for a sector on the current physical track.  Return its index within
// the emulated disk's array of sectors.  Set status and return -1 if there is
// no such sector.  If sector == -1, return the first sector found if any.  If
// side == 0 or 1, perform side compare against sector ID; if -1, don't.
func (cpu *cpu) searchSector(sector int, side side) int {
	disk := &cpu.fdc.disk

	if disk.physicalTrack < 0 ||
		disk.physicalTrack >= maxTracks ||
		cpu.fdc.side == 1 ||
		side == 1 ||
		sector >= jv1SectorsPerTrack ||
		disk.data == nil ||
		disk.physicalTrack != cpu.fdc.track {

		cpu.fdc.status |= diskNotFound
		return -1
    }

	if sector < 0 {
		sector = 0
	}

    return jv1SectorsPerTrack*int(disk.physicalTrack) + sector
}

func (cpu *cpu) dataOffset(index int) int {
	return index*jv1BytesPerSector
}
