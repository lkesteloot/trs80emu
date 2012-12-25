// Copyright 2012 Lawrence Kesteloot

package main

import (
	"testing"
)

func TestDisasm(t *testing.T) {
	if substituteData("CP N", 0x45, 0) != "CP 45" {
		t.Errorf("CP N failed")
	}
	if substituteData("JP NN,6", 0, 0x1234) != "JP 1234,6" {
		t.Errorf("JP NN,6 failed")
	}
	// Shouldn't touch the N in AND.
	if substituteData("AND N", 0x45, 0) != "AND 45" {
		t.Errorf("AND N failed")
	}
}
