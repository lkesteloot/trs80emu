// Copyright 2012 Lawrence Kesteloot

package main

// Information about changes to the CPU or computer.
type vmUpdate struct {
	Cmd  string
	Msg  string
	Reg  string
	Addr int
	Data int
}
