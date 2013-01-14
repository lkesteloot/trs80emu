// Copyright 2012 Lawrence Kesteloot

package main

// Screen constants and utilities.

const (
	screenRows    = 16
	screenColumns = 64
	screenBegin   = 0x3C00
	screenEnd     = screenBegin + screenRows*screenColumns
)

func (vm *vm) setExpandedCharacters(expanded bool) {
	if vm.vmUpdateCh != nil {
		value := 0
		if expanded {
			value = 1
		}

		vm.vmUpdateCh <- vmUpdate{Cmd: "expanded", Data: value}
	}
}
