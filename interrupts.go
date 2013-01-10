// Copyright 2012 Lawrence Kesteloot

package main

// Handle interrupts.

// IRQs
const (
	cassetteRiseIrqMask = 1 << iota
	cassetteFallIrqMask
	timerIrqMask
	ioBusIrqMask
	uartSendIrqMask
	uartReceiveIrqMask
	uartErrorIrqMask

	cassetteIrqMasks = cassetteRiseIrqMask | cassetteFallIrqMask
)

// NMIs
const (
	resetNmiMask = 0x20 << iota
	diskMotorOffNmiMask
	diskIntrqNmiMask
)

// Set the mask for IRQ (regular) interrupts.
func (cpu *cpu) setIrqMask(irqMask byte) {
	cpu.irqMask = irqMask
}

// Set the mask for non-maskable interrupts. (Yes.)
func (cpu *cpu) setNmiMask(nmiMask byte) {
	// Reset is always allowed:
	cpu.nmiMask = nmiMask | resetNmiMask
	cpu.updateNmiSeen()
}

// Reset whether we've seen this NMI interrupt if the mask and latch no longer overlap.
func (cpu *cpu) updateNmiSeen() {
	if (cpu.nmiLatch & cpu.nmiMask) == 0 {
		cpu.nmiSeen = false
	}
}

// Jump to the IRQ handler.
func (vm *vm) handleIrq() {
	vm.pushWord(vm.cpu.pc)
	vm.cpu.iff1 = false
	vm.cpu.pc = 0x38
}

// Jump to the NMI handler.
func (vm *vm) handleNmi() {
	vm.pushWord(vm.cpu.pc)
	vm.cpu.iff1 = false
	vm.cpu.pc = 0x66
}

// Set the state of the reset button interrupt.
func (cpu *cpu) resetButtonInterrupt(state bool) {
	if state {
		cpu.nmiLatch |= resetNmiMask
	} else {
		cpu.nmiLatch &^= resetNmiMask
	}
	cpu.updateNmiSeen()
}

// Set the state of the disk motor off interrupt.
func (cpu *cpu) diskMotorOffInterrupt(state bool) {
	if state {
		cpu.nmiLatch |= diskMotorOffNmiMask
	} else {
		cpu.nmiLatch &^= diskMotorOffNmiMask
	}
	cpu.updateNmiSeen()
}

// Set the state of the disk interrupt.
func (cpu *cpu) diskIntrqInterrupt(state bool) {
	if state {
		cpu.nmiLatch |= diskIntrqNmiMask
	} else {
		cpu.nmiLatch &^= diskIntrqNmiMask
	}
	cpu.updateNmiSeen()
}

// Set the state of the disk interrupt.
func (cpu *cpu) diskDrqInterrupt(state bool) {
	// No effect.
}

// Saw a positive edge on cassette.
func (vm *vm) cassetteRiseInterrupt() {
	vm.cpu.irqLatch = (vm.cpu.irqLatch &^ cassetteRiseIrqMask) |
		(vm.cpu.irqMask & cassetteRiseIrqMask)
}

// Saw a negative edge on cassette.
func (vm *vm) cassetteFallInterrupt() {
	vm.cpu.irqLatch = (vm.cpu.irqLatch &^ cassetteFallIrqMask) |
		(vm.cpu.irqMask & cassetteFallIrqMask)
}

// Reset cassette edge interrupts.
func (vm *vm) cassetteClearInterrupt() {
	vm.cpu.irqLatch &^= cassetteIrqMasks
}

// Check whether the software has enabled these interrupts.
func (vm *vm) cassetteInterruptsEnabled() bool {
	return vm.cpu.irqMask&cassetteIrqMasks != 0
}
