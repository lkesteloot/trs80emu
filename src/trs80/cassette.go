// Copyright 2012 Lawrence Kesteloot

package main

import (
	"log"
)

const (
	// Threshold for 16-bit samples.
	cassetteThreshold = 5000
)

type cassetteState int

const (
	cassetteStateClose = cassetteState(iota)
	cassetteStateRead
	cassetteStateFail
)

type cassetteSpeed int

const (
	cassette500  = cassetteSpeed(500)
	cassette1500 = cassetteSpeed(1500)
)

type cassetteValue int

const (
	cassetteNeutral = cassetteValue(iota)
	cassettePositive
	cassetteNegative
)

type cassetteController struct {
	// Whether the motor is running.
	motorOn bool

	// Information about the cassette itself.
	cassette *wavFile

	// State machine.
	state cassetteState

	// Speed that we're reading at.
	speed cassetteSpeed

	// Byte offset within the input file.
	position int

	// XXX Bogus
	value       cassetteValue
	next        cassetteValue
	flipFlop    bool
	lastNonZero cassetteValue
	transition  uint64

	// When we turned on the motor (started reading the file) and how many samples
	// we've read since then.
	motorOnClock uint64
	samplesRead int
}

func (vm *vm) resetCassette() {
	vm.setCassetteState(cassetteStateClose)
}

func (vm *vm) getCassetteByte() byte {
	cc := &vm.cc

	/// log.Printf("getCassetteByte() start")

	if cc.motorOn {
		vm.setCassetteState(cassetteStateRead)
	}

	vm.cassetteClearInterrupt()

	b := byte(0)
	if cc.flipFlop {
		b |= 0x80
	}
	if cc.lastNonZero == cassettePositive {
		b |= 0x01
	}
	/// log.Printf("getCassetteByte() = %02X", b)
	return b
}

func (vm *vm) putCassetteByte(b byte) {
	// Ignore.
	log.Printf("Sending %02X to cassette", b)
}

func (vm *vm) kickOffCassette() {
	cc := &vm.cc

	log.Printf("kickOffCassette()")
	if cc.motorOn && cc.state == cassetteStateClose && vm.cassetteInterruptsEnabled() {
		// If we're here, then it's a 1500 baud read.
		cc.speed = cassette1500
		cc.transition = vm.clock

		// Kick off the process.
		vm.cassetteRiseInterrupt()
		vm.cassetteFallInterrupt()
	}
}

func (vm *vm) setCassetteMotor(motorOn bool) {
	cc := &vm.cc

	if motorOn != cc.motorOn {
		log.Printf("setCassetteMotor(%v)", motorOn)
		if motorOn {
			cc.transition = vm.clock
			cc.flipFlop = false
			cc.lastNonZero = cassetteNeutral

			// Wait one second, then kick off reading.
			vm.addEvent(eventKickOffCassette, func() { vm.kickOffCassette() }, cpuHz)
		} else {
			vm.setCassetteState(cassetteStateClose)
		}
		cc.motorOn = motorOn
	}
}

// Read some of the cassette to see if we should be triggering a rise/fall interrupt.
func (vm *vm) updateCassette() {
	cc := &vm.cc

	if cc.motorOn && vm.setCassetteState(cassetteStateRead) >= 0 {
		// See how many samples we should have read by now.
		samplesToRead := int((vm.clock - cc.motorOnClock)*
			uint64(cc.cassette.samplesPerSecond)/cpuHz)

		// Catch up.
		for samplesToRead > cc.samplesRead {
			s, err := cc.cassette.readSample()
			if err != nil {
				// XXX Probably EOF.
				panic(err)
			}
			cc.samplesRead++

			// Convert to -1, 0, 1, where 0 is some noisy in-between state.
			value := cassetteNeutral
			if s > cassetteThreshold {
				value = cassettePositive
			} else if s < cassetteThreshold {
				value = cassetteNegative
			}

			if value != cc.value {
				if value == cassettePositive {
					cc.flipFlop = true
					vm.cassetteRiseInterrupt()
				} else if value == cassetteNegative {
					cc.flipFlop = true
					vm.cassetteFallInterrupt()
				}

				cc.value = value
				if value != cassetteNeutral {
					cc.lastNonZero = value
				}
			}
		}
	}
}

// Returns 0 if the state was changed, 1 if it wasn't, and -1 on error.
func (vm *vm) setCassetteState(newState cassetteState) int {
	oldState := vm.cc.state

	// See if we're changing anything.
	if oldState == newState {
		return 1
	}
	log.Printf("setCassetteState(%d)", newState)

	// Once in error, everything will fail until we close.
	if oldState == cassetteStateFail && newState != cassetteStateClose {
		return -1
	}

	// Change things based on new state.
	switch newState {
	case cassetteStateRead:
		vm.cc.position = 0
		vm.openCassetteFile()
	}

	// Update state.
	vm.cc.state = newState
	return 0
}

// Open file, set audio rate, seek to right position.
func (vm *vm) openCassetteFile() {
	cc := &vm.cc

	filename := "cassettes/tron1.wav"
	cassette, err := openWav(filename)
	if err != nil {
		panic(err)
	}

	// Reset the clock.
	cc.cassette = cassette
	cc.motorOnClock = vm.clock
	cc.samplesRead = 0
}
