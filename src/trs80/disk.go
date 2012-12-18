package main

import (
	"log"
	"io/ioutil"
)

// Data about the disk controller.
type fdc struct {
	// Registers.
	status byte
	track byte
	sector byte
	data byte

	// Disks themselves.
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
	panic("readDiskCommand")
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
	panic("writeDiskSelect")
	/*
  cpu.fdc.status &= ~TRSDISK_NOTRDY;
    cpu.fdc.curside = (data & TRSDISK3_SIDE) != 0;
    cpu.fdc.density = (data & TRSDISK3_MFM) != 0;
    if (data & TRSDISK3_WAIT) {
      // If there was an event pending, simulate waiting until it was due.
      if (trs_event_scheduled() != NULL &&
	  trs_event_scheduled() != trs_disk_lostdata) {
	z80_state.t_count = z80_state.sched;
	trs_do_event();
      }
    }
  switch (data & (TRSDISK_0|TRSDISK_1|TRSDISK_2|TRSDISK_3)) {
  case 0:
    cpu.fdc.status |= TRSDISK_NOTRDY;
    break;
  case TRSDISK_0:
    cpu.fdc.curdrive = 0;
    break;
  case TRSDISK_1:
    cpu.fdc.curdrive = 1;
    break;
  // If a drive was selected...
  if (!(cpu.fdc.status & TRSDISK_NOTRDY)) {
    DiskState *d = &disk[cpu.fdc.curdrive];

    // Retrigger emulated motor timeout 
    cpu.fdc.motor_timeout = z80_state.t_count +
      MOTOR_USEC * z80_state.clockMHz;
    trs_disk_motoroff_interrupt(0);

    // If a SIGUSR1 disk change is pending, accept it here 
    if (trs_disk_needchange) {
      trs_disk_change_all();
      trs_disk_needchange = 0;
    }

    // Update our knowledge of whether there is a real disk present 
    if (d->emutype == REAL) real_check_empty(d);
  }
  */
}
