// Copyright 2013 Lawrence Kesteloot

package main

// Profiles a function to sort strings taking into account imbedded numbers.

import (
	"sort"
)

// Return -1, 0, or 1 if a is less than, equal to, or greater than b, taking into
// account embedded numbers.
func compareStringsNumerically(a, b string) int {
	var i, j int

	var isDigit = func(ch byte) bool {
		return ch >= '0' && ch <= '9'
	}

	// Return the next integer in the string and the position after the last digit.
	var parseNextInt = func(s string, i int) (value int, nextI int) {
		value = 0
		for i < len(s) && isDigit(s[i]) {
			value = value*10 + int(s[i]-'0')
			i++
		}
		nextI = i
		return
	}

	// Walk through both strings at the same time.
	for i < len(a) && j < len(b) {
		var chunkA, chunkB int

		// Get the next "chunk", which could be a letter or a number.
		if isDigit(a[i]) && isDigit(b[j]) {
			// Only compare numerically if both are numbers.
			chunkA, i = parseNextInt(a, i)
			chunkB, j = parseNextInt(b, j)
		} else {
			// Compare ASCII.
			chunkA = int(a[i])
			i++
			chunkB = int(b[j])
			j++
		}

		if chunkA < chunkB {
			return -1
		}
		if chunkA > chunkB {
			return 1
		}
	}

	// One or the other ended. Whichever one ended first is "less".
	if i == len(a) && j == len(b) {
		return 0
	}
	if i == len(a) {
		return -1
	}
	return 1
}

// A string slice that implements sort.Interface, comparing numbers properly.
type numericalStringSlice []string

// Return the length of the slice.
func (s numericalStringSlice) Len() int {
	return len(s)
}

// Return whether string at i is less than the one at j. Embedded numbers are
// handled properly, meaning that "B2" is less than "B10".
func (s numericalStringSlice) Less(i, j int) bool {
	return compareStringsNumerically(s[i], s[j]) < 0
}

// Swap strings are positions i and j.
func (s numericalStringSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Sort strings in place, putting numbers in their proper order.
func sortNumerically(s []string) {
	// Use the methods on numericalStringSlice to compare.
	sort.Sort(numericalStringSlice(s))
}
