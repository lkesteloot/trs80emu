package main

// Information about changes to the CPU or computer.
type cpuUpdate struct {
	Cmd  string
	Reg  string
	Addr int
	Data int
}

type startUpdates struct {
	updateCh chan<- cpuUpdate
}

type stopUpdates struct {
}

func dispatchUpdates(updateCh <-chan cpuUpdate, updateCmdCh <-chan interface{}) {
	var dispatchedUpdateCh chan<- cpuUpdate
	var data cpuUpdate
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
