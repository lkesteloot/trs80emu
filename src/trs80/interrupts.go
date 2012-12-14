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

func (cpu *cpu) setInterruptMask(irqMask byte) {
	cpu.irqMask = irqMask
}

func (cpu *cpu) handleIrq() {
	cpu.pushWord(cpu.pc)
	cpu.iff1 = false
	cpu.pc = 0x38
}
