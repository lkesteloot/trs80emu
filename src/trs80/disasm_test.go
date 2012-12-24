// Copyright 2012 Lawrence Kesteloot

package main

import (
	"testing"
)

func TestDisasm(t *testing.T) {
	if nRegExp.ReplaceAllLiteralString("CP N", "45") != "CP 45" {
		t.Errorf("CP N failed")
	}
	if nnRegExp.ReplaceAllLiteralString("JP NN,6", "1234") != "JP 1234,6" {
		t.Errorf("JP NN,6 failed")
	}
	// Shouldn't touch the N in AND.
	if nRegExp.ReplaceAllLiteralString("AND N", "45") != "AND 45" {
		t.Errorf("AND N failed")
	}
}
