// Copyright 2012 Lawrence Kesteloot

package main

// The main code that emulates the Z80.

import (
	"log"
	"runtime"
	"time"
)

// Steps through one instruction.
func (vm *vm) step() {
	// Log PC for retroactive disassembly.
	if historicalPcCount > 0 {
		vm.historicalPcPtr = (vm.historicalPcPtr + 1) % historicalPcCount
		vm.historicalPc[vm.historicalPcPtr] = vm.z80.PC()
	}

	// Execute a single instruction.
	vm.z80.DoOpcode()
	vm.clock += 4 // XXX

	// Dispatch scheduled events.
	vm.events.dispatch(vm.clock)

	// Handle non-maskable interrupts.
	if (vm.nmiLatch&vm.nmiMask) != 0 && !vm.nmiSeen {
		if printDebug {
			log.Print("Non-maskable interrupt %02X", vm.nmiLatch)
		}
		vm.z80.NonMaskableInterrupt()
		vm.nmiSeen = true

		// Simulate the reset button being released.
		vm.resetButtonInterrupt(false)
	}

	// Handle interrupts.
	if (vm.irqLatch&vm.irqMask) != 0 {
		if printDebug {
			log.Print("Maskable interrupt %02X", vm.irqLatch)
		}
		vm.z80.Interrupt()
	}

	if vm.clock > vm.previousDumpClock+cpuHz {
		now := time.Now()
		if vm.previousDumpClock > 0 {
			elapsed := now.Sub(vm.previousDumpTime)
			computerTime := float64(vm.clock-vm.previousDumpClock) / float64(cpuHz)
			log.Printf("Computer time: %.1fs, elapsed: %.1fs, mult: %.1f, slept: %dms",
				computerTime, elapsed.Seconds(), computerTime/elapsed.Seconds(),
				vm.sleptSinceDump/time.Millisecond)
			vm.sleptSinceDump = 0
		}
		vm.previousDumpTime = now
		vm.previousDumpClock = vm.clock
	}

	// Slow down CPU if we're going too fast.
	if !*profiling && vm.clock > vm.previousAdjustClock+1000 {
		now := time.Now().UnixNano()
		elapsedReal := time.Duration(now - vm.startTime)
		elapsedFake := time.Duration(vm.clock * cpuPeriodNs)
		aheadNs := elapsedFake - elapsedReal
		if aheadNs > 0 {
			time.Sleep(aheadNs)
			vm.sleptSinceDump += aheadNs
		} else {
			// Yield periodically so that we can get messages from other
			// goroutines like the one sending us commands.
			runtime.Gosched()
		}
		vm.previousAdjustClock = vm.clock
	}

	// Set off a timer interrupt.
	if vm.clock > vm.previousTimerClock+timerCycles {
		vm.handleTimer()
		vm.previousTimerClock = vm.clock
	}

	// Update cassette state.
	vm.updateCassette()
}
