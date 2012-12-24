// Copyright 2012 Lawrence Kesteloot

package main

import (
	"fmt"
)

const screenRows = 16
const screenColumns = 64
const screenBegin = 0x3C00
const screenEnd = screenBegin + screenRows*screenColumns

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
