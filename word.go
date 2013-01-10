// Copyright 2012 Lawrence Kesteloot

package main

// A 16-bit unsigned int that can be accessed by its bytes.
type word uint16

// Gets the low-order byte.
func (w word) l() byte {
	return byte(w)
}

// Gets the high-order byte.
func (w word) h() byte {
	return byte(w >> 8)
}

// Sets the low-order byte.
func (w *word) setL(l byte) {
	*w = (*w &^ 0x00FF) | word(l)
}

// Sets the high-order byte.
func (w *word) setH(h byte) {
	*w = (*w &^ 0xFF00) | (word(h) << 8)
}
