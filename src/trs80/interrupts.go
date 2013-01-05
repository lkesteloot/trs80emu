// Copyright 2012 Lawrence Kesteloot

package main

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

func (cpu *cpu) setIrqMask(irqMask byte) {
	cpu.irqMask = irqMask
}

func (cpu *cpu) setNmiMask(nmiMask byte) {
	// Reset is always allowed:
	cpu.nmiMask = nmiMask | resetNmiMask
	cpu.updateNmiSeen()
}

func (cpu *cpu) updateNmiSeen() {
	if (cpu.nmiLatch & cpu.nmiMask) == 0 {
		cpu.nmiSeen = false
	}
}

func (vm *vm) handleIrq() {
	vm.pushWord(vm.cpu.pc)
	vm.cpu.iff1 = false
	vm.cpu.pc = 0x38
}

func (vm *vm) handleNmi() {
	vm.pushWord(vm.cpu.pc)
	vm.cpu.iff1 = false
	vm.cpu.pc = 0x66
}

func (cpu *cpu) resetButtonInterrupt(state bool) {
	if state {
		cpu.nmiLatch |= resetNmiMask
	} else {
		cpu.nmiLatch &^= resetNmiMask
	}
	cpu.updateNmiSeen()
}

func (cpu *cpu) diskMotorOffInterrupt(state bool) {
	if state {
		cpu.nmiLatch |= diskMotorOffNmiMask
	} else {
		cpu.nmiLatch &^= diskMotorOffNmiMask
	}
	cpu.updateNmiSeen()
}

func (cpu *cpu) diskIntrqInterrupt(state bool) {
	if state {
		cpu.nmiLatch |= diskIntrqNmiMask
	} else {
		cpu.nmiLatch &^= diskIntrqNmiMask
	}
	cpu.updateNmiSeen()
}

func (cpu *cpu) diskDrqInterrupt(state bool) {
	// No effect.
}

func (vm *vm) cassetteRiseInterrupt() {
	vm.cpu.irqLatch = (vm.cpu.irqLatch &^ cassetteRiseIrqMask) |
		(vm.cpu.irqMask & cassetteRiseIrqMask)
	vm.updateCassette()
}

func (vm *vm) cassetteFallInterrupt() {
	vm.cpu.irqLatch = (vm.cpu.irqLatch &^ cassetteFallIrqMask) |
		(vm.cpu.irqMask & cassetteFallIrqMask)
	vm.updateCassette()
}

func (vm *vm) cassetteClearInterrupt() {
	vm.cpu.irqLatch &^= cassetteIrqMasks
}

func (vm *vm) cassetteInterruptsEnabled() bool {
	return vm.cpu.irqMask&cassetteIrqMasks != 0
}
