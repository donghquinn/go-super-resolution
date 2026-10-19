package main

// Execution devices for ONNX Runtime: CUDA (NVIDIA GPUs), CoreML (Apple GPU / Neural Engine,
// used for "mps" mode) and CPU.

import (
	"fmt"
	"runtime"
	"strings"

	ort "github.com/yalue/onnxruntime_go"
	"google.golang.org/protobuf/encoding/protowire"
)

type Device string

const (
	DeviceCUDA   Device = "cuda"
	DeviceCoreML Device = "coreml"
	DeviceCPU    Device = "cpu"
)

var deviceNames = map[Device]string{DeviceCUDA: "CUDA", DeviceCoreML: "CoreML", DeviceCPU: "CPU"}

// deviceChain lists the devices to try for mode, fastest first. CPU is always the last resort.
// In auto mode, attention (transformer) models skip CoreML: DAT and HAT models measured 2-10x
// slower on CoreML than on CPU, while convolutional models ran 5-9x faster.
func deviceChain(mode InferenceMode, attention bool) []Device {
	var chain []Device
	switch mode {
	case ModeAuto:
		if runtime.GOOS != "darwin" {
			chain = append(chain, DeviceCUDA) // CUDA doesn't exist on macOS
		} else if !attention {
			chain = append(chain, DeviceCoreML)
		}
	case ModeGPU:
		chain = append(chain, DeviceCUDA)
	case ModeMPS:
		chain = append(chain, DeviceCoreML)
	}
	return append(chain, DeviceCPU)
}

// sessionOptionsFor returns session options that run on d. It fails when this ONNX Runtime
// build doesn't include the device's execution provider.
func sessionOptionsFor(d Device) (*ort.SessionOptions, error) {
	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	switch d {
	case DeviceCUDA:
		var cuda *ort.CUDAProviderOptions
		if cuda, err = ort.NewCUDAProviderOptions(); err == nil {
			err = options.AppendExecutionProviderCUDA(cuda)
			cuda.Destroy()
		}
	case DeviceCoreML:
		err = options.AppendExecutionProviderCoreMLV2(map[string]string{
			"ModelFormat":    "MLProgram",
			"MLComputeUnits": "ALL",
		})
	}
	if err != nil {
		options.Destroy()
		return nil, err
	}
	return options, nil
}

func initONNXRuntime() error {
	ort.SetSharedLibraryPath(onnxRuntimeLibrary())
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("failed to load ONNX Runtime (set SR_ONNXRUNTIME_LIB to libonnxruntime): %v", err)
	}
	return nil
}

// DeviceStatus reports whether inference can run on a device (see sr-backend -devices).
type DeviceStatus struct {
	ID        Device `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// probeDevices checks each device by creating a session for a tiny model on it, which also
// catches providers that are built in but can't start (e.g. no NVIDIA driver).
func probeDevices() ([]DeviceStatus, error) {
	if err := initONNXRuntime(); err != nil {
		return nil, err
	}
	defer ort.DestroyEnvironment()

	model := probeModel()
	var statuses []DeviceStatus
	for _, d := range []Device{DeviceCUDA, DeviceCoreML, DeviceCPU} {
		status := DeviceStatus{ID: d, Name: deviceNames[d]}
		switch {
		case d == DeviceCUDA && runtime.GOOS == "darwin":
			status.Reason = "CUDA is not supported on macOS"
		case d == DeviceCoreML && runtime.GOOS != "darwin":
			status.Reason = "CoreML is only available on macOS"
		default:
			options, err := sessionOptionsFor(d)
			if err == nil {
				var session *ort.DynamicAdvancedSession
				session, err = ort.NewDynamicAdvancedSessionWithONNXData(model, []string{"x"}, []string{"y"}, options)
				options.Destroy()
				if err == nil {
					session.Destroy()
				}
			}
			status.Available = err == nil
			if err != nil {
				status.Reason = firstLine(err.Error())
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// probeModel builds a minimal ONNX model: y = Relu(x), x and y float32 [1, 3, 8, 8].
func probeModel() []byte {
	field := func(msg []byte, num protowire.Number, value []byte) []byte {
		return protowire.AppendBytes(protowire.AppendTag(msg, num, protowire.BytesType), value)
	}
	varint := func(msg []byte, num protowire.Number, value uint64) []byte {
		return protowire.AppendVarint(protowire.AppendTag(msg, num, protowire.VarintType), value)
	}
	valueInfo := func(name string) []byte {
		var shape []byte
		for _, d := range []uint64{1, 3, 8, 8} {
			shape = field(shape, 1, varint(nil, 1, d)) // TensorShapeProto.dim.dim_value
		}
		tensor := field(varint(nil, 1, 1), 2, shape) // elem_type FLOAT, shape
		return field(field(nil, 1, []byte(name)), 2, field(nil, 1, tensor))
	}
	node := field(field(field(nil, 1, []byte("x")), 2, []byte("y")), 4, []byte("Relu"))
	graph := field(nil, 1, node)
	graph = field(graph, 2, []byte("probe"))
	graph = field(graph, 11, valueInfo("x"))
	graph = field(graph, 12, valueInfo("y"))
	model := varint(nil, 1, 8)                 // ir_version
	model = field(model, 7, graph)             // graph
	return field(model, 8, varint(nil, 2, 17)) // opset_import { version: 17 }
}

// hasAttention reports whether the model's graph contains Softmax, i.e. attention layers.
func hasAttention(model []byte) bool {
	found := false
	rewriteMessageField(model, 7, func(graph []byte) ([]byte, error) { // ModelProto.graph
		return rewriteMessageField(graph, 1, func(node []byte) ([]byte, error) { // GraphProto.node
			rewriteMessageField(node, 4, func(opType []byte) ([]byte, error) { // NodeProto.op_type
				found = found || string(opType) == "Softmax"
				return opType, nil
			})
			return node, nil
		})
	})
	return found
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	return s
}
