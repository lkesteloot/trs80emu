// Copyright 2012 Lawrence Kesteloot

package main

// The TRS-80 Model III has a 30 Hz timer that interrupts the CPU. This is used
// for things like blinking the cursor.

const (
	timerHz     = 30
	timerCycles = cpuHz / timerHz
)

// Set or reset the timer interrupt.
func (cpu *cpu) timerInterrupt(state bool) {
	if state {
		cpu.irqLatch |= timerIrqMask
	} else {
		cpu.irqLatch &^= timerIrqMask
	}
}

// What to do when the hardware timer goes off.
func (vm *vm) handleTimer() {
	if !disableTimer {
		vm.cpu.timerInterrupt(true)
		vm.cpu.diskMotorOffInterrupt(vm.checkDiskMotorOff())
	}
}
