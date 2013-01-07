// Copyright 2012 Lawrence Kesteloot

package main

// Parse .WAV files for cassette support.

import (
	"log"
	"fmt"
	"io"
	"os"
)

type wavFile struct {
	io.ReadSeeker
	channelCount uint16
	sampleRate uint32
	bytesPerSample uint16
	bitsPerSample uint16
}

// Parses .WAV file headers.
func openWav(filename string) (w *wavFile, err error) {
	// Open the file.
	f, err := os.Open(filename)
	if err != nil {
		return
	}

	w = &wavFile{f, 0, 0, 0, 0}

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
	w.sampleRate, err = w.parseInt()
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
