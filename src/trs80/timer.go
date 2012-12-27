// Copyright 2012 Lawrence Kesteloot

package main

const (
	timerHz     = 30
	timerCycles = cpuHz / timerHz
)

func (cpu *cpu) timerInterrupt(state bool) {
	if state {
		cpu.irqLatch |= timerIrqMask
	} else {
		cpu.irqLatch &^= timerIrqMask
	}
}

func (vm *vm) handleTimer() {
	if !disableTimer {
		vm.cpu.timerInterrupt(true)
		vm.cpu.diskMotorOffInterrupt(vm.checkDiskMotorOff())
	}
}
