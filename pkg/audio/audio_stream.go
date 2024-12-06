package audio

import (
	"fmt"
	"math"
	"sync"
	"time"
	"unsafe"

	"github.com/algo-boyz/snowgirl/pkg/state"
	"github.com/gordonklaus/portaudio"
)

// AudioStream implements a sliding window audio stream
type AudioStream struct {
	openStream    func() error
	closeStream   func() error
	getNextFrame  func() ([]float32, error)
	windowSize    int
	slidingWindow int
	mu            sync.Mutex
}

const sampleRate = 16000

// NewAudioStream creates a new AudioStream
func NewAudioStream(
	openStream func() error,
	closeStream func() error,
	getNextFrame func() ([]float32, error),
	windowLengthSecs float32,
	slidingWindowSecs float32,
) *AudioStream {
	var (
		windowSize        = int(windowLengthSecs * float32(sampleRate))
		slidingWindowSize = int(max(1, slidingWindowSecs*float32(sampleRate)))
	)
	return &AudioStream{
		openStream:    openStream,
		closeStream:   closeStream,
		getNextFrame:  getNextFrame,
		windowSize:    windowSize,
		slidingWindow: slidingWindowSize,
	}
}

// StartStream begins the audio stream
func (c *AudioStream) Start() (err error) {
	// Reset output audio to zeros
	if err = c.openStream(); err != nil {
		return err
	}
	// Prefill the buffer
	for i := 0; i < sampleRate/c.slidingWindow-1; i++ {
		_, err := c.GetFrame()
		if err != nil {
			return err
		}
	}
	return nil
}

// CloseStream stops the audio stream
func (c *AudioStream) CloseStream() (err error) {
	return c.closeStream()
}

// GetFrame retrieves a 1-second audio frame with sliding window
func (c *AudioStream) GetFrame() ([]float32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Get next audio frame
	frames, err := c.getNextFrame()
	if err != nil {
		return nil, err
	}
	// Slide the window
	var pcm = make([]float32, c.windowSize)
	// When opening a stream with a single-channel float input on PortAudio,
	// the input buffer is already a float32 slice
	copy(pcm[c.windowSize-c.slidingWindow:], (*[1 << 30]float32)(unsafe.Pointer(&frames[0]))[:c.slidingWindow:c.slidingWindow])
	return pcm, nil
}

// SimpleMicStream implements a microphone audio stream
type MicStream struct {
	*AudioStream
	stream          *portaudio.Stream
	framesPerBuffer int
	subscribers     []chan []float32
	subscribersMu   sync.RWMutex
}

// NewMicStream creates a new microphone audio stream
func NewMicStream(ctx state.Context, windowLengthSecs, slidingWindowSecs float32) (*MicStream, error) {
	// Initialize PortAudio
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("portaudio.Initialize: %w", err)
	}
	// Calculate chunk size
	chunkSize := round(slidingWindowSecs * float32(sampleRate))
	fmt.Println("chunk size:", chunkSize)

	deviceInfo, err := portaudio.DefaultInputDevice()
	if err != nil {
		return nil, fmt.Errorf("portaudio.DefaultInputDevice: %w", err)
	}
	inputParams := portaudio.LowLatencyParameters(deviceInfo, nil)
	inputParams.Input.Channels = 1
	inputParams.Output.Channels = 0
	inputParams.SampleRate = float64(sampleRate)
	inputParams.FramesPerBuffer = chunkSize

	var stream *portaudio.Stream
	var buffer = make([]float32, chunkSize)
	stream, err = portaudio.OpenStream(inputParams, buffer)
	if err != nil {
		return nil, fmt.Errorf("portaudio.OpenStream: %w", err)
	}
	go ctx.Defer(func() {
		fmt.Println("portaudio exiting")
		if err = stream.Stop(); err != nil {
			fmt.Printf("failed to stop audio stream: %s\n", err)
		}
		if err = stream.Close(); err != nil {
			fmt.Printf("failed to close audio stream: %s\n", err)
		}
		if err = portaudio.Terminate(); err != nil {
			fmt.Printf("failed to terminate portaudio: %s\n", err)
		}
	})
	var micStream = &MicStream{
		AudioStream: NewAudioStream(
			stream.Start,
			stream.Stop,
			func() ([]float32, error) {
				return buffer, stream.Read()
			},
			windowLengthSecs,
			slidingWindowSecs,
		),
		subscribers:     make([]chan []float32, 0),
		framesPerBuffer: chunkSize,
		stream:          stream,
	}
	go func() {
		time.Sleep(time.Minute)
		micStream.broadcast(ctx)
	}()
	return micStream, nil
}

// Subscribe creates a new channel for receiving audio frames
func (s *MicStream) Subscribe() <-chan []float32 {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	var ch = make(chan []float32, 10) // Buffered channel to prevent blocking
	s.subscribers = append(s.subscribers, ch)
	return ch
}

// Unsubscribe removes a specific subscriber channel
func (s *MicStream) Unsubscribe(ch <-chan []float32) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for i, subscriber := range s.subscribers {
		if subscriber == ch {
			close(subscriber)
			// Remove the channel from the slice
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			break
		}
	}
}

// broadcast continuously reads audio frames and sends them to all subscribers
func (s *MicStream) broadcast(ctx state.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read audio frame
			frame, err := s.GetFrame()
			if err != nil {
				panic(fmt.Errorf("micStream.GetFrame: %w", err))
			}
			// Broadcast to all subscribers
			s.subscribersMu.RLock()
			for _, ch := range s.subscribers {
				select {
				case ch <- append([]float32(nil), frame...):
				default:
					// Skip if channel is full to prevent blocking
				}
			}
			s.subscribersMu.RUnlock()
		}
	}
}

func round(v float32) int {
	if v >= 0 {
		return int(math.Floor(float64(v + 0.5)))
	}
	return int(math.Ceil(float64(v - 0.5)))
}
