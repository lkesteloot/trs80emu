// Copyright 2012 Lawrence Kesteloot

package main

// Screen constants and utilities.

import (
	"fmt"
)

const (
	screenRows    = 16
	screenColumns = 64
	screenBegin   = 0x3C00
	screenEnd     = screenBegin + screenRows*screenColumns
)

// Dump the contents of the screen to the terminal.
func (vm *vm) dumpScreen() {
	addr := word(screenBegin)
	for y := 0; y < screenRows; y++ {
		for x := 0; x < screenColumns; x++ {
			b := vm.memory[addr]
			if b < 32 || b > 127 {
				fmt.Printf("(%02X)", b)
			} else {
				fmt.Printf("%c", b)
			}
			addr++
		}
		fmt.Println()
	}
	fmt.Println()
}
