package main

import (
	"time"
)

const timerHz = 30

func getTimerCh() <-chan time.Time {
	return time.Tick(time.Second/timerHz)
}

func (cpu *cpu) timerInterrupt(state bool) {
    if state {
		cpu.irqLatch |= timerIrqBit
	} else {
		cpu.irqLatch &^= timerIrqBit
	}

	cpu.updateIrq()
}
