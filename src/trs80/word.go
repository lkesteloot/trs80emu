package main

type word uint16

func (w word) l() byte {
	return byte(w)
}

func (w word) h() byte {
	return byte(w >> 8)
}

func (w *word) setL(l byte) {
	*w = (*w &^ 0x00FF) | word(l)
}

func (w *word) setH(h byte) {
	*w = (*w &^ 0xFF00) | (word(h) << 8)
}
