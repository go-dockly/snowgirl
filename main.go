package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/algo-boyz/snowgirl/pkg/hotword"
	"github.com/algo-boyz/snowgirl/pkg/onnx"
	"github.com/algo-boyz/snowgirl/pkg/silence"
	"github.com/algo-boyz/snowgirl/pkg/state"
)

var (
	ctx                                                        = state.NewContext()
	onnxPath, hotwordEmbedPath, silenceNetPath, hotwordNetPath string
	err                                                        error
)

func init() {
	flag.StringVar(&silenceNetPath, "silence", silence.OnnxModelPath(), "silero-vad .onnx path")
	flag.StringVar(&hotwordNetPath, "hotword", hotword.OnnxModelPath(), "efficient-wordnet .onnx path")
	flag.StringVar(&hotwordEmbedPath, "embedding", hotword.EmbeddingsPath(), "hotword embedding .json path")
}

func main() {
	flag.Parse()
	go func() {
		if err = run(ctx); err != nil {
			log.Fatal(err)
		}
	}()
	ctx.AwaitExit()
}

func run(ctx state.Context) error {
	if err = onnx.FetchRuntime(); err != nil {
		return fmt.Errorf("path to onnx runtime is required: %w", err)
	}

	snowgirl, err := NewSnowGirl(ctx, DefaultConfig())
	if err != nil {
		return err
	}
	return snowgirl.Listen()
}
