// Copyright 2012 Lawrence Kesteloot

package main

// Implementation of the TRS-80 Model III floppy disk controller. This file
// borrows heavily from the xtrs file trs_disk.c. We support both JV1 and JV3
// file formats, but JV1 is untested.
//
// JV1 is a Model I format that's just the sectors laid out end to end. There
// are 35 tracks, 10 sectors per track, and 256 bytes per sector.
//
// JV3 is a Model III format that allows variable-sized sectors. There are two
// blocks that specify where each sector is and how large it is. The first
// block is at the beginning of the file. It describes at most 2901 sectors,
// which follow the block. The next block is after that, followed by more
// sectors described by the second block. Each block is 2901 3-byte sector info
// structures. The first byte is the track number, the second is the sector
// number within that track, and the third is some flags that specify the size
// of the sector. See the jv3Sector structure.

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
)

const (
	// How many physical drives in the machine.
	driveCount = 4

	// How long the disk motor stays on after drive selected (in seconds).
	motorTimeAfterSelect = 2

	// Width of the index hole as a fraction of the circumference.
	diskHoleWidth = 0.01

	// Speed of disk.
	diskRpm             = 300
	clocksPerRevolution = cpuHz * 60 / diskRpm

	// Whether disks are write-protected. We make them all write-protected
	// since we don't implement writing to disk.
	writeProtection = true

	// Never have more than this many tracks.
	maxTracks = 255

	// JV1 info.
	jv1BytesPerSector  = 256
	jv1SectorsPerTrack = 10
	jv1DirectoryTrack  = 17

	// JV3 info.
	jv3MaxSides        = 2                      // Number of sides supported by this format.
	jv3IdStart         = 0                      // Where in file the IDs start.
	jv3SectorStart     = 34 * 256               // Start of sectors within file (end of IDs).
	jv3SectorsPerBlock = jv3SectorStart / 3     // Number of jv3Sector structs per info block.
	jv3SectorsMax      = 2 * jv3SectorsPerBlock // There are two info blocks maximum.
)

// Type I status bits.
const (
	diskBusy     = 1 << iota // Whether the disk is actively doing work.
	diskIndex                // The head is currently over the index hole.
	diskTrkZero              // Head is on track 0.
	diskCrcErr               // CRC error.
	diskSeekErr              // Seek error.
	diskHeadEngd             // Head engaged.
	diskWritePrt             // Write-protected.
	diskNotRdy               // Disk not ready (motor not running).
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
	diskWait // Controller should block OUT until operation is done.
	diskMfm  // Double density.

	diskDriveMask = diskDrive0 | diskDrive1 | diskDrive2 | diskDrive3
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

	jv3Free = 0xff // In track/sector fields
)

// Disk emulation types. After loading a disk file we detect what type of disk
// it is.
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
	motorOn        bool
	motorTimeout   uint64
	lastReadAdr    int // Id index found by last readadr.

	// Disks themselves.
	disks [driveCount]disk
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

// Each block of a JV3 file has jv3SectorsPerBlock of these.
type jv3Sector struct {
	track, sector, flags byte
}

// Sets all elements to jv3Free, marking the sector as free.
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

// Returns which side this sector is on.
func (id *jv3Sector) side() (side side) {
	side.setFromBoolean(id.flags&jv3Side != 0)
	return
}

// Returns whether this sector is double density (true) or single density (false).
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

	// Which bit to flip (invert) in the size code to get it in order.
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

// Returns a side from a boolean, where true is back and false is front.
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

// Loads the file from filename into drive.
func (vm *vm) loadDisk(drive int, filename string) error {
	return vm.fdc.disks[drive].load(filename)
}

// Loads the file from filename into this disk.
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
		disk.loadJv3Data()
	default:
		log.Fatalf("Don't know format of %d-byte disk", len(disk.data))
	}
}

// Loads JV3-specific data from the file and creates the in-memory data structures.
func (disk *disk) loadJv3Data() {
	// Mark the whole id array as free.
	for i := 0; i < len(disk.jv3.id); i++ {
		disk.jv3.id[i].makeFree()
	}

	// How many info blocks are on the disk. This is incremented in loadJv3Block().
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

	// Sort the IDs for fast lookup.
	disk.jv3.sortIds()
}

// Load one of the blocks of sector infos in JV3 disks. Return the byte
// offset of the end of the sectors described by this block.
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

		// Compute offsets of each sector.
		offset := blockStart + jv3SectorStart
		for i := 0; i < jv3SectorsPerBlock; i++ {
			disk.jv3.offset[idStart+i] = offset
			offset += disk.jv3.id[idStart+i].getSize()
		}

		// Where next block would begin.
		return offset
	}

	// Doesn't matter here, this would only happen for the second block, and
	// we don't care about the offset for that one.
	return 0
}

// Initialize the FDC.
func (vm *vm) diskInit(powerOn bool) {
	fdc := &vm.fdc

	if diskDebug {
		log.Printf("diskInit(%v)", powerOn)
	}

	// Registers.
	fdc.status = diskNotRdy | diskTrkZero
	fdc.track = 0
	fdc.sector = 0
	fdc.data = 0

	// Various state.
	fdc.currentCommand = diskRestore
	fdc.byteCount = 0
	fdc.side = 0
	fdc.doubleDensity = false
	fdc.currentDrive = 0
	fdc.motorOn = false
	fdc.motorTimeout = 0
	vm.fdc.lastReadAdr = -1

	for i := 0; i < len(fdc.disks); i++ {
		fdc.disks[i].init(powerOn)
	}

	// Cancel any pending disk event.
	vm.events.cancelEvents(eventDisk)
}

// Initialize the drive.
func (disk *disk) init(powerOn bool) {
	if powerOn {
		disk.physicalTrack = 0
	}
}

// Event used for delayed command completion.  Clears BUSY,
// sets any additional bits specified, and generates a command
// completion interrupt.
func (vm *vm) diskDone(bits byte) {
	if diskDebug {
		log.Printf("diskDone(%02X)", bits)
	}

	vm.fdc.status &^= diskBusy
	vm.fdc.status |= bits
	vm.diskIntrqInterrupt(true)
}

// Event to abort the last command with LOSTDATA if it is
// still in progress.
func (vm *vm) diskLostData(cmd byte) {
	if diskDebug {
		log.Printf("diskLostData(%02X)", cmd)
	}

	if vm.fdc.currentCommand == cmd {
		vm.fdc.status &^= diskBusy
		vm.fdc.status |= diskLostData
		vm.fdc.byteCount = 0
		vm.diskIntrqInterrupt(true)
	}
}

// Event used as a delayed command start. Sets DRQ, generates a DRQ interrupt,
// sets any additional bits specified, and schedules a diskLostData() event.
func (vm *vm) diskFirstDrq(bits byte) {
	if diskDebug {
		log.Printf("diskFirstDrq(%v)", bits)
	}

	vm.fdc.status |= diskDrq | bits
	vm.diskDrqInterrupt(true)
	// Evaluate this now, not when the callback is run.
	currentCommand := vm.fdc.currentCommand
	// If we've not finished our work within half a second, trigger a lost data
	// interrupt.
	vm.addEvent(eventDiskLostData, func() { vm.diskLostData(currentCommand) }, cpuHz/2)
}

// If we've not used this drive within the timeout period, shut off the motor. Returns
// whether we shut the motor off.
func (vm *vm) checkDiskMotorOff() bool {
	stopped := vm.clock > vm.fdc.motorTimeout
	if stopped {
		vm.setDiskMotor(false)
		vm.fdc.status |= diskNotRdy

		// See if we were in the middle of doing something.
		if isReadWriteCommand(vm.fdc.currentCommand) && (vm.fdc.status&diskDrq) != 0 {
			// Also end the command and set Lost Data for good measure
			vm.fdc.status = (vm.fdc.status | diskLostData) & ^byte(diskBusy|diskDrq)
			vm.fdc.byteCount = 0
		}
	}

	return stopped
}

// Turns the motor on or off. Updates the UI.
func (vm *vm) setDiskMotor(motorOn bool) {
	if vm.fdc.motorOn != motorOn {
		vm.fdc.motorOn = motorOn
		vm.updateDiskMotorLights()
	}
}

// Return a value in [0,1) indicating how far we've rotated
// from the leading edge of the index hole. For the first diskHoleWidth we're
// on the hole itself.
func (vm *vm) diskAngle() float32 {
	// Use simulated time.
	return float32(vm.clock%clocksPerRevolution) / float32(clocksPerRevolution)
}

// Whether the current disk command is read or write (as opposed to seek, etc.).
func isReadWriteCommand(cmd byte) bool {
	cmdType := commandType(cmd)
	return cmdType == 2 || cmdType == 3
}

// Returns the type of command we're currently executing. See the constants
// starting with diskCommandMask for details.
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

// If we're doing a non-read/write command, update the status with the state
// of the disk, track, and head position.
func (vm *vm) updateDiskStatus() {
	disk := &vm.fdc.disks[vm.fdc.currentDrive]

	if isReadWriteCommand(vm.fdc.currentCommand) {
		// Don't modify status.
		return
	}

	if disk.data == nil {
		vm.fdc.status |= diskIndex
	} else {
		// See if we're over the index hole.
		if vm.diskAngle() < diskHoleWidth {
			vm.fdc.status |= diskIndex
		} else {
			vm.fdc.status &^= diskIndex
		}

		// We're always write-protected.
		if writeProtection {
			vm.fdc.status |= diskWritePrt
		} else {
			vm.fdc.status &^= diskWritePrt
		}
	}

	// See if we're on track 0, which for some reason has a special bit.
	if disk.physicalTrack == 0 {
		vm.fdc.status |= diskTrkZero
	} else {
		vm.fdc.status &^= diskTrkZero
	}

	// RDY and HLT inputs are wired together on TRS-80 I/III/4/4P.
	if vm.fdc.status&diskNotRdy != 0 {
		vm.fdc.status &^= diskHeadEngd
	} else {
		vm.fdc.status |= diskHeadEngd
	}
}

// Return the disk status from the I/O port.
func (vm *vm) readDiskStatus() byte {
	// If no disk was loaded into drive 0, just pretend that we don't
	// have a disk system. Otherwise we have to hold down Break while
	// booting (to get to cassette BASIC) and that's annoying.
	if driveCount == 0 || vm.fdc.disks[0].data == nil {
		return 0xFF
	}

	vm.updateDiskStatus()

	// Turn off motor if it's running and been too long.
	if vm.fdc.status&diskNotRdy == 0 {
		if vm.clock > vm.fdc.motorTimeout {
			vm.setDiskMotor(false)
			vm.fdc.status |= diskNotRdy
		}
	}

	// Clear interrupt.
	vm.diskIntrqInterrupt(false)

	if diskDebug {
		log.Printf("readDiskStatus() = %02X", vm.fdc.status)
	}

	return vm.fdc.status
}

// Read the track register.
func (vm *vm) readDiskTrack() byte {
	if diskDebug {
		log.Printf("readDiskTrack() = %02X", vm.fdc.track)
	}

	return vm.fdc.track
}

// Read the sector register.
func (vm *vm) readDiskSector() byte {
	if diskDebug {
		log.Printf("readDiskSector() = %02X", vm.fdc.sector)
	}

	return vm.fdc.sector
}

// Read a byte of data from the sector.
func (vm *vm) readDiskData() byte {
	disk := &vm.fdc.disks[vm.fdc.currentDrive]

	// The read command can do various things depending on the specific current command,
	// but we only support reading from the diskette.
	switch vm.fdc.currentCommand & diskCommandMask {
	case diskRead:
		// Keep reading from the buffer.
		if vm.fdc.byteCount > 0 && (vm.fdc.status&diskDrq) != 0 {
			var c byte
			if disk.dataOffset >= len(disk.data) {
				c = 0xE5
				if disk.emulationType == emuJv3 {
					vm.fdc.status &^= diskRecType
					vm.fdc.status |= disk1791FB
				}
			} else {
				c = disk.data[disk.dataOffset]
				disk.dataOffset++
			}
			vm.fdc.data = c
			vm.fdc.byteCount--
			if vm.fdc.byteCount <= 0 {
				vm.fdc.byteCount = 0
				vm.fdc.status &^= diskDrq
				vm.diskDrqInterrupt(false)
				vm.events.cancelEvents(eventDiskLostData)
				vm.addEvent(eventDiskDone, func() { vm.diskDone(0) }, 64)
			}
		}

	default:
		// Might be okay, not sure.
		panic("Unhandled case in readDiskData()")
	}

	if diskDebug {
		log.Printf("readDiskData() = %02X (%d left)", vm.fdc.data, vm.fdc.byteCount)
	}

	return vm.fdc.data
}

// Set the current command.
func (vm *vm) writeDiskCommand(cmd byte) {
	disk := &vm.fdc.disks[vm.fdc.currentDrive]

	if diskDebug {
		log.Printf("writeDiskCommand(%02X)", cmd)
	}

	// Cancel "lost data" event.
	vm.events.cancelEvents(eventDiskLostData)

	vm.diskIntrqInterrupt(false)
	vm.fdc.byteCount = 0
	vm.fdc.currentCommand = cmd

	// Kick off anything that's based on the command.
	switch cmd & diskCommandMask {
	case diskRestore:
		vm.fdc.lastReadAdr = -1
		disk.physicalTrack = 0
		vm.fdc.track = 0
		vm.fdc.status = diskTrkZero | diskBusy
		if cmd&diskVMask != 0 {
			vm.diskVerify()
		}
		vm.addEvent(eventDiskDone, func() { vm.diskDone(0) }, 2000)
	case diskSeek:
		vm.fdc.lastReadAdr = -1
		disk.physicalTrack += vm.fdc.data - vm.fdc.track
		vm.fdc.track = vm.fdc.data
		if disk.physicalTrack <= 0 {
			// vm.fdc.track too?
			disk.physicalTrack = 0
			vm.fdc.status = diskTrkZero | diskBusy
		} else {
			vm.fdc.status = diskBusy
		}
		// Should this set lastDirection?
		if cmd&diskVMask != 0 {
			vm.diskVerify()
		}
		vm.addEvent(eventDiskDone, func() { vm.diskDone(0) }, 2000)
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
		// Read the sector. The bytes will be read later.
		vm.fdc.lastReadAdr = -1
		vm.fdc.status = 0
		goalSide := side(-1)
		if cmd&diskCMask != 0 {
			goalSide.setFromBoolean((cmd & diskBMask) != 0)
		}

		// Look for the sector in the file.
		sectorIndex := vm.searchSector(int(vm.fdc.sector), goalSide)
		if sectorIndex == -1 {
			vm.fdc.status |= diskBusy
			vm.addEvent(eventDiskDone, func() { vm.diskDone(0) }, 512)
			log.Printf("Didn't find sector %02X on track %02X",
				vm.fdc.sector, disk.physicalTrack)
		} else {
			var newStatus byte = 0
			switch disk.emulationType {
			case emuJv1:
				if disk.physicalTrack == jv1DirectoryTrack {
					newStatus = disk1791F8
				}
				vm.fdc.byteCount = jv1BytesPerSector
				disk.dataOffset = disk.getDataOffset(sectorIndex)
			case emuJv3:
				if !vm.fdc.doubleDensity {
					// Single density 179x.
					switch disk.jv3.id[sectorIndex].flags & jv3Dam {
					case jv3DamSdFB:
						newStatus = disk1791FB
						break
					case jv3DamSdFA:
						newStatus = disk1791F8
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
				vm.fdc.byteCount = disk.jv3.id[sectorIndex].getSize()
				disk.dataOffset = disk.getDataOffset(sectorIndex)
			default:
				panic("Unhandled case in diskRead")
			}
			vm.fdc.status |= diskBusy
			vm.addEvent(eventDiskFirstDrq, func() { vm.diskFirstDrq(newStatus) }, 64)
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
		vm.events.cancelEvents(eventDisk)
		vm.fdc.status = 0
		vm.updateDiskStatus()
		if (cmd & 0x07) != 0 {
			panic("Conditional interrupt features not implemented")
		} else if (cmd & 0x08) != 0 {
			// Immediate interrupt.
			vm.diskIntrqInterrupt(true)
		} else {
			vm.diskIntrqInterrupt(false)
		}
	default:
		panic(fmt.Sprintf("Unknown disk command %02X", cmd))
	}
}

// Set the track register for later reads.
func (vm *vm) writeDiskTrack(value byte) {
	if diskDebug {
		log.Printf("writeDiskTrack(%02X)", value)
	}

	vm.fdc.track = value
}

// Set the sector register for later reads.
func (vm *vm) writeDiskSector(value byte) {
	if diskDebug {
		log.Printf("writeDiskSector(%02X)", value)
	}

	vm.fdc.sector = value
}

// Write to the data register. We don't support writing to the diskette,
// but the register is used by other commands, such as seeking.
func (vm *vm) writeDiskData(value byte) {
	if diskDebug {
		log.Printf("writeDiskData(%02X)", value)
	}

	switch vm.fdc.currentCommand & diskCommandMask {
	case diskWrite:
		panic("diskWrite")
	case diskWriteTrk:
		panic("diskWriteTrk")
	default:
		// No action, just fall through and store data.
		break
	}

	vm.fdc.data = value
}

// Select a disk drive.
func (vm *vm) writeDiskSelect(value byte) {
	if diskDebug {
		log.Printf("writeDiskSelect(%02X)", value)
	}

	vm.fdc.status &^= diskNotRdy
	vm.fdc.side.setFromBoolean((value & diskSide) != 0)
	vm.fdc.doubleDensity = (value & diskMfm) != 0
	if value&diskWait != 0 {
		// If there was an event pending, simulate waiting until it was due.
		event := vm.events.getFirstEvent(eventDisk &^ eventDiskLostData)
		if event != nil {
			if diskDebug {
				log.Printf("Advancing clock from %d to %d", vm.clock, event.clock)
			}
			// This puts the clock ahead immediately, but the main loop of the emulator
			// will then sleep to make the real-time correct.
			vm.clock = event.clock
			vm.events.dispatch(vm.clock)
		}
	}

	// Which drive is being enabled?
	switch value & diskDriveMask {
	case 0:
		vm.fdc.status |= diskNotRdy
	case diskDrive0:
		vm.fdc.currentDrive = 0
	case diskDrive1:
		vm.fdc.currentDrive = 1
	case diskDrive2:
		vm.fdc.currentDrive = 2
	case diskDrive3:
		vm.fdc.currentDrive = 3
	default:
		panic("Disk not handled")
	}
	vm.updateDiskMotorLights()

	// If a drive was selected, turn on its motor.
	if vm.fdc.status&diskNotRdy == 0 {
		vm.setDiskMotor(true)
		// XXX Could replace this with an event.
		vm.fdc.motorTimeout = vm.clock + motorTimeAfterSelect*cpuHz
		vm.diskMotorOffInterrupt(false)
	}
}

// Search for a sector on the current physical track.  Return its index within
// the emulated disk's array of sectors.  Set status and return -1 if there is
// no such sector.  If sector == -1, return the first sector found if any.  If
// side == 0 or 1, perform side compare against sector ID; if -1, don't.
func (vm *vm) searchSector(sector int, side side) int {
	disk := &vm.fdc.disks[vm.fdc.currentDrive]

	if disk.data == nil {
		vm.fdc.status |= diskNotFound
		return -1
	}

	switch disk.emulationType {
	case emuNone:
		vm.fdc.status |= diskNotFound
		return -1
	case emuJv1:
		// Check for error.
		if disk.physicalTrack < 0 ||
			disk.physicalTrack >= maxTracks ||
			vm.fdc.side == 1 ||
			side == 1 ||
			sector >= jv1SectorsPerTrack ||
			disk.data == nil ||
			disk.physicalTrack != vm.fdc.track {

			vm.fdc.status |= diskNotFound
			return -1
		}

		if sector < 0 {
			sector = 0
		}

		// All sectors are the same size, so just use a formula.
		return jv1SectorsPerTrack*int(disk.physicalTrack) + sector
	case emuJv3:
		// Check for error.
		if disk.physicalTrack < 0 ||
			disk.physicalTrack >= maxTracks ||
			vm.fdc.side >= jv3MaxSides ||
			(side != -1 && side != vm.fdc.side) ||
			disk.physicalTrack != vm.fdc.track ||
			disk.data == nil {

			vm.fdc.status |= diskNotFound
			return -1
		}
		if !disk.jv3.sortedValid {
			disk.jv3.sortIds()
		}

		// Look up in the sorted array.
		i := disk.jv3.trackStart[disk.physicalTrack][vm.fdc.side]
		if i != -1 {
			for {
				id := disk.jv3.sortedId[i]
				sid := &disk.jv3.id[id]
				if sid.track != disk.physicalTrack ||
					sid.side() != vm.fdc.side {

					break
				}
				if (sector == -1 || int(sid.sector) == sector) &&
					sid.doubleDensity() == vm.fdc.doubleDensity {

					return id
				}
				i++
			}
		}
		vm.fdc.status |= diskNotFound
		return -1
	}

	panic("Unhandled case in searchSector()")
}

// Get the byte offset of the given sector.
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
func (vm *vm) diskVerify() {
	disk := &vm.fdc.disks[vm.fdc.currentDrive]

	switch disk.emulationType {
	case emuNone:
		vm.fdc.status |= diskNotFound
	case emuJv1:
		if disk.data == nil {
			vm.fdc.status |= diskNotFound
		}
		if vm.fdc.doubleDensity {
			vm.fdc.status |= diskNotFound
		} else if vm.fdc.track != disk.physicalTrack {
			vm.fdc.status |= diskSeekErr
		}
	case emuJv3:
		// diskSeekErr == diskNotFound
		vm.searchSector(-1, -1)
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

// (Re-)create the sortedId data structure.
func (jv3 *jv3) sortIds() {
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

// Update the status of the red lights on the display.
func (vm *vm) updateDiskMotorLights() {
	if vm.vmUpdateCh != nil {
		for drive := 0; drive < driveCount; drive++ {
			var motorOnInt int
			if vm.fdc.motorOn && vm.fdc.currentDrive == drive {
				motorOnInt = 1
			} else {
				motorOnInt = 0
			}

			vm.vmUpdateCh <- vmUpdate{Cmd: "motor", Addr: drive, Data: motorOnInt}
		}
	}
}
