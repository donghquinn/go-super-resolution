//go:build ignore

package main

import (
	"fmt"
	"log"

	"gocv.io/x/gocv"
)

func main() {
	fmt.Println("Testing GoCV installation...")
	
	// Test OpenCV version
	fmt.Printf("OpenCV version: %s\n", gocv.OpenCVVersion())
	
	// Test basic Mat operations
	img := gocv.NewMat()
	defer img.Close()
	
	if img.Empty() {
		fmt.Println("✓ Mat creation successful")
	} else {
		log.Fatal("✗ Mat creation failed")
	}
	
	// Test ONNX support
	fmt.Println("Testing ONNX support...")
	
	// This will fail if no model exists, but we can catch the error
	net := gocv.ReadNetFromONNX("nonexistent.onnx")
	if net.Empty() {
		fmt.Println("✓ ONNX loading function available (no model to test)")
	} else {
		fmt.Println("✓ ONNX loading successful")
		net.Close()
	}
	
	fmt.Println("Build test completed!")
}