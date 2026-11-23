package inference

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

var (
	envMu       sync.Mutex
	initialized bool
)

// Detector wraps the ONNX Runtime session and coco labels.
type Detector struct {
	session    *ort.DynamicAdvancedSession
	inputName  string
	outputName string
	labels     []string
	modelPath  string
}

// InitONNXEnvironment initializes the ONNX shared library once per process.
func InitONNXEnvironment(libPath string) error {
	envMu.Lock()
	defer envMu.Unlock()
	if initialized || ort.IsInitialized() {
		return nil
	}
	ort.SetSharedLibraryPath(libPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("init onnx environment (%s): %w", libPath, err)
	}
	initialized = true
	return nil
}

// NewDetector initializes a dynamic ONNX session for YOLOX inference.
func NewDetector(modelPath, classesPath, libPath string) (*Detector, error) {
	if err := InitONNXEnvironment(libPath); err != nil {
		return nil, err
	}

	labels, err := loadLabels(classesPath)
	if err != nil {
		return nil, fmt.Errorf("load classes: %w", err)
	}

	inputs, outputs, err := ort.GetInputOutputInfo(modelPath)
	if err != nil {
		return nil, fmt.Errorf("get input/output info (%s): %w", modelPath, err)
	}
	if len(inputs) != 1 || len(outputs) != 1 {
		return nil, fmt.Errorf("expected 1 input and 1 output, got %d/%d", len(inputs), len(outputs))
	}

	session, err := ort.NewDynamicAdvancedSession(modelPath, []string{inputs[0].Name}, []string{outputs[0].Name}, nil)
	if err != nil {
		return nil, fmt.Errorf("create onnx session: %w", err)
	}

	return &Detector{
		session:    session,
		inputName:  inputs[0].Name,
		outputName: outputs[0].Name,
		labels:     labels,
		modelPath:  modelPath,
	}, nil
}

func (d *Detector) Detect(img image.Image, opts InferenceOptions) ([]Detection, error) {
	if d == nil || d.session == nil {
		return nil, fmt.Errorf("detector session is not initialized")
	}

	data, prep, err := PreprocessImage(img, 640, 640)
	if err != nil {
		return nil, err
	}

	shape := ort.NewShape(1, 3, 640, 640)
	tensor, err := ort.NewTensor(shape, data)
	if err != nil {
		return nil, fmt.Errorf("create tensor: %w", err)
	}
	defer tensor.Destroy()

	outputs := []ort.Value{nil}
	if err := d.session.Run([]ort.Value{tensor}, outputs); err != nil {
		return nil, fmt.Errorf("run onnx session: %w", err)
	}
	if outputs[0] == nil {
		return nil, fmt.Errorf("session returned empty output")
	}
	defer outputs[0].Destroy()

	outputTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output tensor type")
	}

	return DecodeYOLOXOutput(outputTensor.GetData(), d.labels, prep, opts)
}

func (d *Detector) Close() error {
	if d != nil && d.session != nil {
		err := d.session.Destroy()
		d.session = nil
		return err
	}
	return nil
}

func loadLabels(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var labels []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			labels = append(labels, text)
		}
	}
	return labels, scanner.Err()
}

// ResolveRuntimePaths resolves the model, classes and onnx library paths.
func ResolveRuntimePaths(pluginRoot string) (string, string, string) {
	candidates := []string{
		pluginRoot,
		".",
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Dir(exe)
		for i := 0; i < 6; i++ {
			candidates = append(candidates, p)
			p = filepath.Dir(p)
		}
	}
	for _, dir := range candidates {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			absDir = dir
		}
		m := filepath.Join(absDir, "runtime", "models", "yolox_s.onnx")
		c := filepath.Join(absDir, "runtime", "models", "coco_classes.txt")
		l := filepath.Join(absDir, "runtime", "lib", "libonnxruntime.so")
		if fileExists(m) && fileExists(c) && fileExists(l) {
			return m, c, l
		}
	}
	// Fallback to absolute relative to cwd
	cwd, _ := os.Getwd()
	absM, _ := filepath.Abs(filepath.Join(cwd, "runtime", "models", "yolox_s.onnx"))
	absC, _ := filepath.Abs(filepath.Join(cwd, "runtime", "models", "coco_classes.txt"))
	absL, _ := filepath.Abs(filepath.Join(cwd, "runtime", "lib", "libonnxruntime.so"))
	return absM, absC, absL
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

