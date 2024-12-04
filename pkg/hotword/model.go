package hotword

import (
	"fmt"

	"github.com/algo-boyz/snowgirl/pkg/state"
	onnx "github.com/yalue/onnxruntime_go"
	"go.uber.org/multierr"
)

var (
	useCoreML  bool
	sampleRate = 16000 // todo make 8000 an option
)

type Model struct {
	networkPath string
	Options     *onnx.SessionOptions
	InputInfo   []onnx.InputOutputInfo
	OutputInfo  []onnx.InputOutputInfo
	Embeddings  [][]float32
}

func NewModel(ctx state.Context, onnxPath, hotwordNetPath string, embeddings [][]float32) (m *Model, err error) {
	onnx.SetSharedLibraryPath(onnxPath)
	if err = onnx.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to init onnx lib: %w", err)
	}
	inputs, outputs, err := onnx.GetInputOutputInfo(hotwordNetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get net info for %s: %w", hotwordNetPath, err)
	}
	printInfo(hotwordNetPath, inputs, outputs)
	options, err := getOptions()
	m = &Model{
		InputInfo:   inputs,
		OutputInfo:  outputs,
		Options:     options,
		Embeddings:  embeddings,
		networkPath: hotwordNetPath,
	}
	go ctx.Defer(func() {
		if err = m.Destroy(); err != nil {
			fmt.Printf("failed to destroy eff-word-net: %s\n", err)
		}
		fmt.Println("eff-word-net exit")
	})
	return m, err
}

func (m *Model) Destroy() error {
	return multierr.Combine(m.Options.Destroy(), onnx.DestroyEnvironment())
}

func getOptions() (options *onnx.SessionOptions, err error) {
	options, err = onnx.NewSessionOptions()
	if err != nil {
		err = fmt.Errorf("failed to create onnx session options: %w", err)
		return nil, err
	}
	if useCoreML {
		if err = options.AppendExecutionProviderCoreML(0); err != nil {
			err = fmt.Errorf("failed to enable CoreML: %w", err)
			return nil, err
		}
	}
	return options, nil
}

func (m *Model) ProcessFrame(frame []float32) (distances []float32, err error) {
	input, err := onnx.NewTensor(m.InputInfo[0].Dimensions, frame)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}
	defer func() {
		if err != nil {
			err = multierr.Combine(err, input.Destroy())
		}
	}()
	output, err := onnx.NewEmptyTensor[float32](m.OutputInfo[0].Dimensions)
	if err != nil {
		err = fmt.Errorf("failed to create output tensor: %w", err)
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Combine(err, output.Destroy())
		}
	}()
	session, err := onnx.NewAdvancedSession(
		m.networkPath,
		[]string{m.InputInfo[0].Name},
		[]string{m.OutputInfo[0].Name},
		[]onnx.ArbitraryTensor{input},
		[]onnx.ArbitraryTensor{output},
		m.Options,
	)
	if err != nil {
		err = fmt.Errorf("failed to create onnx session: %w", err)
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Combine(err, session.Destroy())
		}
	}()
	if err = session.Run(); err != nil {
		err = fmt.Errorf("failed to run eff-word net: %w", err)
		return nil, err
	}
	return output.GetData(), nil
}

func printInfo(hotwordNetPath string, inputs, outputs []onnx.InputOutputInfo) {
	fmt.Printf("%s:", hotwordNetPath)
	for _, i := range inputs {
		fmt.Printf("	%s:%v ->", i.Name, i.Dimensions)
	}
	for _, o := range outputs {
		fmt.Printf("	%s\n", o.Name)
	}
}
