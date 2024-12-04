package audio

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
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

// https://cloud.google.com/text-to-speech/docs/voices
func GoogleSay(msg string) (err error) {
	var url = fmt.Sprintf("http://translate.google.com/translate_tts?ie=UTF-8&total=1&idx=0&textlen=32&client=tw-ob&q=%s&tl=%s", url.QueryEscape(msg), "en")
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var sig = make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return errors.New("read response")
	}
	id, data, err := readChunk(bytes.NewReader(body))
	if err != nil {
		return err
	}
	if id.String() != "FORM" {
		return fmt.Errorf("bad file format: %s", id)
	}
	_, err = data.Read(id[:])
	if err != nil {
		return err
	}
	if id.String() != "AIFF" {
		return fmt.Errorf("bad file format: %s", id)
	}
	var c commonChunk
	var audio io.Reader
	for {
		id, chunk, err := readChunk(data)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch id.String() {
		case "COMM":
			if err = binary.Read(chunk, binary.BigEndian, &c); err != nil {
				fmt.Println("error reading COMM chunk:", err)
			}
		case "SSND":
			_, err = chunk.Seek(8, 1) //ignore offset and block
			if err != nil {
				fmt.Println("error seeking SSND chunk:", err)
			}
			audio = chunk
		default:
			fmt.Printf("ignoring chunk '%s'\n", id)
		}
	}
	//assume 44100 sample rate, mono, 32 bit
	if err = portaudio.Initialize(); err != nil {
		return fmt.Errorf("portaudio.Initialize: %w", err)
	}
	defer func() {
		err = multierr.Combine(err, portaudio.Terminate())
	}()
	var out = make([]int32, 8192)
	stream, err := portaudio.OpenDefaultStream(0, 1, 44100, len(out), &out)
	if err != nil {
		return err
	}
	defer stream.Close()
	if err = stream.Start(); err != nil {
		return err
	}
	defer func() {
		err = multierr.Combine(err, stream.Stop())
	}()
	for remaining := int(c.NumSamples); remaining > 0; remaining -= len(out) {
		if len(out) > remaining {
			out = out[:remaining]
		}
		if err := binary.Read(audio, binary.BigEndian, out); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err = stream.Write(); err != nil {
			return err
		}
		select {
		case <-sig:
			return nil
		default:
		}
	}
	return nil
}

func readChunk(r readerAtSeeker) (id ID, data *io.SectionReader, err error) {
	_, err = r.Read(id[:])
	if err != nil {
		return
	}
	var n int32
	err = binary.Read(r, binary.BigEndian, &n)
	if err != nil {
		return
	}
	off, _ := r.Seek(0, 1)
	data = io.NewSectionReader(r, off, int64(n))
	_, err = r.Seek(int64(n), 1)
	return
}

type readerAtSeeker interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

type ID [4]byte

func (id ID) String() string {
	return string(id[:])
}

type commonChunk struct {
	NumChans      int16
	NumSamples    int32
	BitsPerSample int16
	SampleRate    [10]byte
}
