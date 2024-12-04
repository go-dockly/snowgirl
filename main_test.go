package main

import (
	"testing"

	"github.com/algo-boyz/snowgirl/pkg/audio"
	"github.com/algo-boyz/snowgirl/pkg/hotword"
	"github.com/algo-boyz/snowgirl/pkg/onnx"
	"github.com/algo-boyz/snowgirl/pkg/state"
	"github.com/stretchr/testify/require"
)

func TestSnowgirl(t *testing.T) {
	embeddings, err := hotword.LoadEmbeddings("model/hotword/computer_ref.json")
	require.NoError(t, err, "failed to load embeddings")

	model, err := hotword.NewModel(state.NewContext(), onnx.LibPath(), hotword.OnnxModelPath(), embeddings)
	require.NoError(t, err, "failed to init onnx session")
	defer func() {
		require.NoError(t, model.Destroy(), "failed to destroy onnx session")
	}()

	audioData, err := audio.Load("model/hotword/computer.mp3")
	require.NoError(t, err, "failed to load mp3")

	frame, err := hotword.DefaultLogMelSpectrogram().AudioToVector(audioData)
	require.NoError(t, err, "failed to vectorize audio frame")

	processed, err := model.ProcessFrame(frame)
	require.NoError(t, err, "failed to process audio frame")

	confidence := model.ScoreVector(processed)

	require.True(t, confidence > 0.7, "expected confidence > 0.7 got %f", confidence)
}
