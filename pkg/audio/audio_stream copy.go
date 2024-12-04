package audio

// import (
// 	"fmt"
// 	"sync"

// 	"github.com/algo-boyz/snowgirl/pkg/state"
// 	"github.com/gordonklaus/portaudio"
// )

// // AudioStream implements a sliding window audio stream
// type AudioStream struct {
// 	openStream    func() error
// 	closeStream   func() error
// 	getNextFrame  func() ([]int16, error)
// 	windowSize    int
// 	slidingWindow int
// 	outputAudio   []float32
// 	mu            sync.Mutex
// }

// const sampleRate = 16000

// // NewAudioStream creates a new AudioStream
// func NewAudioStream(
// 	openStream func() error,
// 	closeStream func() error,
// 	getNextFrame func() ([]int16, error),
// 	windowLengthSecs float32,
// 	slidingWindowSecs float32,
// ) *AudioStream {
// 	var (
// 		windowSize        = int(windowLengthSecs * float32(sampleRate))
// 		slidingWindowSize = int(max(1, slidingWindowSecs*float32(sampleRate)))
// 		outputAudio       = make([]float32, windowSize) // Initialize output audio with zeros
// 	)
// 	return &AudioStream{
// 		openStream:    openStream,
// 		closeStream:   closeStream,
// 		getNextFrame:  getNextFrame,
// 		windowSize:    windowSize,
// 		slidingWindow: slidingWindowSize,
// 		outputAudio:   outputAudio,
// 	}
// }

// // StartStream begins the audio stream
// func (c *AudioStream) Start() (err error) {
// 	// Reset output audio to zeros
// 	c.outputAudio = make([]float32, c.windowSize)
// 	if err = c.openStream(); err != nil {
// 		return err
// 	}
// 	// Prefill the buffer
// 	for i := 0; i < sampleRate/c.slidingWindow-1; i++ {
// 		_, err := c.GetFrame()
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// // CloseStream stops the audio stream
// func (c *AudioStream) CloseStream() (err error) {
// 	err = c.closeStream()
// 	c.outputAudio = make([]float32, c.windowSize)
// 	return err
// }

// // GetFrame retrieves a 1-second audio frame with sliding window
// func (c *AudioStream) GetFrame() ([]float32, error) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	// Get next audio frame
// 	frames, err := c.getNextFrame()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(frames) != c.slidingWindow {
// 		return nil, fmt.Errorf("audio frame size does not match sliding window size")
// 	}
// 	// Slide the window
// 	copy(c.outputAudio, c.outputAudio[c.slidingWindow:])
// 	// Convert audio frame to PCM
// 	pcm := make([]float32, len(frames))
// 	for i, sample := range frames {
// 		pcm[i] = float32(sample) / float32(1<<15)
// 	}
// 	copy(c.outputAudio[c.windowSize-c.slidingWindow:], pcm)

// 	return c.outputAudio, nil
// }

// // SimpleMicStream implements a microphone audio stream
// type MicStream struct {
// 	*AudioStream
// 	stream          *portaudio.Stream
// 	framesPerBuffer int
// 	subscribers     []chan []float32
// 	subscribersMu   sync.RWMutex
// }

// // NewMicStream creates a new microphone audio stream
// func NewMicStream(ctx state.Context, windowLengthSecs, slidingWindowSecs float32) (*MicStream, error) {
// 	// Initialize PortAudio
// 	if err := portaudio.Initialize(); err != nil {
// 		return nil, fmt.Errorf("portaudio.Initialize: %w", err)
// 	}
// 	// Calculate chunk size
// 	chunkSize := round(slidingWindowSecs * float32(sampleRate))
// 	fmt.Println("chunk size:", chunkSize)

// 	deviceInfo, err := portaudio.DefaultInputDevice()
// 	if err != nil {
// 		return nil, fmt.Errorf("portaudio.DefaultInputDevice: %w", err)
// 	}
// 	inputParams := portaudio.LowLatencyParameters(deviceInfo, nil)
// 	inputParams.Input.Channels = 1
// 	inputParams.Output.Channels = 0
// 	inputParams.SampleRate = float64(sampleRate)
// 	inputParams.FramesPerBuffer = chunkSize

// 	var stream *portaudio.Stream
// 	var buffer = make([]int16, chunkSize)
// 	stream, err = portaudio.OpenStream(inputParams, buffer)
// 	if err != nil {
// 		return nil, fmt.Errorf("portaudio.OpenStream: %w", err)
// 	}
// 	go ctx.Defer(func() {
// 		fmt.Println("portaudio exiting")
// 		if err = stream.Stop(); err != nil {
// 			fmt.Printf("failed to stop audio stream: %s\n", err)
// 		}
// 		if err = stream.Close(); err != nil {
// 			fmt.Printf("failed to close audio stream: %s\n", err)
// 		}
// 		if err = portaudio.Terminate(); err != nil {
// 			fmt.Printf("failed to terminate portaudio: %s\n", err)
// 		}
// 	})
// 	var micStream = &MicStream{
// 		AudioStream: NewAudioStream(
// 			stream.Start,
// 			stream.Stop,
// 			func() ([]int16, error) {
// 				return buffer, stream.Read()
// 			},
// 			windowLengthSecs,
// 			slidingWindowSecs,
// 		),
// 		subscribers:     make([]chan []float32, 0),
// 		framesPerBuffer: chunkSize,
// 		stream:          stream,
// 	}
// 	go micStream.broadcast(ctx)
// 	return micStream, nil
// }

// // Subscribe creates a new channel for receiving audio frames
// func (s *MicStream) Subscribe() <-chan []float32 {
// 	s.subscribersMu.Lock()
// 	defer s.subscribersMu.Unlock()

// 	var ch = make(chan []float32, 10) // Buffered channel to prevent blocking
// 	s.subscribers = append(s.subscribers, ch)
// 	return ch
// }

// // Unsubscribe removes a specific subscriber channel
// func (s *MicStream) Unsubscribe(ch <-chan []float32) {
// 	s.subscribersMu.Lock()
// 	defer s.subscribersMu.Unlock()

// 	for i, subscriber := range s.subscribers {
// 		if subscriber == ch {
// 			close(subscriber)
// 			// Remove the channel from the slice
// 			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
// 			break
// 		}
// 	}
// }

// // broadcast continuously reads audio frames and sends them to all subscribers
// func (s *MicStream) broadcast(ctx state.Context) {
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		default:
// 			// Read audio frame
// 			frame, err := s.GetFrame()
// 			if err != nil {
// 				panic(fmt.Errorf("micStream.GetFrame: %w", err))
// 			}
// 			// Broadcast to all subscribers
// 			s.subscribersMu.RLock()
// 			for _, ch := range s.subscribers {
// 				select {
// 				case ch <- append([]float32(nil), frame...):
// 				default:
// 					// Skip if channel is full to prevent blocking
// 				}
// 			}
// 			s.subscribersMu.RUnlock()
// 		}
// 	}
// }

// func round(f float32) int {
// 	if f < 0 {
// 		return int(f - 0.5)
// 	}
// 	return int(f + 0.5)
// }
