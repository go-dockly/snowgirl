package hotword

import (
	"fmt"
	"math"

	"github.com/mjibson/go-dsp/fft"
	"gonum.org/v1/gonum/mat"
)

type LogMelSpectrogram struct {
	SampleRate   int
	WindowLen    int
	HopLength    int
	NumMelBands  int
	NFFTSize     int
	LowFreq      float32
	HighFreq     float32
	PreEmphCoeff float32
	WindowFunc   func(int) []float64
}

// This assumes a mono channel input
func DefaultLogMelSpectrogram() *LogMelSpectrogram {
	return NewLogMelSpectrogram(
		sampleRate,
		0.025,                 // window length (seconds)
		0.01,                  // window step (seconds)
		26,                    // number of mel bands
		512,                   // FFT size
		0,                     // low frequency
		float32(sampleRate)/2, // high frequency
		0.97,                  // preemphasis coefficient
		HannWindow,            // window function
	)
}

// NewLogMelSpectrogram creates a new LogMelSpectrogram configuration
func NewLogMelSpectrogram(
	sampleRate int,
	winlen float32,
	winstep float32,
	nfilt int,
	nfft int,
	lowfreq float32,
	highfreq float32,
	preemph float32,
	windowFunc func(int) []float64,
) *LogMelSpectrogram {
	if windowFunc == nil {
		windowFunc = DefaultWindow
	}
	return &LogMelSpectrogram{
		SampleRate:   sampleRate,
		WindowLen:    int(winlen * float32(sampleRate)),
		HopLength:    int(winstep * float32(sampleRate)),
		NumMelBands:  nfilt,
		NFFTSize:     nfft,
		LowFreq:      lowfreq,
		HighFreq:     highfreq,
		PreEmphCoeff: preemph,
		WindowFunc:   windowFunc,
	}
}

// DefaultWindow returns the full frame
func DefaultWindow(size int) []float64 {
	return make([]float64, size)
}

// HannWindow creates a Hann window
func HannWindow(size int) []float64 {
	var window = make([]float64, size)
	for i := 0; i < size; i++ {
		window[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(size-1)))
	}
	return window
}

// Preemphasis applies a high-pass filter to the signal
func Preemphasis(signal []float32, coeff float32) []float32 {
	if len(signal) <= 1 || coeff == 0 {
		return signal
	}
	var preemphasized = make([]float32, len(signal))
	preemphasized[0] = signal[0]
	for i := 1; i < len(signal); i++ {
		preemphasized[i] = signal[i] - coeff*signal[i-1]
	}
	return preemphasized
}

// HzToMel converts frequency from Hz to Mel scale
func HzToMel(hz float32) float32 {
	return float32(2595 * math.Log10(1+float64(hz)/700.0))
}

// MelToHz converts frequency from Mel scale to Hz
func MelToHz(mel float32) float32 {
	return float32(700 * (math.Pow(10, float64(mel)/2595.0) - 1))
}

// CreateMelFilterbank generates mel filterbank matrix
func CreateMelFilterbank(numMelBands, windowSize, sampleRate int, lowFreq, highFreq float32) *mat.Dense {
	var (
		melMin     = HzToMel(lowFreq)
		melMax     = HzToMel(highFreq)
		melPoints  = make([]float32, numMelBands+2)
		freqPoints = make([]float32, numMelBands+2)
		fftBins    = make([]int, numMelBands+2)
		filterbank = mat.NewDense(numMelBands, windowSize/2+1, nil)
	)
	for i := 0; i < numMelBands+2; i++ {
		melPoints[i] = melMin + (melMax-melMin)*float32(i)/float32(numMelBands+1)
	}
	for i := 0; i < numMelBands+2; i++ {
		freqPoints[i] = MelToHz(melPoints[i])
	}
	for i := 0; i < numMelBands+2; i++ {
		fftBins[i] = int(math.Floor(float64(float32(windowSize+1) * freqPoints[i] / float32(sampleRate))))
	}
	for j := 0; j < numMelBands; j++ {
		for i := fftBins[j]; i < fftBins[j+1]; i++ {
			filterbank.Set(j, i, float64(i-fftBins[j])/float64(fftBins[j+1]-fftBins[j]))
		}
		for i := fftBins[j+1]; i < fftBins[j+2]; i++ {
			filterbank.Set(j, i, float64(fftBins[j+2]-i)/float64(fftBins[j+2]-fftBins[j+1]))
		}
	}
	return filterbank
}

// ComputeLogMelSpectrogram generates a log mel spectrogram from audio signal
func (lms *LogMelSpectrogram) ComputeLogMelSpectrogram(signal []float32) ([][]float32, error) {
	// Preemphasis
	signal = Preemphasis(signal, lms.PreEmphCoeff)
	// Compute number of frames
	var numFrames = 1 + (len(signal)-lms.WindowLen)/lms.HopLength
	if numFrames <= 0 {
		return nil, fmt.Errorf("signal too short for given window and hop lengths")
	}
	// Apply window function
	window := lms.WindowFunc(lms.WindowLen)
	// Create mel filterbank
	melFilterbank := CreateMelFilterbank(
		lms.NumMelBands,
		lms.NFFTSize,
		lms.SampleRate,
		lms.LowFreq,
		lms.HighFreq,
	)
	// Prepare output mel spectrogram
	melSpectrogram := make([][]float32, lms.NumMelBands)
	for i := range melSpectrogram {
		melSpectrogram[i] = make([]float32, numFrames)
	}
	// Process each frame
	for frame := 0; frame < numFrames; frame++ {
		start := frame * lms.HopLength
		// Extract and window the frame
		framedAudio := make([]float64, lms.WindowLen)
		for i := 0; i < lms.WindowLen; i++ {
			if start+i < len(signal) {
				framedAudio[i] = float64(signal[start+i]) * window[i]
			}
		}
		// Compute FFT
		fftResult := fft.FFTReal(framedAudio)
		// Compute magnitude spectrum
		magnitudeSpectrum := make([]float64, len(fftResult)/2+1)
		for i := 0; i <= len(fftResult)/2; i++ {
			magnitudeSpectrum[i] = math.Sqrt(real(fftResult[i])*real(fftResult[i]) + imag(fftResult[i])*imag(fftResult[i]))
		}
		// Apply mel filterbank
		melSpectrum := make([]float64, lms.NumMelBands)
		for m := 0; m < lms.NumMelBands; m++ {
			var sum float64
			for k := 0; k < len(magnitudeSpectrum); k++ {
				sum += melFilterbank.At(m, k) * magnitudeSpectrum[k]
			}
			melSpectrum[m] = sum
		}
		// Convert to log scale (with small epsilon to avoid log(0))
		for m := 0; m < lms.NumMelBands; m++ {
			melSpectrogram[m][frame] = float32(math.Log(melSpectrum[m] + 1e-10))
		}
	}
	return melSpectrogram, nil
}

func (lms *LogMelSpectrogram) AudioToVector(inpAudio []float32) ([]float32, error) {
	// Compute log mel spectrogram features
	features, err := lms.ComputeLogMelSpectrogram(inpAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to compute log mel spectrogram: %v", err)
	}
	// Ensure the input matches the expected shape [1, 1, 149, 64]
	expectedFrames := 149
	expectedMelBands := 64
	// Pad or truncate mel bands
	if len(features) > expectedMelBands {
		features = features[:expectedMelBands]
	} else if len(features) < expectedMelBands {
		paddedFeatures := make([][]float32, expectedMelBands)
		for m := 0; m < expectedMelBands; m++ {
			if m < len(features) {
				paddedFeatures[m] = make([]float32, expectedFrames)
				copy(paddedFeatures[m], features[m])
			} else {
				paddedFeatures[m] = make([]float32, expectedFrames)
			}
		}
		features = paddedFeatures
	}
	// Pad or truncate frames
	for m := 0; m < len(features); m++ {
		if len(features[m]) > expectedFrames {
			features[m] = features[m][:expectedFrames]
		} else if len(features[m]) < expectedFrames {
			paddedFrame := make([]float32, expectedFrames)
			copy(paddedFrame, features[m])
			features[m] = paddedFrame
		}
	}
	// Reshape features for ONNX input: [1, 1, num_mel_bands, num_frames]
	reshapedFeatures := make([]float32, 1*1*expectedMelBands*expectedFrames)
	for m := 0; m < expectedMelBands; m++ {
		for t := 0; t < expectedFrames; t++ {
			reshapedFeatures[m*expectedFrames+t] = features[m][t]
		}
	}
	return reshapedFeatures, nil
}

// ScoreVector calculates the maximum cosine similarity score between an input vector
// and a set of embeddings
func (m *Model) ScoreVector(inputVector []float32) float32 {
	if len(inputVector) != 2048 {
		return .0 // Dimension mismatch
	}
	// Compute cosine similarities for each embedding
	var cosineSimilarities []float32
	for _, embedding := range m.Embeddings {
		// Compute raw dot product without explicit normalization
		dotProd := dotProduct(inputVector, embedding)
		// Normalize score to [0, 1] range
		similarity := (dotProd + 1) / 2
		cosineSimilarities = append(cosineSimilarities, similarity)
	}
	// Find maximum similarity
	var maxSimilarity float32 = .0
	for _, sim := range cosineSimilarities {
		if sim > maxSimilarity {
			maxSimilarity = sim
		}
	}
	return maxSimilarity
}

// Compute the dot product of two vectors
func dotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}
