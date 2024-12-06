package silence

// import (
// 	"os"
// 	"testing"

// 	"github.com/streamer45/silero-vad-go/speech"
// 	"github.com/stretchr/testify/require"

// 	"github.com/go-audio/wav"
// )

// func TestVad(t *testing.T) {
// 	sd, err := speech.NewDetector(speech.DetectorConfig{
// 		ModelPath:            "../../model/silence/silero_vad_16k.onnx",
// 		SampleRate:           16000,
// 		Threshold:            0.5,
// 		MinSilenceDurationMs: 100,
// 		SpeechPadMs:          30,
// 	})
// 	require.NoError(t, err, "failed to create speech detector")

// 	f, err := os.Open("../../model/silence/sample.wav")
// 	require.NoError(t, err, "failed to open sample audio file")
// 	defer f.Close()

// 	dec := wav.NewDecoder(f)

// 	require.True(t, dec.IsValidFile(), "invalid WAV file")

// 	buf, err := dec.FullPCMBuffer()
// 	require.NoError(t, err, "failed to get PCM buffer")

// 	segments, err := sd.Detect(buf.AsFloat32Buffer().Data)
// 	require.NoError(t, err, "detection failed")

// 	require.NotEmpty(t, segments, "no segments detected")
// 	require.NoError(t, sd.Destroy(), "failed to destroy detector")
// }
