// Copyright 2013 Lawrence Kesteloot

package main

import (
	"testing"
)

func TestCompareStringsNumerically(t *testing.T) {
	cmp := func (a, b string, expected int) {
		result := compareStringsNumerically(a, b)
		if result != expected {
			t.Errorf("Comparing \"%s\" and \"%s\" returned %d, expected %d", a, b, result, expected)
		}
	}

	// Normal sorting.
	cmp("", "", 0)
	cmp("A", "A", 0)
	cmp("A", "B", -1)
	cmp("B", "A", 1)
	cmp("AA", "A", 1)
	cmp("A", "AA", -1)
	cmp("A20", "A20", 0)
	cmp("A25", "A24", 1)
	cmp("A24", "A25", -1)

	// Numerical sorting.
	cmp("A2", "A25", -1)
	cmp("A2", "A10", -1)
	cmp("A", "A10", -1)
	cmp("B", "A10", 1)
	cmp("A123B", "A123C", -1)
	cmp("A123C", "A123B", 1)
	cmp("A123C5", "A123C44", -1)
}
