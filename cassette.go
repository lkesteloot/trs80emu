// Copyright 2012 Lawrence Kesteloot

package main

const (
	// Threshold for 16-bit signed samples.
	cassetteThreshold = 5000
)

// State of the hardware. We don't support writing.
type cassetteState int

const (
	cassetteStateClose = cassetteState(iota)
	cassetteStateRead
	cassetteStateFail
)

// Value of wave in audio: negative, neutral (around zero), or positive.
type cassetteValue int

const (
	cassetteNeutral = cassetteValue(iota)
	cassettePositive
	cassetteNegative
)

// Internal state of the cassette controller.
type cassetteController struct {
	// Filename to read for cassette data. This should be a WAV file.
	filename string

	// Whether the motor is running.
	motorOn bool

	// Information about the cassette itself.
	cassette *wavFile

	// State machine.
	state cassetteState

	// Internal register state.
	value       cassetteValue
	lastNonZero cassetteValue
	flipFlop    bool

	// When we turned on the motor (started reading the file) and how many samples
	// we've read since then.
	motorOnClock uint64
	samplesRead  int
}

// Reset the controller to a known state.
func (vm *vm) resetCassette() {
	vm.setCassetteState(cassetteStateClose)
}

// Get a byte from the I/O port.
func (vm *vm) getCassetteByte() byte {
	cc := &vm.cc

	// If the motor's running, and we're reading a byte, then get into read mode.
	if cc.motorOn {
		vm.setCassetteState(cassetteStateRead)
	}

	// Clear any interrupt that may have triggered this read.
	vm.cassetteClearInterrupt()

	// Cassette owns bits 0 and 7.
	b := byte(0)
	if cc.flipFlop {
		b |= 0x80
	}
	if cc.lastNonZero == cassettePositive {
		b |= 0x01
	}
	return b
}

// Write to the cassette port. We don't support writing tapes, but this is used
// for 500-baud reading to trigger the next analysis of the tape.
func (vm *vm) putCassetteByte(b byte) {
	cc := &vm.cc

	if cc.motorOn {
		if cc.state == cassetteStateRead {
			vm.updateCassette()
			cc.flipFlop = false
		}
	}
}

// Kick off the reading process when doing 1500-baud reads.
func (vm *vm) kickOffCassette() {
	cc := &vm.cc

	if cc.motorOn && cc.state == cassetteStateClose && vm.cassetteInterruptsEnabled() {
		// Kick off the process.
		vm.cassetteRiseInterrupt()
		vm.cassetteFallInterrupt()
	}
}

// Turn the motor on or off.
func (vm *vm) setCassetteMotor(motorOn bool) {
	cc := &vm.cc

	if motorOn != cc.motorOn {
		if motorOn {
			cc.flipFlop = false
			cc.lastNonZero = cassetteNeutral

			// Wait one second, then kick off reading.
			vm.addEvent(eventKickOffCassette, func() { vm.kickOffCassette() }, cpuHz)
		} else {
			vm.setCassetteState(cassetteStateClose)
		}
		cc.motorOn = motorOn
		vm.updateCassetteMotorLight();
	}
}

// Read some of the cassette to see if we should be triggering a rise/fall interrupt.
func (vm *vm) updateCassette() {
	cc := &vm.cc

	if cc.motorOn && vm.setCassetteState(cassetteStateRead) >= 0 {
		// See how many samples we should have read by now.
		samplesToRead := int((vm.clock - cc.motorOnClock) *
			uint64(cc.cassette.samplesPerSecond) / cpuHz)

		// Catch up.
		for samplesToRead > cc.samplesRead {
			s, err := cc.cassette.readSample()
			if err != nil {
				panic(err)
			}
			cc.samplesRead++

			// Convert to state, where neutral is some noisy in-between state.
			value := cassetteNeutral
			if s > cassetteThreshold {
				value = cassettePositive
			} else if s < cassetteThreshold {
				value = cassetteNegative
			}

			// See if we've changed value.
			if value != cc.value {
				if value == cassettePositive {
					// Positive edge.
					cc.flipFlop = true
					vm.cassetteRiseInterrupt()
				} else if value == cassetteNegative {
					// Negative edge.
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

	// Once in error, everything will fail until we close.
	if oldState == cassetteStateFail && newState != cassetteStateClose {
		return -1
	}

	// Change things based on new state.
	switch newState {
	case cassetteStateRead:
		vm.openCassetteFile()
	}

	// Update state.
	vm.cc.state = newState
	return 0
}

// Open file, get metadata, and get read to read the tape.
func (vm *vm) openCassetteFile() {
	cc := &vm.cc

	cassette, err := openWav("cassettes/" + cc.filename)
	if err != nil {
		panic(err)
	}

	// Reset the clock.
	cc.cassette = cassette
	cc.motorOnClock = vm.clock
	cc.samplesRead = 0
}

// Update the status of the red light on the display.
func (vm *vm) updateCassetteMotorLight() {
	var motorOnInt int
	if vm.cc.motorOn {
		motorOnInt = 1
	} else {
		motorOnInt = 0
	}

	vm.vmUpdateCh <- vmUpdate{Cmd: "motor", Addr: -1, Data: motorOnInt}
}
