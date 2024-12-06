package silence

import (
	"fmt"

	"github.com/algo-boyz/snowgirl/pkg/audio"
	"github.com/algo-boyz/snowgirl/pkg/state"
	onnx "github.com/yalue/onnxruntime_go"
)

type Model struct {
}

type Config struct {
	ModelPath            string
	SampleRate           int
	Threshold            float32
	MinSilenceDurationMs int
	SpeechPadMs          int
}

func DefaultConfig() Config {
	return Config{
		ModelPath:            OnnxModelPath(),
		SampleRate:           16000,
		Threshold:            0.5,
		MinSilenceDurationMs: 100,
		SpeechPadMs:          30,
	}
}

func OnnxModelPath() string {
	return "model/silence/silero_vad_16k.onnx"
}

func NewModel(ctx state.Context, cfg Config, stream *audio.MicStream) (*Model, error) {
	// vad, err := speech.NewDetector(speech.DetectorConfig{
	// 	ModelPath:            cfg.ModelPath,
	// 	SampleRate:           cfg.SampleRate,
	// 	Threshold:            cfg.Threshold,
	// 	MinSilenceDurationMs: cfg.MinSilenceDurationMs,
	// 	SpeechPadMs:          cfg.SpeechPadMs,
	// })
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create speech detector: %w", err)
	// }
	// inputs, outputs, err := onnx.GetInputOutputInfo(cfg.ModelPath)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get net info for %s: %w", cfg.ModelPath, err)
	// }
	// printInfo(cfg.ModelPath, inputs, outputs)
	go ctx.Defer(func() {
		// if err = vad.Destroy(); err != nil {
		// 	fmt.Printf("failed to destroy speech detector: %s\n", err)
		// }
		fmt.Println("speech detector exit")
	})
	return nil, nil
}

func (d *Model) ProcessFrame(frame []float32) ([]any, error) {
	// segments, err := d.Detect(frame)
	// if err != nil {
	// 	return nil, fmt.Errorf("d.Detect: %w", err)
	// }
	// for _, s := range segments {
	// 	log.Printf("speech starts at %0.2fs", s.SpeechStartAt)
	// 	if s.SpeechEndAt > 0 {
	// 		log.Printf("speech ends at %0.2fs", s.SpeechEndAt)
	// 	}
	// }
	return nil, nil
}

func printInfo(silenceNetPath string, inputs, outputs []onnx.InputOutputInfo) {
	fmt.Printf("%s:", silenceNetPath)
	for _, i := range inputs {
		fmt.Printf("	%s:%v ->", i.Name, i.Dimensions)
	}
	for _, o := range outputs {
		fmt.Printf("	%s\n", o.Name)
	}
}
