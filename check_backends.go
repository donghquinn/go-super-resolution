package main

import (
	"fmt"

	"gocv.io/x/gocv"
)

func main() {
	fmt.Println("=== OpenCV Version ===")
	fmt.Printf("OpenCV version: %s\n", gocv.OpenCVVersion())
	
	// Check available backends
	fmt.Println("\n=== Available Backends ===")
	backends := []struct {
		name string
		id   gocv.NetBackendType
	}{
		{"OpenCV", gocv.NetBackendOpenCV},
		{"Default", gocv.NetBackendDefault},
		{"Halide", gocv.NetBackendHalide},
		{"OpenVINO", gocv.NetBackendOpenVINO},
		{"VKCOM", gocv.NetBackendVKCOM},
		{"CUDA", gocv.NetBackendCUDA},
	}
	
	for _, backend := range backends {
		fmt.Printf("%s: %d\n", backend.name, backend.id)
	}
	
	fmt.Println("\n=== Available Targets ===")
	targets := []struct {
		name string
		id   gocv.NetTargetType
	}{
		{"CPU", gocv.NetTargetCPU},
		{"Vulkan", gocv.NetTargetVulkan},
		{"FPGA", gocv.NetTargetFPGA},
		{"CUDA", gocv.NetTargetCUDA},
		{"CUDA FP16", gocv.NetTargetCUDAFP16},
	}
	
	for _, target := range targets {
		fmt.Printf("%s: %d\n", target.name, target.id)
	}
	
	fmt.Println("\n=== Testing Network Creation ===")
	// Test if we can create a simple network
	fmt.Println("Testing basic DNN functionality...")
}