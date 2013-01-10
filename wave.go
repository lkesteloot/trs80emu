// Copyright 2012 Lawrence Kesteloot

package main

// Parse .WAV files for cassette support.

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Holds information about the WAV file.
type wavFile struct {
	io.ReadSeeker
	channelCount     uint16
	samplesPerSecond uint32
	bytesPerSample   uint16
	bitsPerSample    uint16
	isEof            bool
}

// Parses .WAV file headers.
func openWav(filename string) (w *wavFile, err error) {
	// Open the file.
	f, err := os.Open(filename)
	if err != nil {
		return
	}

	w = &wavFile{f, 0, 0, 0, 0, false}

	// Parse header.
	err = w.parseChunkId("RIFF")
	if err != nil {
		return
	}
	// Length of the rest of the file.
	_, err = w.parseInt()
	if err != nil {
		return
	}
	err = w.parseChunkId("WAVE")
	if err != nil {
		return
	}
	err = w.parseChunkId("fmt ")
	if err != nil {
		return
	}
	// Format chunk size.
	chunkSize, err := w.parseInt()
	if err != nil {
		return
	}
	// Format.
	format, err := w.parseShort()
	if err != nil {
		return
	}
	if format != 1 {
		panic(fmt.Sprintf("We only handle PCM format (1), not %d", format))
	}
	// Number of channels.
	w.channelCount, err = w.parseShort()
	if err != nil {
		return
	}
	// Sample rate.
	w.samplesPerSecond, err = w.parseInt()
	if err != nil {
		return
	}
	// Ignore this int.
	_, err = w.parseInt()
	if err != nil {
		return
	}
	// Bytes per sample.
	w.bytesPerSample, err = w.parseShort()
	if err != nil {
		return
	}
	// Bits per sample.
	w.bitsPerSample, err = w.parseShort()
	if err != nil {
		return
	}
	// Read rest of format chunk.
	chunkSize -= 16 // Amount read already.
	if chunkSize > 0 {
		if wavDebug {
			log.Printf("Skipping %d bytes in fmt chunk", chunkSize)
		}
		b := make([]byte, chunkSize)
		_, err = io.ReadFull(w, b)
		if err != nil {
			return
		}
	}
	// Start data chunk.
	err = w.parseChunkId("data")
	if err != nil {
		return
	}
	// Size of data chunk.
	_, err = w.parseInt()
	if err != nil {
		return
	}

	return
}

// Parse a 4-byte ASCII chunk ID and verify that it matches the given ID.
func (w *wavFile) parseChunkId(expectedChunkId string) error {
	// Read four bytes.
	b := make([]byte, 4)
	_, err := io.ReadFull(w, b)
	if err != nil {
		return err
	}

	// Compare to expected chunk ID.
	foundChunkId := string(b)
	if foundChunkId != expectedChunkId {
		return fmt.Errorf("Expected chunk ID \"%s\" but got \"%s\"", expectedChunkId, foundChunkId)
	}

	if wavDebug {
		log.Printf("Found expected chunk ID \"%s\"", expectedChunkId)
	}

	return nil
}

// Loads a 4-byte little-endian int.
func (w *wavFile) parseInt() (uint32, error) {
	// Read four bytes.
	b := make([]byte, 4)
	_, err := io.ReadFull(w, b)
	if err != nil {
		return 0, err
	}

	// Little-endian.
	n := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24

	if wavDebug {
		log.Printf("Found 4-byte integer %d (0x%08X)", n, n)
	}

	return n, nil
}

// Loads a 2-byte little-endian int.
func (w *wavFile) parseShort() (uint16, error) {
	// Read two bytes.
	b := make([]byte, 2)
	_, err := io.ReadFull(w, b)
	if err != nil {
		return 0, err
	}

	// Little-endian.
	n := uint16(b[0]) | uint16(b[1])<<8

	if wavDebug {
		log.Printf("Found 2-byte integer %d (0x%04X)", n, n)
	}

	return n, nil
}

// Loads a sample.
func (w *wavFile) readSample() (int16, error) {
	// Only handle simple case.
	if w.channelCount != 1 || w.bytesPerSample != 2 || w.bitsPerSample != 16 {
		panic("Don't handle WAV file format")
	}

	if w.isEof {
		// Pretend that the tape stopped and that we're just
		// reading silence. That's probably what the original
		// computer did.
		return 0, nil
	}

	s, err := w.parseShort()
	if err == io.EOF {
		log.Print("End of cassette")
		w.isEof = true
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return int16(s), nil
}
