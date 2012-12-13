package main

// IRQs
const (
	cassetteRiseIrqBit = 1 << iota
	cassetteFallIrqBit
	timerIrqBit
	ioBusIrqBit
	uartSendIrqBit
	uartReceiveIrqBit
	uartErrorIrqBit
)

// NMIs
const (
	resetNmiBit    = 0x20 << iota
	motorOffNmiBit // FDC
	intrqNmiBit    // FDC
)

// Compute whether IRQ handling is needed. XXX Can we remove this?
func (cpu *cpu) updateIrq() {
	cpu.irq = (cpu.irqLatch & cpu.irqMask) != 0
}

func (cpu *cpu) setInterruptMask(irqMask byte) {
	cpu.irqMask = irqMask
	cpu.updateIrq()
}

func (cpu *cpu) handleIrq() {
	cpu.pushWord(cpu.pc)
	cpu.iff1 = false
	cpu.pc = 0x38
}
