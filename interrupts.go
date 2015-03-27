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
func (vm *vm) setIrqMask(irqMask byte) {
	vm.irqMask = irqMask
}

// Set the mask for non-maskable interrupts. (Yes.)
func (vm *vm) setNmiMask(nmiMask byte) {
	// Reset is always allowed:
	vm.nmiMask = nmiMask | resetNmiMask
	vm.updateNmiSeen()
}

// Reset whether we've seen this NMI interrupt if the mask and latch no longer overlap.
func (vm *vm) updateNmiSeen() {
	if (vm.nmiLatch & vm.nmiMask) == 0 {
		vm.nmiSeen = false
	}
}

// Set the state of the reset button interrupt.
func (vm *vm) resetButtonInterrupt(state bool) {
	if state {
		vm.nmiLatch |= resetNmiMask
	} else {
		vm.nmiLatch &^= resetNmiMask
	}
	vm.updateNmiSeen()
}

// Set the state of the disk motor off interrupt.
func (vm *vm) diskMotorOffInterrupt(state bool) {
	if state {
		vm.nmiLatch |= diskMotorOffNmiMask
	} else {
		vm.nmiLatch &^= diskMotorOffNmiMask
	}
	vm.updateNmiSeen()
}

// Set the state of the disk interrupt.
func (vm *vm) diskIntrqInterrupt(state bool) {
	if state {
		vm.nmiLatch |= diskIntrqNmiMask
	} else {
		vm.nmiLatch &^= diskIntrqNmiMask
	}
	vm.updateNmiSeen()
}

// Set the state of the disk interrupt.
func (vm *vm) diskDrqInterrupt(state bool) {
	// No effect.
}

// Saw a positive edge on cassette.
func (vm *vm) cassetteRiseInterrupt() {
	vm.cassetteRiseInterruptCount++
	vm.irqLatch = (vm.irqLatch &^ cassetteRiseIrqMask) |
		(vm.irqMask & cassetteRiseIrqMask)
}

// Saw a negative edge on cassette.
func (vm *vm) cassetteFallInterrupt() {
	vm.cassetteFallInterruptCount++
	vm.irqLatch = (vm.irqLatch &^ cassetteFallIrqMask) |
		(vm.irqMask & cassetteFallIrqMask)
}

// Reset cassette edge interrupts.
func (vm *vm) cassetteClearInterrupt() {
	vm.irqLatch &^= cassetteIrqMasks
}

// Check whether the software has enabled these interrupts.
func (vm *vm) cassetteInterruptsEnabled() bool {
	return vm.irqMask&cassetteIrqMasks != 0
}
