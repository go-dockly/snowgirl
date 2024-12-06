package main

import (
	"fmt"
	"time"

	"github.com/algo-boyz/snowgirl/pkg/audio"
	"github.com/algo-boyz/snowgirl/pkg/hotword"
	"github.com/algo-boyz/snowgirl/pkg/onnx"
	"github.com/algo-boyz/snowgirl/pkg/silence"
	"github.com/algo-boyz/snowgirl/pkg/state"
)

type Config struct {
	OnnxPath, SilenceNetPath, HotwordNetPath, HotwordEmbedPath string
}

func DefaultConfig() Config {
	return Config{
		OnnxPath:         onnx.LibPath(),
		HotwordNetPath:   hotword.OnnxModelPath(),
		HotwordEmbedPath: hotword.EmbeddingsPath(),
	}
}

type SnowGirl struct {
	cfg          Config
	ctx          state.Context
	silenceModel *silence.Model
	hotwordModel *hotword.Model
	logMelSpec   *hotword.LogMelSpectrogram
	mic          *audio.MicStream
}

func NewSnowGirl(ctx state.Context, cfg Config) (*SnowGirl, error) {
	embeddings, err := hotword.LoadEmbeddings(cfg.HotwordEmbedPath)
	if err != nil {
		return nil, err
	}
	hotwordModel, err := hotword.NewModel(ctx, cfg.OnnxPath, cfg.HotwordNetPath, embeddings)
	if err != nil {
		return nil, err
	}
	silenceModel, err := silence.NewModel(ctx, silence.DefaultConfig(), nil)
	if err != nil {
		return nil, err
	}
	stream, err := audio.NewMicStream(ctx, 1.5, 0.75)
	if err != nil {
		return nil, fmt.Errorf("failed to create mic stream: %w", err)
	}
	if err = stream.Start(); err != nil {
		return nil, fmt.Errorf("failed to start mic stream: %w", err)
	}
	return &SnowGirl{
		ctx:          ctx,
		cfg:          cfg,
		mic:          stream,
		silenceModel: silenceModel,
		hotwordModel: hotwordModel,
		logMelSpec:   hotword.DefaultLogMelSpectrogram(),
	}, nil
}

func (s *SnowGirl) Listen() (err error) {
	time.Sleep(time.Millisecond * 500)
	audioChan := s.mic.Subscribe()
	defer s.mic.Unsubscribe(audioChan)
	for frame := range audioChan {
		normalized, err := s.logMelSpec.AudioToVector(frame)
		if err != nil {
			return fmt.Errorf("logMelSpec.AudioToVector: %w", err)
		}
		output, err := s.hotwordModel.ProcessFrame(normalized)
		if err != nil {
			return fmt.Errorf("model.ProcessFrame: %w", err)
		}
		fmt.Println("mic_frame: ", len(frame), "normalized: ", len(normalized))
		var detected string
		var confidence = s.hotwordModel.ScoreVector(output)
		if confidence > 0.9 {
			detected = "DETECTED!"
		}
		fmt.Printf("Confidence: %f %s\n", confidence, detected)
	}
	return nil
}
