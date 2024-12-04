package hotword

import (
	"encoding/json"
	"fmt"
	"os"
)

func OnnxModelPath() string {
	return "model/hotword/resnet_qint8.onnx"
}

func EmbeddingsPath() string {
	return "model/hotword/computer_ref.json"
}

type embeddingsJSON struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func LoadEmbeddings(filePath string) (weights [][]float32, err error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embeddings file %s: %w", filePath, err)
	}
	var v = new(embeddingsJSON)
	if err = json.Unmarshal(b, v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embeddings file %s: %w", filePath, err)
	}
	return v.Embeddings, nil
}
