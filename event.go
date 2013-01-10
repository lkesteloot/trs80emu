// Copyright 2012 Lawrence Kesteloot

package main

// Infrastructure to trigger events in the future. This is usually for hardware events.

import (
	"log"
)

const (
	eventDiskDone = eventType(1 << iota)
	eventDiskLostData
	eventDiskFirstDrq
	eventKickOffCassette

	// Masks for multiple events.
	eventDisk = eventDiskDone | eventDiskLostData | eventDiskFirstDrq
)

type eventType uint
type eventCallback func()

// A single scheduled event.
type event struct {
	eventType eventType
	callback  eventCallback
	clock     uint64
	next      *event
}

// All scheduled events.
type events struct {
	head *event
}

// Queue up an event to happen at clock, using a delta clock relative to the
// current time.
func (vm *vm) addEvent(eventType eventType, callback eventCallback, deltaClock uint64) {
	vm.events.add(eventType, callback, vm.clock+deltaClock)
}

// Queue up an event to happen at clock.
func (events *events) add(eventType eventType, callback eventCallback, clock uint64) {
	event := &event{eventType, callback, clock, nil}

	// Insert into list sorted by clock.
	eventPtr := &events.head
	place := 0
	for *eventPtr != nil && (*eventPtr).clock < clock {
		*eventPtr = (*eventPtr).next
		place++
	}

	event.next = *eventPtr
	*eventPtr = event

	if eventDebug {
		log.Printf("events.add(%d at %d in place %d)", eventType, clock, place)
	}
}

// Dispatch all events that are scheduled for clock or later.
func (events *events) dispatch(clock uint64) {
	for events.head != nil && events.head.clock <= clock {
		// Remove from list before calling, to allow callback to
		// modify the list.
		event := events.head
		events.head = event.next

		if eventDebug {
			log.Printf("events.dispatch(%d at %d)", event.eventType, clock)
		}

		event.callback()
	}
}

// Remove all events in list that match the mask eventMask.
func (events *events) cancelEvents(eventMask eventType) {
	eventPtr := &events.head

	for *eventPtr != nil {
		nextEventPtr := &(*eventPtr).next
		eventType := (*eventPtr).eventType

		if eventType&eventMask != 0 {
			// Remove it from list.
			*eventPtr = *nextEventPtr

			if eventDebug {
				log.Printf("events.cancelEvents(%d)", eventType)
			}
		} else {
			// Move to next one.
			eventPtr = nextEventPtr
		}
	}
}

// Returns the first event that matches the specified mask, or nil if none are
// found.
func (events *events) getFirstEvent(eventMask eventType) *event {
	for event := events.head; event != nil; event = event.next {
		if event.eventType&eventMask != 0 {
			return event
		}
	}

	return nil
}
