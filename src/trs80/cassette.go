// Copyright 2012 Lawrence Kesteloot

package main

import (
	"log"
)

type cassetteState int
const (
	cassetteStateClose = cassetteState(iota)
	cassetteStateRead
	cassetteStateFail
)

type cassetteSpeed int
const (
	cassette500 = cassetteSpeed(500)
	cassette1500 = cassetteSpeed(1500)
)

type cassetteController struct {
	// Whether the motor is running.
	motorOn bool

	// State machine.
	state cassetteState

	// Speed that we're reading at.
	speed cassetteSpeed

	// Byte offset within the input file.
	position int

	// Bogus
	flipFlop, lastNonZero byte
	transition uint64
}

func (vm *vm) resetCassette() {
	vm.setCassetteState(cassetteStateClose)
}

func (vm *vm) getCassetteByte() byte {
	cc := &vm.cc

	vm.cassetteClearInterrupt()
	vm.updateCassette()

	value := cc.flipFlop
	if cc.lastNonZero == 1 {
		value |= 1
	}
	log.Printf("getCassetteByte() = %02X", value)
	return value
}

func (vm *vm) putCassetteByte(value byte) {
	// Ignore.
	log.Printf("Sending %02X to cassette", value)
}

func (vm *vm) kickOffCassette() {
	cc := &vm.cc

	log.Printf("kickOffCassette()")
	if cc.motorOn && cc.state == cassetteStateClose && vm.cassetteInterruptsEnabled() {
		// If we're here, then it's a 1500 baud read.
		cc.speed = cassette1500
		cc.transition = vm.clock
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
			cc.flipFlop = 0
			cc.lastNonZero = 0

			// Wait one second, then kick off reading.
			vm.addEvent(eventKickOffCassette, func () { vm.kickOffCassette() }, cpuHz)
		} else {
			vm.setCassetteState(cassetteStateClose)
		}
		cc.motorOn = motorOn
	}
}

func (vm *vm) updateCassette() {
	cc := &vm.cc

	if cc.motorOn && vm.setCassetteState(cassetteStateRead) >= 0 {
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

	switch newState {
	case cassetteStateRead:
		vm.cc.position = 0
		// XXX Open file, set audio rate, seek to right position.
	}

	// Update state.
	vm.cc.state = newState
	return 0
}
