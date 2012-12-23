package main

// This file borrows heavily from the xtrs file trs_disk.c.

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
)

const (
	diskDebug = true
	diskSortDebug = true
)

// Type I status bits.
const (
	diskBusy    = 1 << iota
	diskIndex   // Over index hole.
	diskTrkZero // On track 0.
	diskCrcErr
	diskSeekErr
	diskHeadEngd // Head engaged.
	diskWritePrt // Write-protected.
	diskNotRdy   // Disk not ready (motor not running).
)

// Read status bits.
const (
	diskDrq      = 0x02
	diskLostData = 0x04
	diskNotFound = 0x10
	diskRecType  = 0x60
	disk1791FB   = 0x00
	disk1791F8   = 0x20
)

// Select register bits for writeDiskSelect().
const (
	diskDrive0 = 1 << iota
	diskDrive1
	diskDrive2
	diskDrive3
	diskSide // 0 = front, 1 = back.
	diskPrecomp
	diskWait
	diskMfm // Double density.

	diskDriveMask = diskDrive0 | diskDrive1 | diskDrive2 | diskDrive3
)

const (
	// How long the disk motor stays on after drive selected (in seconds).
	motorTimeAfterSelect = 2

	// Width of the index hole as a fraction of the entire circumference.
	diskHoleWidth = 0.01

	// Speed of disk.
	diskRpm             = 300
	clocksPerRevolution = cpuHz * 60 / diskRpm

	// Whether disks are write-protected.
	writeProtection = true

	// Never have more than this many tracks.
	maxTracks = 255

	// I don't know what this is but it defaults to false on xtrs.
	diskTrueDam = false

	// JV1 info.
	jv1BytesPerSector  = 256
	jv1SectorsPerTrack = 10

	// JV3 info.
	jv3MaxSides        = 2                      // Number of sides supported by this format.
	jv3IdStart         = 0                      // Where in file the IDs start.
	jv3SectorStart     = 34 * 256               // Start of sectors within file (end of IDs).
	jv3SectorsPerBlock = jv3SectorStart / 3     // Number of jv3Sector structs per info block.
	jv3SectorsMax      = 2 * jv3SectorsPerBlock // There are two info blocks maximum.
)

const (
	// for writeDiskCommand().
	diskCommandMask = 0xF0

	// Type I commands: cccchvrr, where
	//     cccc = command number
	//     h = head load
	//     v = verify (i.e., read next address to check we're on the right track)
	//     rr = step rate:  00=6ms, 01=12ms, 10=20ms, 11=40ms
	diskRestore  = 0x00
	diskSeek     = 0x10
	diskStep     = 0x20 // Doesn't update track register
	diskStepU    = 0x30 // Updates track register
	diskStepIn   = 0x40
	diskStepInU  = 0x50
	diskStepOut  = 0x60
	diskStepOutU = 0x70
	diskUMask    = 0x10
	diskHMask    = 0x08
	diskVMask    = 0x04

	// Type II commands: ccccbecd, where
	//     cccc = command number
	//     e = delay for head engage (10ms)
	//     b = side expected
	//     c = side compare (0=disable, 1=enable)
	//     d = select data address mark (writes only, 0 for reads):
	//         0=FB (normal), 1=F8 (deleted)
	diskRead   = 0x80 // Single sector
	diskReadM  = 0x90 // Multiple sectors
	diskWrite  = 0xa0
	diskWriteM = 0xb0
	diskMMask  = 0x10
	diskBMask  = 0x08
	diskEMask  = 0x04
	diskCMask  = 0x02
	diskDMask  = 0x01

	// Type III commands: ccccxxxs (?), where
	//     cccc = command number
	//     xxx = ?? (usually 010)
	//     s = 1=READTRK no synchronize; otherwise 0
	diskReadAdr  = 0xc0
	diskReadTrk  = 0xe0
	diskWriteTrk = 0xf0

	// Type IV command: cccciiii, where
	//     cccc = command number
	//     iiii = bitmask of events to terminate and interrupt on (unused on trs80).
	//            0000 for immediate terminate with no interrupt.
	diskForceInt = 0xd0
)

// JV3 flags and constants.
const (
	jv3Density = 0x80 // 1=dden, 0=sden
	jv3Dam     = 0x60 // Data address mark; values follow.
	jv3DamSdFB = 0x00
	jv3DamSdFA = 0x20
	jv3DamSdF9 = 0x40
	jv3DamSdF8 = 0x60
	jv3DamDdFB = 0x00
	jv3DamDdF8 = 0x20
	jv3Side    = 0x10 // 0=side 0, 1=side 1
	jv3Error   = 0x08 // 0=ok, 1=CRC error
	jv3NonIbm  = 0x04 // 0=normal, 1=short (for VTOS 3.0, xtrs only)
	jv3Size    = 0x03 // See comment in getSizeCode().

	jv3Free  = 0xff // In track/sector fields
	jv3FreeF = 0xfc // In flags field, or'd with size code
)

// Emulation types.
type emulationType uint

const (
	emuNone = emulationType(iota)
	emuJv1
	emuJv3
)

// Data about the disk controller. We only emulate the WD1791/93, not the
// Model I's WD1771.
type fdc struct {
	// Registers.
	status byte
	track  byte
	sector byte
	data   byte

	// Various state.
	currentCommand byte
	byteCount      int // Bytes left to transfer this command.
	side           side
	doubleDensity  bool
	currentDrive   int
	motorIsOn      bool
	motorTimeout   uint64
	lastReadAdr    int // Id index found by last readadr.

	// Disks themselves.
	driveCount int
	disk       disk
}

// Data about the floppy that has been inserted.
type disk struct {
	// What kind of diskette this is.
	emulationType emulationType

	// Which physical track the head is on.
	physicalTrack byte

	// Where we're pointing to within the data.
	dataOffset int

	// Nil if no disk is inserted, or the contents of the disk.
	data []byte

	// JV3-specific data.
	jv3 jv3
}

// JV3-specific data.
type jv3 struct {
	freeId      [4]int                       // The first free id, if any, of each size.
	lastUsedId  int                          // Id of the last used sector.
	blockCount  int                          // Number of blocks of ids, 1 or 2.
	sortedValid bool                         // Whether the sortedId array is valid.
	id          [jv3SectorsMax + 1]jv3Sector // Extra one is a loop sentinel.
	offset      [jv3SectorsMax + 1]int       // Offset into file for each id.
	sortedId    [jv3SectorsMax + 1]int       // Mapping from sorted id[] to real one.
	trackStart  [maxTracks][jv3MaxSides]int  // Where each side/side track starts in id.
}

// The first block of a JV3 file has jv3SectorsPerBlock of these.
type jv3Sector struct {
	track, sector, flags byte
}

// Sets all elements to jv3Free, making the sector as free.
func (id *jv3Sector) makeFree() {
	id.track = jv3Free
	id.sector = jv3Free
	id.flags = jv3Free
}

// Fill the three bytes from an array.
func (id *jv3Sector) fillFromSlice(data []byte) {
	id.track = data[0]
	id.sector = data[1]
	id.flags = data[2]
	if diskSortDebug {
		log.Printf("jv3 data: track %02X, sector %02X, flags %02X", id.track, id.sector, id.flags)
	}
}

func (id *jv3Sector) side() (side side) {
	side.setFromBoolean(id.flags&jv3Side != 0)
	return
}

func (id *jv3Sector) doubleDensity() bool {
	return id.flags&jv3Density != 0
}

// Return the size of this sector in bytes.
func (id *jv3Sector) getSize() int {
	return 128 << id.getSizeCode()
}

// Return the size code for this sector: 0-3 for 128, 256, 512, 1024.
func (id *jv3Sector) getSizeCode() byte {
	// In used sectors: 0=256,1=128,2=1024,3=512
	// In free sectors: 0=512,1=1024,2=128,3=256
	code := id.flags & jv3Size

	sectorIsFree := id.track == jv3Free

	var flipMask byte
	if sectorIsFree {
		flipMask = 2
	} else {
		flipMask = 1
	}

	return code ^ flipMask
}

// -1 = unspecified
// 0 = front
// 1 = back
type side int

func sideFromBoolean(value bool) (side side) {
	side.setFromBoolean(value)
	return
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

	// Figure out what kind of disk this is.
	disk.recognizeDisk()

	return nil
}

// Set the emulationType field and fill initial data structures.
func (disk *disk) recognizeDisk() {
	// Just recognize type based on the size of the file. The xtrs algorithm
	// is complex.
	switch len(disk.data) {
	case 0:
		disk.emulationType = emuNone
	case 89600:
		disk.emulationType = emuJv1
	case 193024, 377344:
		disk.emulationType = emuJv3
		disk.loadJv3Data(0)
	default:
		log.Fatalf("Don't know format of %d-byte disk", len(disk.data))
	}
}

func (disk *disk) loadJv3Data(drive int) {
	// Mark the whole id array as free.
	for i := 0; i < len(disk.jv3.id); i++ {
		disk.jv3.id[i].makeFree()
	}

	disk.jv3.blockCount = 0

	// Load first block.
	offset := disk.loadJv3Block(0, jv3IdStart)

	// Load second block, if it's there.
	disk.loadJv3Block(jv3SectorsPerBlock, offset)

	// We pre-compute some information about used and free sectors that we'll
	// use later when writing.
	for i := 0; i < 4; i++ {
		disk.jv3.freeId[i] = jv3SectorsMax
	}

	disk.jv3.lastUsedId = -1
	for idIndex := 0; idIndex < jv3SectorsMax; idIndex++ {
		if disk.jv3.id[idIndex].track == jv3Free {
			sizeCode := disk.jv3.id[idIndex].getSizeCode()
			if disk.jv3.freeId[sizeCode] == jv3SectorsMax {
				disk.jv3.freeId[sizeCode] = idIndex
			}
		} else {
			disk.jv3.lastUsedId = idIndex
		}
	}
	disk.jv3.sortIds(drive)
}

func (disk *disk) loadJv3Block(idStart, blockStart int) int {
	// Make sure there's enough there to read.
	if blockStart+3*jv3SectorsPerBlock <= len(disk.data) {
		disk.jv3.blockCount++

		// Read the block into the sector info.
		start := blockStart
		for i := 0; i < jv3SectorsPerBlock; i++ {
			disk.jv3.id[idStart+i].fillFromSlice(disk.data[start : start+3])
			start += 3
		}
	}

	// Compute offsets of each sector.
	offset := blockStart + jv3SectorStart
	for i := 0; i < jv3SectorsPerBlock; i++ {
		disk.jv3.offset[idStart+i] = offset
		offset += disk.jv3.id[idStart+i].getSize()
	}

	return offset
}

func (cpu *cpu) diskInit(powerOn bool) {
	if diskDebug {
		log.Printf("diskInit(%v)", powerOn)
	}

	// Registers.
	cpu.fdc.status = diskNotRdy | diskTrkZero
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

	// Cancel any pending disk event.
	cpu.events.cancelEvents(eventDisk)

	cpu.fdc.driveCount = 1
}

// Event used for delayed command completion.  Clears BUSY,
// sets any additional bits specified, and generates a command
// completion interrupt.
func (cpu *cpu) diskDone(bits byte) {
	if diskDebug {
		log.Printf("diskDone(%02X)", bits)
	}

	cpu.fdc.status &^= diskBusy
	cpu.fdc.status |= bits
	cpu.diskIntrqInterrupt(true)
}

// Event to abort the last command with LOSTDATA if it is
// still in progress.
func (cpu *cpu) diskLostData(cmd byte) {
	if diskDebug {
		log.Printf("diskLostData(%02X)", cmd)
	}

	if cpu.fdc.currentCommand == cmd {
		cpu.fdc.status &^= diskBusy
		cpu.fdc.status |= diskLostData
		cpu.fdc.byteCount = 0
		cpu.diskIntrqInterrupt(true)
	}
}

// Event used as a delayed command start. Sets DRQ, generates a DRQ interrupt,
// sets any additional bits specified, and schedules a diskLostData() event.
func (cpu *cpu) diskFirstDrq(bits byte) {
	if diskDebug {
		log.Printf("diskFirstDrq(%v)", bits)
	}

	cpu.fdc.status |= diskDrq | bits
	cpu.diskDrqInterrupt(true)
	// Evaluate this now, not when the callback is run.
	currentCommand := cpu.fdc.currentCommand
	cpu.addEvent(eventDiskLostData, func() { cpu.diskLostData(currentCommand) }, cpuHz/2)
}

func (cpu *cpu) checkDiskMotorOff() bool {
	stopped := cpu.clock > cpu.fdc.motorTimeout
	if stopped {
		cpu.setDiskMotor(false)
		cpu.fdc.status |= diskNotRdy

		if isReadWriteCommand(cpu.fdc.currentCommand) && (cpu.fdc.status&diskDrq) != 0 {
			// Also end the command and set Lost Data for good measure
			cpu.fdc.status = (cpu.fdc.status | diskLostData) & ^byte(diskBusy|diskDrq)
			cpu.fdc.byteCount = 0
		}
	}

	return stopped
}

func (cpu *cpu) setDiskMotor(value bool) {
	if cpu.fdc.motorIsOn != value {
		var intValue int
		if value {
			if diskDebug {
				log.Print("Starting motor")
			}
			intValue = 1
		} else {
			if diskDebug {
				log.Print("Stopping motor")
			}
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
	return float32(cpu.clock%clocksPerRevolution) / float32(clocksPerRevolution)
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
		// Don't modify status.
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
	if cpu.fdc.status&diskNotRdy != 0 {
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

	// Turn off motor if it's running and been too long.
	if cpu.fdc.status&diskNotRdy == 0 {
		if cpu.clock > cpu.fdc.motorTimeout {
			cpu.setDiskMotor(false)
			cpu.fdc.status |= diskNotRdy
		}
	}

	cpu.diskIntrqInterrupt(false)

	if diskDebug {
		log.Printf("readDiskStatus() = %02X", cpu.fdc.status)
	}

	return cpu.fdc.status
}

func (cpu *cpu) readDiskTrack() byte {
	if diskDebug {
		log.Printf("readDiskTrack() = %02X", cpu.fdc.track)
	}

	return cpu.fdc.track
}

func (cpu *cpu) readDiskSector() byte {
	if diskDebug {
		log.Printf("readDiskSector() = %02X", cpu.fdc.sector)
	}

	return cpu.fdc.sector
}

func (cpu *cpu) readDiskData() byte {
	disk := &cpu.fdc.disk

	switch cpu.fdc.currentCommand & diskCommandMask {
	case diskRead:
		if cpu.fdc.byteCount > 0 && (cpu.fdc.status&diskDrq) != 0 {
			var c byte
			if disk.dataOffset >= len(disk.data) {
				c = 0xE5
				if disk.emulationType == emuJv3 {
					cpu.fdc.status &^= diskRecType
					cpu.fdc.status |= disk1791FB
				}
			} else {
				c = disk.data[disk.dataOffset]
				disk.dataOffset++
			}
			cpu.fdc.data = c
			cpu.fdc.byteCount--
			if cpu.fdc.byteCount <= 0 {
				cpu.fdc.byteCount = 0
				cpu.fdc.status &^= diskDrq
				cpu.diskDrqInterrupt(false)
				cpu.events.cancelEvents(eventDiskLostData)
				cpu.addEvent(eventDiskDone, func() { cpu.diskDone(0) }, 64)
			}
		}

	case diskReadAdr:
		panic("diskReadAdr")
		/*
			    if (cpu.fdc.byteCount <= 0 || !(cpu.fdc.status & TRSDISK_DRQ)) break
			    if (d->emutype == REAL) {
			#if 0
			      cpu.fdc.sector = d->u.real.buf[0]; //179x data sheet says this
			#else
			      cpu.fdc.track = d->u.real.buf[0]; //let's guess it meant this
			      cpu.fdc.sector = d->u.real.buf[2]; //1771 data sheet says this
			#endif
			      cpu.fdc.data = d->u.real.buf[6 - cpu.fdc.byteCount]

			    } else if (d->emutype == DMK) {
			      cpu.fdc.data = d->u.dmk.buf[d->u.dmk.curbyte]
			#if 0
			      if (cpu.fdc.byteCount == 6) {
				cpu.fdc.sector = cpu.fdc.data; //179x data sheet says this
			      }
			#else
			      if (cpu.fdc.byteCount == 6) {
				cpu.fdc.track = cpu.fdc.data; //let's guess it meant this!!
			      } else if (cpu.fdc.byteCount == 4) {
				cpu.fdc.sector = cpu.fdc.data;  //1771 data sheet says this
			      }
			#endif
			      d->u.dmk.curbyte += dmk_incr(d)

			    } else if (cpu.fdc.lastReadAdr >= 0) {
			      if (d->emutype == JV1) {
				switch (cpu.fdc.byteCount) {
				case 6:
				  cpu.fdc.data = d->physicalTrack
			#if 0
				  cpu.fdc.sector = d->physicalTrack; //179x data sheet says this
			#else
				  cpu.fdc.track = d->physicalTrack; //let's guess it meant this
			#endif
				  break
				case 5:
				  cpu.fdc.data = 0
				  break
				case 4:
				  cpu.fdc.data = jv1_interleave[cpu.fdc.lastReadAdr % JV1_SECPERTRK]
				  cpu.fdc.sector = cpu.fdc.data;  //1771 data sheet says this
				  break
				case 3:
				  cpu.fdc.data = 0x01;  // 256 bytes always
				  break
				case 2:
				case 1:
				  cpu.fdc.data = cpu.fdc.crc >> 8
				  break
				}
			      } else if (d->emutype == JV3) {
				sid = &d->u.jv3.id[d->u.jv3.sortedId[cpu.fdc.lastReadAdr]]
				switch (cpu.fdc.byteCount) {
				case 6:
				  cpu.fdc.data = sid->track
			#if 0
				  cpu.fdc.sector = sid->track; //179x data sheet says this
			#else
				  cpu.fdc.track = sid->track; //let's guess it meant this
			#endif
				  break
				case 5:
				  cpu.fdc.data = (sid->flags & jv3Side) != 0
				  break
				case 4:
				  cpu.fdc.data = sid->sector
				  cpu.fdc.sector = sid->sector;  //1771 data sheet says this
				  break
				case 3:
				  cpu.fdc.data =
				    id_index_to_size_code(d, d->u.jv3.sortedId[cpu.fdc.lastReadAdr])
				  break
				case 2:
				case 1:
				  cpu.fdc.data = cpu.fdc.crc >> 8
				  break
				}
			      }
			    }
			    cpu.fdc.crc = calc_crc1(cpu.fdc.crc, cpu.fdc.data)
			    cpu.fdc.byteCount--
			    if (cpu.fdc.byteCount <= 0) {
			      if (d->emutype == DMK && cpu.fdc.crc != 0) {
				cpu.fdc.status |= diskCrcErr
			      }
			      cpu.fdc.byteCount = 0
			      cpu.fdc.status &= ~TRSDISK_DRQ
			      trs_disk_drq_interrupt(0)
			      if (trs_event_scheduled() == trs_disk_lostdata) {
				trs_cancel_event()
			      }
			      trs_schedule_event(trs_disk_done, 0, 64)
			    }
			    break
		*/

	case diskReadTrk:
		panic("diskReadTrk")
		/*
			    // assert(emutype == DMK)
			    if (!(cpu.fdc.status & TRSDISK_DRQ)) break
			    if (cpu.fdc.byteCount > 0) {
			      cpu.fdc.data = d->u.dmk.buf[d->u.dmk.curbyte]
			      d->u.dmk.curbyte += dmk_incr(d)
			      cpu.fdc.byteCount = cpu.fdc.byteCount - 2 + cpu.fdc.doubleDensity
			    }
			    if (cpu.fdc.byteCount <= 0) {
			      cpu.fdc.byteCount = 0
			      cpu.fdc.status &= ~TRSDISK_DRQ
			      trs_disk_drq_interrupt(0)
			      if (trs_event_scheduled() == trs_disk_lostdata) {
				trs_cancel_event()
			      }
			      trs_schedule_event(trs_disk_done, 0, 64)
			    }
			    break
		*/
	default:
		// Might be okay, not sure.
		panic("Unhandled case in readDiskData()")
	}

	if diskDebug {
		log.Printf("readDiskData() = %02X (%d left)", cpu.fdc.data, cpu.fdc.byteCount)
	}

	return cpu.fdc.data
}

func (cpu *cpu) writeDiskCommand(cmd byte) {
	if diskDebug {
		log.Printf("writeDiskCommand(%02X)", cmd)
	}

	// Cancel "lost data" event.
	cpu.events.cancelEvents(eventDiskLostData)

	cpu.diskIntrqInterrupt(false)
	cpu.fdc.byteCount = 0
	cpu.fdc.currentCommand = cmd

	switch cmd & diskCommandMask {
	case diskRestore:
		cpu.fdc.lastReadAdr = -1
		cpu.fdc.disk.physicalTrack = 0
		cpu.fdc.track = 0
		cpu.fdc.status = diskTrkZero | diskBusy
		if cmd&diskVMask != 0 {
			cpu.diskVerify()
		}
		cpu.addEvent(eventDiskDone, func() { cpu.diskDone(0) }, 2000)
	case diskSeek:
		cpu.fdc.lastReadAdr = -1
		cpu.fdc.disk.physicalTrack += cpu.fdc.data - cpu.fdc.track
		cpu.fdc.track = cpu.fdc.data
		if cpu.fdc.disk.physicalTrack <= 0 {
			// cpu.fdc.track too?
			cpu.fdc.disk.physicalTrack = 0
			cpu.fdc.status = diskTrkZero | diskBusy
		} else {
			cpu.fdc.status = diskBusy
		}
		// Should this set lastDirection?
		if cmd&diskVMask != 0 {
			cpu.diskVerify()
		}
		cpu.addEvent(eventDiskDone, func() { cpu.diskDone(0) }, 2000)
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
		if cmd&diskCMask != 0 {
			goalSide.setFromBoolean((cmd & diskBMask) != 0)
		}
		sectorIndex := cpu.searchSector(int(cpu.fdc.sector), goalSide)
		if sectorIndex == -1 {
			cpu.fdc.status |= diskBusy
			cpu.addEvent(eventDiskDone, func() { cpu.diskDone(0) }, 512)
			log.Printf("Didn't find sector %02X on track %02X",
				cpu.fdc.sector, cpu.fdc.disk.physicalTrack)
		} else {
			disk := &cpu.fdc.disk
			var newStatus byte = 0
			switch disk.emulationType {
			case emuJv1:
				if disk.physicalTrack == 17 {
					newStatus = disk1791F8
				}
				cpu.fdc.byteCount = jv1BytesPerSector
				disk.dataOffset = disk.getDataOffset(sectorIndex)
			case emuJv3:
				if !cpu.fdc.doubleDensity {
					// Single density 179x.
					switch disk.jv3.id[sectorIndex].flags & jv3Dam {
					case jv3DamSdFB:
						newStatus = disk1791FB
						break
					case jv3DamSdFA:
						if diskTrueDam {
							newStatus = disk1791FB
						} else {
							newStatus = disk1791F8
						}
						break
					case jv3DamSdF9:
						newStatus = disk1791F8
						break
					case jv3DamSdF8:
						newStatus = disk1791F8
						break
					}
				} else {
					// Double density 179x.
					switch disk.jv3.id[sectorIndex].flags & jv3Dam {
					default: /*impossible*/
					case jv3DamDdFB:
						newStatus = disk1791FB
						break
					case jv3DamDdF8:
						newStatus = disk1791F8
						break
					}
				}
				if disk.jv3.id[sectorIndex].flags&jv3Error != 0 {
					newStatus |= diskCrcErr
				}
				cpu.fdc.byteCount = disk.jv3.id[sectorIndex].getSize()
				disk.dataOffset = disk.getDataOffset(sectorIndex)
			default:
				panic("Unhandled case in diskRead")
			}
			cpu.fdc.status |= diskBusy
			cpu.addEvent(eventDiskFirstDrq, func() { cpu.diskFirstDrq(newStatus) }, 64)
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
		cpu.events.cancelEvents(eventDisk)
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
	if diskDebug {
		log.Printf("writeDiskTrack(%02X)", value)
	}

	cpu.fdc.track = value
}

func (cpu *cpu) writeDiskSector(value byte) {
	if diskDebug {
		log.Printf("writeDiskSector(%02X)", value)
	}

	cpu.fdc.sector = value
}

func (cpu *cpu) writeDiskData(value byte) {
	if diskDebug {
		log.Printf("writeDiskData(%02X)", value)
	}

	switch cpu.fdc.currentCommand & diskCommandMask {
	case diskWrite:
		panic("diskWrite")
	case diskWriteTrk:
		panic("diskWriteTrk")
	default:
		// No action, just fall through and store data.
		break
	}

	cpu.fdc.data = value
}

func (cpu *cpu) writeDiskSelect(value byte) {
	if diskDebug {
		log.Printf("writeDiskSelect(%02X)", value)
	}

	cpu.fdc.status &^= diskNotRdy
	cpu.fdc.side.setFromBoolean((value & diskSide) != 0)
	cpu.fdc.doubleDensity = (value & diskMfm) != 0
	if value&diskWait != 0 {
		// If there was an event pending, simulate waiting until it was due.
		event := cpu.events.getFirstEvent(eventDisk &^ eventDiskLostData)
		if event != nil {
			if diskDebug {
				log.Printf("Advancing clock from %d to %d", cpu.clock, event.clock)
			}
			cpu.clock = event.clock
			cpu.events.dispatch(cpu.clock)
		}
	}

	// Which drive is being enabled?
	switch value & diskDriveMask {
	case 0:
		cpu.fdc.status |= diskNotRdy
	case diskDrive0:
		cpu.fdc.currentDrive = 0
	case diskDrive1:
		cpu.fdc.currentDrive = 1
	case diskDrive2:
		cpu.fdc.currentDrive = 2
	case diskDrive3:
		cpu.fdc.currentDrive = 3
	default:
		panic("Disk not handled")
	}

	// Sanity check.
	if cpu.fdc.currentDrive >= cpu.fdc.driveCount {
		// XXX?
		log.Printf("Drive too high (%d >= %d)", cpu.fdc.currentDrive, cpu.fdc.driveCount)
	}

	// If a drive was selected, turn on its motor.
	if cpu.fdc.status&diskNotRdy == 0 {
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

	switch disk.emulationType {
	case emuNone:
		cpu.fdc.status |= diskNotFound
		return -1
	case emuJv1:
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
	case emuJv3:
		if disk.physicalTrack < 0 ||
			disk.physicalTrack >= maxTracks ||
			cpu.fdc.side >= jv3MaxSides ||
			(side != -1 && side != cpu.fdc.side) ||
			disk.physicalTrack != cpu.fdc.track ||
			disk.data == nil {

			cpu.fdc.status |= diskNotFound
			return -1
		}
		if !disk.jv3.sortedValid {
			disk.jv3.sortIds(cpu.fdc.currentDrive)
		}

		i := disk.jv3.trackStart[disk.physicalTrack][cpu.fdc.side]
		if i != -1 {
			for {
				id := disk.jv3.sortedId[i]
				sid := &disk.jv3.id[id]
				if sid.track != disk.physicalTrack ||
					sid.side() != cpu.fdc.side {

					break
				}
				if (sector == -1 || int(sid.sector) == sector) &&
					sid.doubleDensity() == cpu.fdc.doubleDensity {

					return id
				}
				i++
			}
		}
		cpu.fdc.status |= diskNotFound
		return -1
	}

	panic("Unhandled case in searchSector()")
}

func (disk *disk) getDataOffset(index int) int {
	switch disk.emulationType {
	case emuJv1:
		return index * jv1BytesPerSector
	case emuJv3:
		return disk.jv3.offset[index]
	}

	panic("Unimplemented case in getDataOffset()")
}

// Verify that head is on the expected track.
func (cpu *cpu) diskVerify() {
	disk := &cpu.fdc.disk

	switch disk.emulationType {
	case emuNone:
		cpu.fdc.status |= diskNotFound
	case emuJv1:
		if disk.data == nil {
			cpu.fdc.status |= diskNotFound
		}
		if cpu.fdc.doubleDensity {
			cpu.fdc.status |= diskNotFound
		} else if cpu.fdc.track != disk.physicalTrack {
			cpu.fdc.status |= diskSeekErr
		}
	case emuJv3:
		// diskSeekErr == diskNotFound
		cpu.searchSector(-1, -1)
	}
}

// Satisfy the sort.Interface interface so we can sort the []sortedId slice.
func (jv3 *jv3) Len() int {
	return len(jv3.sortedId)
}
func (jv3 *jv3) Less(i, j int) bool {
	// Sort first by track, second by side, third by position in emulated-disk
	// sector array (i.e., physical sector order on track).
	si := jv3.sortedId[i]
	sj := jv3.sortedId[j]
	idi := &jv3.id[si]
	idj := &jv3.id[sj]

	return idi.track < idj.track ||
		(idi.track == idj.track && (idi.side() < idj.side() ||
			(idi.side() == idj.side() && si < sj)))
}
func (jv3 *jv3) Swap(i, j int) {
	jv3.sortedId[i], jv3.sortedId[j] = jv3.sortedId[j], jv3.sortedId[i]
}

// (Re-)create the sortedId data structure for the given drive.
func (jv3 *jv3) sortIds(drive int) {
	// Start with one-to-one map.
	for i := 0; i <= jv3SectorsMax; i++ {
		jv3.sortedId[i] = i
	}

	// Sort. See the Len(), Less(), and Swap() methods on jv3.
	sort.Sort(jv3)

	// Figure out where each track starts.
	for track := 0; track < maxTracks; track++ {
		jv3.trackStart[track][0] = -1
		jv3.trackStart[track][1] = -1
	}
	track := -1
	side := side(-1)
	for i := 0; i < jv3SectorsMax; i++ {
		id := &jv3.id[jv3.sortedId[i]]

		// See if it's a new track or side.
		if int(id.track) != track || id.side() != side {
			track = int(id.track)
			if track == jv3Free {
				// End of sectors.
				break
			}
			side = id.side()
			jv3.trackStart[track][side] = i
		}
	}

	jv3.sortedValid = true

	if diskSortDebug {
		for i := 0; i < jv3SectorsMax; i++ {
			index := jv3.sortedId[i]
			log.Printf("%04X -> %04X = %02X %02X %02X", i, index,
				jv3.id[index].track,
				jv3.id[index].sector,
				jv3.id[index].flags)
		}
	}
}
