package audio

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-audio/wav"
	"github.com/hajimehoshi/go-mp3"
	"go.uber.org/multierr"
)

func Load(filePath string) (frame []float32, err error) {
	switch ext := filepath.Ext(filePath); ext {
	case ".mp3":
		return loadMP3(filePath)
	case ".wav":
		return loadWAV(filePath)
	default:
		return nil, fmt.Errorf("unsupported audio file extension: %s", ext)
	}
}

func loadMP3(filePath string) (frame []float32, err error) {
	audioFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening MP3 file: %v", err)
	}
	defer func() {
		err = multierr.Combine(err, audioFile.Close())
	}()
	// Decode the MP3 file
	decoder, err := mp3.NewDecoder(audioFile)
	if err != nil {
		return nil, fmt.Errorf("error creating MP3 decoder: %v", err)
	}
	// Read audio data
	var b = make([]byte, sampleRate*2) // 16-bit audio, so 2 bytes per sample
	n, err := decoder.Read(b)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading MP3 data: %v", err)
	}
	// Convert to float64 slice
	frame = make([]float32, n/2)
	for i := 0; i < len(frame); i++ {
		// Convert 16-bit PCM to float64
		var sample = int16(b[i*2]) | int16(b[i*2+1])<<8
		frame[i] = float32(sample) / 32768.0
	}
	return frame, nil
}

func loadWAV(filePath string) (frame []float32, err error) {
	audioFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening WAV file: %v", err)
	}
	defer func() {
		err = multierr.Combine(err, audioFile.Close())
	}()
	// Decode the WAV file
	decoder := wav.NewDecoder(audioFile)
	// Check if the file is valid and has PCM format
	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file")
	}
	// Decode WAV file to PCM samples
	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("error decoding WAV file: %v", err)
	}
	// Convert PCM int samples to float32
	frame = make([]float32, len(buffer.Data))
	for i, sample := range buffer.Data {
		frame[i] = float32(sample / 1 << 15) // Assuming 16-bit PCM
	}
	return frame, nil
}
