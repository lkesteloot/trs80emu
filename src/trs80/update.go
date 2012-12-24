package main

// Information about changes to the CPU or computer.
type vmUpdate struct {
	Cmd  string
	Reg  string
	Addr int
	Data int
}

type startUpdates struct {
	updateCh chan<- vmUpdate
}

type stopUpdates struct {
}

func dispatchUpdates(updateCh <-chan vmUpdate, updateCmdCh <-chan interface{}) {
	var dispatchedUpdateCh chan<- vmUpdate
	var data vmUpdate
	var cmd interface{}

	for {
		select {
		case data = <-updateCh:
			if dispatchedUpdateCh != nil {
				dispatchedUpdateCh <- data
			}
		case cmd = <-updateCmdCh:
			start, ok := cmd.(startUpdates)
			if ok {
				dispatchedUpdateCh = start.updateCh
			}
			_, ok = cmd.(stopUpdates)
			if ok {
				dispatchedUpdateCh = nil
			}
		}
	}
}
