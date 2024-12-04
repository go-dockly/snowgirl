package hotword

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScoreVector(t *testing.T) {
	model := &Model{
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			{0.9, 0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.2, 0.1, 0.0},
		},
	}
	tests := []struct {
		inputVector []float32
		expected    float32
	}{
		{
			inputVector: make([]float32, 2048),
			expected:    0.5, // Assuming the embeddings are normalized
		},
		{
			inputVector: []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			expected:    0.0, // Dimension mismatch
		},
	}
	for _, test := range tests {
		result := model.ScoreVector(test.inputVector)
		require.Equal(t, test.expected, result, "expected %v, got %v", test.expected, result)
	}
}
