// Copyright 2012 Lawrence Kesteloot

package main

// The TRS-80 Model III has a 30 Hz timer that interrupts the CPU. This is used
// for things like blinking the cursor.

const (
	timerHz     = 30
	timerCycles = cpuHz / timerHz
)

// Set or reset the timer interrupt.
func (vm *vm) timerInterrupt(state bool) {
	if state {
		vm.irqLatch |= timerIrqMask
	} else {
		vm.irqLatch &^= timerIrqMask
	}
}

// What to do when the hardware timer goes off.
func (vm *vm) handleTimer() {
	if !disableTimer {
		vm.timerInterrupt(true)
		vm.diskMotorOffInterrupt(vm.checkDiskMotorOff())
	}
}
