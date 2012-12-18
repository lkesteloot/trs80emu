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
)

// NMIs
const (
	resetNmiMask    = 0x20 << iota
	motorOffNmiMask // FDC
	intrqNmiMask    // FDC
)

func (cpu *cpu) setIrqMask(irqMask byte) {
	cpu.irqMask = irqMask
}

func (cpu *cpu) setNmiMask(nmiMask byte) {
	// Always allowed:
	cpu.nmiMask = nmiMask | resetNmiMask
	cpu.updateNmiSeen()
}

func (cpu *cpu) updateNmiSeen() {
	if (cpu.nmiLatch&cpu.nmiMask) == 0 {
		cpu.nmiSeen = false
	}
}

func (cpu *cpu) handleIrq() {
	cpu.pushWord(cpu.pc)
	cpu.iff1 = false
	cpu.pc = 0x38
}

func (cpu *cpu) handleNmi() {
	cpu.pushWord(cpu.pc)
	cpu.iff1 = false
	cpu.pc = 0x66
}

func (cpu *cpu) resetButtonInterrupt(state bool) {
    if state {
		cpu.nmiLatch |= resetNmiMask
	} else {
		cpu.nmiLatch &^= resetNmiMask
	}
	cpu.updateNmiSeen()
}
