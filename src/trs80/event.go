package main

const (
	eventDiskDone = eventType(iota)
	eventDiskLostData
	eventDiskFirstDrq
)

type eventType int
type eventCallback func ()

type event struct {
	eventType eventType
	callback eventCallback
	clock uint64
	next *event
}

type events struct {
	head *event
}

// Queue up an event to happen at clock, using a delta clock relative to the
// current time.
func (cpu *cpu) addEvent(eventType eventType, callback eventCallback, deltaClock uint64) {
	cpu.events.add(eventType, callback, cpu.clock + deltaClock)
}

// Queue up an event to happen at clock.
func (events *events) add(eventType eventType, callback eventCallback, clock uint64) {
	event := &event{eventType, callback, clock, nil}

	// Insert into list sorted by clock.
	eventPtr := &events.head
	for *eventPtr != nil && (*eventPtr).clock < clock {
		*eventPtr = (*eventPtr).next
	}

	event.next = *eventPtr
	*eventPtr = event
}

// Dispatch all events that are scheduled for clock or later.
func (events *events) dispatch(clock uint64) {
	for events.head != nil && events.head.clock <= clock {
		// Remove from list before calling, to allow callback to
		// modify the list.
		event := events.head
		events.head = event.next

		event.callback()
	}
}

// Remove all events in list that are of type eventType.
func (events *events) cancelEvents(eventType eventType) {
	eventPtr := &events.head

	for *eventPtr != nil {
		nextEventPtr := &(*eventPtr).next

		if (*eventPtr).eventType == eventType {
			// Skip it.
			*eventPtr = *nextEventPtr
		} else {
			// Move to next one.
			eventPtr = nextEventPtr
		}
	}
}

// Cancel all disk-related events.
func (events *events) cancelDiskEvents() {
	events.cancelEvents(eventDiskDone)
	events.cancelEvents(eventDiskLostData)
	events.cancelEvents(eventDiskFirstDrq)
}

// Returns the first event of the specified type, or nil if none are of that
// type.
func (events *events) getFirstEvent(eventType eventType) *event {
	for event := events.head; event != nil; event = event.next {
		if event.eventType == eventType {
			return event
		}
	}

	return nil
}
