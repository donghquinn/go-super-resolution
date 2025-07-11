package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"gocv.io/x/gocv"
)

type SuperResolutionGUI struct {
	app           app.App
	window        fyne.Window
	originalImage *canvas.Image
	resultImage   *canvas.Image
	statusLabel   *widget.Label
	progressBar   *widget.ProgressBar
	
	// Settings
	engineSelect  *widget.Select
	modeSelect    *widget.Select
	scaleSelect   *widget.Select
	modelEntry    *widget.Entry
	widthEntry    *widget.Entry
	heightEntry   *widget.Entry
	
	// Current state
	currentImagePath string
	currentConfig    Config
	isProcessing     bool
	processor        *SuperResolutionProcessor
}

func NewSuperResolutionGUI() *SuperResolutionGUI {
	myApp := app.New()
	myApp.SetIcon(theme.DocumentCreateIcon())
	
	myWindow := myApp.NewWindow("Super Resolution - AI Image Upscaler")
	myWindow.Resize(fyne.NewSize(1200, 800))
	myWindow.CenterOnScreen()
	
	gui := &SuperResolutionGUI{
		app:    myApp,
		window: myWindow,
	}
	
	gui.setupUI()
	return gui
}

func (gui *SuperResolutionGUI) setupUI() {
	// Create image containers
	gui.originalImage = canvas.NewImageFromResource(theme.DocumentIcon())
	gui.originalImage.FillMode = canvas.ImageFillContain
	gui.originalImage.SetMinSize(fyne.NewSize(400, 300))
	
	gui.resultImage = canvas.NewImageFromResource(theme.DocumentIcon())
	gui.resultImage.FillMode = canvas.ImageFillContain
	gui.resultImage.SetMinSize(fyne.NewSize(400, 300))
	
	// Create drop zone
	dropZone := gui.createDropZone()
	
	// Create settings panel
	settingsPanel := gui.createSettingsPanel()
	
	// Create image comparison view
	imageComparison := gui.createImageComparisonView()
	
	// Create control buttons
	controlButtons := gui.createControlButtons()
	
	// Create status bar
	statusBar := gui.createStatusBar()
	
	// Layout
	leftPanel := container.NewVBox(
		widget.NewLabel("📁 Drop Image Here"),
		dropZone,
		widget.NewSeparator(),
		widget.NewLabel("⚙️ Settings"),
		settingsPanel,
		widget.NewSeparator(),
		controlButtons,
	)
	
	rightPanel := container.NewVBox(
		widget.NewLabel("🖼️ Image Comparison"),
		imageComparison,
		widget.NewSeparator(),
		statusBar,
	)
	
	// Main layout
	content := container.NewHSplit(
		leftPanel,
		rightPanel,
	)
	content.SetOffset(0.35) // 35% left panel, 65% right panel
	
	gui.window.SetContent(content)
}

func (gui *SuperResolutionGUI) createDropZone() *container.VBox {
	dropLabel := widget.NewLabel("📎 Drag and drop an image file here\n\nSupported formats: JPG, PNG, BMP, TIFF")
	dropLabel.Alignment = fyne.TextAlignCenter
	
	// Create browse button as alternative
	browseButton := widget.NewButton("📁 Browse Files", func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser) {
			if reader != nil {
				gui.loadImage(reader.URI().Path())
				reader.Close()
			}
		}, gui.window)
		
		// Set file filters for images
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png", ".bmp", ".tiff"}))
		fileDialog.Show()
	})
	
	dropZone := container.NewVBox(
		dropLabel,
		browseButton,
	)
	
	return dropZone
}

func (gui *SuperResolutionGUI) createSettingsPanel() *container.VBox {
	// Engine selection
	gui.engineSelect = widget.NewSelect([]string{"opencv", "onnx", "tensorflow", "tflite"}, nil)
	gui.engineSelect.Selected = "opencv"
	
	// Mode selection
	gui.modeSelect = widget.NewSelect([]string{"cpu", "gpu", "mps"}, nil)
	gui.modeSelect.Selected = "cpu"
	
	// Scale selection
	gui.scaleSelect = widget.NewSelect([]string{"2x", "4x", "8x", "Custom"}, nil)
	gui.scaleSelect.Selected = "4x"
	gui.scaleSelect.OnChanged = func(selected string) {
		gui.onScaleChanged(selected)
	}
	
	// Model path
	gui.modelEntry = widget.NewEntry()
	gui.modelEntry.SetText("models/Real-ESRGAN-x4plus.onnx")
	modelBrowse := widget.NewButton("Browse", gui.onBrowseModel)
	modelContainer := container.NewBorder(nil, nil, nil, modelBrowse, gui.modelEntry)
	
	// Custom dimensions
	gui.widthEntry = widget.NewEntry()
	gui.widthEntry.SetText("0")
	gui.widthEntry.Hide()
	
	gui.heightEntry = widget.NewEntry()
	gui.heightEntry.SetText("0")
	gui.heightEntry.Hide()
	
	customSizeContainer := container.NewHBox(
		widget.NewLabel("W:"),
		gui.widthEntry,
		widget.NewLabel("H:"),
		gui.heightEntry,
	)
	customSizeContainer.Hide()
	
	// Settings form
	form := container.NewVBox(
		widget.NewFormItem("Engine:", gui.engineSelect).Widget,
		widget.NewFormItem("Mode:", gui.modeSelect).Widget,
		widget.NewFormItem("Scale:", gui.scaleSelect).Widget,
		customSizeContainer,
		widget.NewFormItem("Model:", modelContainer).Widget,
	)
	
	return form
}

func (gui *SuperResolutionGUI) createImageComparisonView() *container.HSplit {
	// Original image container
	originalContainer := container.NewVBox(
		widget.NewLabelWithStyle("Original", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		gui.originalImage,
	)
	
	// Result image container
	resultContainer := container.NewVBox(
		widget.NewLabelWithStyle("Super Resolution", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		gui.resultImage,
	)
	
	// Split container
	split := container.NewHSplit(originalContainer, resultContainer)
	split.SetOffset(0.5)
	
	return split
}

func (gui *SuperResolutionGUI) createControlButtons() *container.HBox {
	processButton := widget.NewButton("🚀 Process Image", gui.onProcessImage)
	processButton.Importance = widget.HighImportance
	
	saveButton := widget.NewButton("💾 Save Result", gui.onSaveResult)
	saveButton.Disable()
	
	clearButton := widget.NewButton("🗑️ Clear", gui.onClear)
	
	return container.NewHBox(
		processButton,
		saveButton,
		clearButton,
	)
}

func (gui *SuperResolutionGUI) createStatusBar() *container.VBox {
	gui.statusLabel = widget.NewLabel("Ready")
	gui.statusLabel.TextStyle = fyne.TextStyle{Monospace: true}
	
	gui.progressBar = widget.NewProgressBar()
	gui.progressBar.Hide()
	
	return container.NewVBox(
		gui.statusLabel,
		gui.progressBar,
	)
}

func (gui *SuperResolutionGUI) onScaleChanged(selected string) {
	if selected == "Custom" {
		gui.widthEntry.Show()
		gui.heightEntry.Show()
		gui.widthEntry.Parent().Show()
	} else {
		gui.widthEntry.Hide()
		gui.heightEntry.Hide()
		gui.widthEntry.Parent().Hide()
	}
}

func (gui *SuperResolutionGUI) onBrowseModel() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser) {
		if reader != nil {
			gui.modelEntry.SetText(reader.URI().Path())
			reader.Close()
		}
	}, gui.window)
	
	// Set file filters
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".onnx", ".pb", ".tflite"}))
	fileDialog.Show()
}

func (gui *SuperResolutionGUI) onProcessImage() {
	if gui.currentImagePath == "" {
		dialog.ShowError(fmt.Errorf("please select an image first"), gui.window)
		return
	}
	
	if gui.isProcessing {
		return
	}
	
	go gui.processImageAsync()
}

func (gui *SuperResolutionGUI) processImageAsync() {
	gui.isProcessing = true
	gui.statusLabel.SetText("Processing...")
	gui.progressBar.Show()
	
	// Create config
	config := gui.createConfig()
	
	// Create processor
	processor, err := NewSuperResolutionProcessor(config)
	if err != nil {
		gui.showError(fmt.Errorf("failed to create processor: %v", err))
		gui.isProcessing = false
		gui.progressBar.Hide()
		return
	}
	defer processor.Close()
	
	// Process image
	err = processor.ProcessImage()
	if err != nil {
		gui.showError(fmt.Errorf("failed to process image: %v", err))
		gui.isProcessing = false
		gui.progressBar.Hide()
		return
	}
	
	// Load result image
	gui.loadResultImage(config.OutputPath)
	
	gui.statusLabel.SetText("Processing completed!")
	gui.progressBar.Hide()
	gui.isProcessing = false
}

func (gui *SuperResolutionGUI) createConfig() Config {
	// Parse scale
	var scaleFactor int
	var targetWidth, targetHeight int
	
	switch gui.scaleSelect.Selected {
	case "2x":
		scaleFactor = 2
	case "4x":
		scaleFactor = 4
	case "8x":
		scaleFactor = 8
	case "Custom":
		scaleFactor = 1
		// Parse custom dimensions
		if gui.widthEntry.Text != "" && gui.widthEntry.Text != "0" {
			fmt.Sscanf(gui.widthEntry.Text, "%d", &targetWidth)
		}
		if gui.heightEntry.Text != "" && gui.heightEntry.Text != "0" {
			fmt.Sscanf(gui.heightEntry.Text, "%d", &targetHeight)
		}
	}
	
	// Generate output path
	outputPath := gui.generateOutputPath()
	
	return Config{
		InputPath:    gui.currentImagePath,
		OutputPath:   outputPath,
		ModelPath:    gui.modelEntry.Text,
		Mode:         InferenceMode(gui.modeSelect.Selected),
		Engine:       UpscaleEngine(gui.engineSelect.Selected),
		ScaleFactor:  scaleFactor,
		TargetWidth:  targetWidth,
		TargetHeight: targetHeight,
	}
}

func (gui *SuperResolutionGUI) generateOutputPath() string {
	if gui.currentImagePath == "" {
		return ""
	}
	
	dir := filepath.Dir(gui.currentImagePath)
	base := filepath.Base(gui.currentImagePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	
	suffix := fmt.Sprintf("_sr_%s_%s", gui.engineSelect.Selected, gui.scaleSelect.Selected)
	
	return filepath.Join(dir, name+suffix+ext)
}

func (gui *SuperResolutionGUI) loadResultImage(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}
	
	resource, err := fyne.LoadResourceFromPath(path)
	if err != nil {
		log.Printf("Failed to load result image: %v", err)
		return
	}
	
	gui.resultImage.Resource = resource
	gui.resultImage.Refresh()
}

func (gui *SuperResolutionGUI) onSaveResult() {
	// Save dialog
	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser) {
		if writer != nil {
			// Copy result file
			// Implementation depends on current result path
			writer.Close()
		}
	}, gui.window)
	
	saveDialog.SetFileName("super_resolution_result.jpg")
	saveDialog.Show()
}

func (gui *SuperResolutionGUI) onClear() {
	gui.originalImage.Resource = theme.DocumentIcon()
	gui.resultImage.Resource = theme.DocumentIcon()
	gui.originalImage.Refresh()
	gui.resultImage.Refresh()
	gui.currentImagePath = ""
	gui.statusLabel.SetText("Ready")
}

func (gui *SuperResolutionGUI) showError(err error) {
	dialog.ShowError(err, gui.window)
	gui.statusLabel.SetText("Error: " + err.Error())
}

func (gui *SuperResolutionGUI) loadImage(path string) {
	gui.currentImagePath = path
	
	resource, err := fyne.LoadResourceFromPath(path)
	if err != nil {
		gui.showError(fmt.Errorf("failed to load image: %v", err))
		return
	}
	
	gui.originalImage.Resource = resource
	gui.originalImage.Refresh()
	
	// Get image dimensions
	img := gocv.IMRead(path, gocv.IMReadColor)
	if !img.Empty() {
		gui.statusLabel.SetText(fmt.Sprintf("Loaded: %dx%d", img.Cols(), img.Rows()))
		img.Close()
	}
}

func (gui *SuperResolutionGUI) Run() {
	gui.window.ShowAndRun()
}

// runCLI runs the original CLI version
func runCLI() {
	// This is the original main function from main.go
	// We'll integrate this properly
}