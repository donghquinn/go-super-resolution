package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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

type GUI struct {
	app           fyne.App
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
	customContainer *fyne.Container
	
	// Buttons
	processButton *widget.Button
	saveButton    *widget.Button
	
	// Current state
	currentImagePath string
	currentResultPath string
	isProcessing     bool
}

func NewGUI() *GUI {
	myApp := app.NewWithID("com.example.super-resolution")
	myApp.SetIcon(theme.DocumentCreateIcon())
	
	myWindow := myApp.NewWindow("Super Resolution - AI Image Upscaler")
	myWindow.Resize(fyne.NewSize(1200, 800))
	myWindow.CenterOnScreen()
	
	gui := &GUI{
		app:    myApp,
		window: myWindow,
	}
	
	gui.setupUI()
	return gui
}

func (gui *GUI) setupUI() {
	// Create image containers
	gui.originalImage = canvas.NewImageFromResource(theme.DocumentIcon())
	gui.originalImage.FillMode = canvas.ImageFillContain
	gui.originalImage.SetMinSize(fyne.NewSize(400, 300))
	
	gui.resultImage = canvas.NewImageFromResource(theme.DocumentIcon())
	gui.resultImage.FillMode = canvas.ImageFillContain
	gui.resultImage.SetMinSize(fyne.NewSize(400, 300))
	
	// Create UI components
	dropZone := gui.createDropZone()
	settingsPanel := gui.createSettingsPanel()
	imageComparison := gui.createImageComparisonView()
	controlButtons := gui.createControlButtons()
	statusBar := gui.createStatusBar()
	
	// Layout
	leftPanel := container.NewVBox(
		widget.NewCard("", "📁 Select Image", dropZone),
		widget.NewCard("", "⚙️ Settings", settingsPanel),
		controlButtons,
	)
	
	rightPanel := container.NewVBox(
		widget.NewCard("", "🖼️ Image Comparison", imageComparison),
		statusBar,
	)
	
	// Main layout
	content := container.NewHSplit(leftPanel, rightPanel)
	content.SetOffset(0.35)
	
	gui.window.SetContent(content)
}

func (gui *GUI) createDropZone() *fyne.Container {
	dropLabel := widget.NewLabel("📎 Drag and drop an image file here\n\nSupported formats: JPG, PNG, BMP, TIFF")
	dropLabel.Alignment = fyne.TextAlignCenter
	
	browseButton := widget.NewButton("📁 Browse Files", func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				gui.showError(err)
				return
			}
			if reader != nil {
				gui.loadImage(reader.URI().Path())
				reader.Close()
			}
		}, gui.window)
		
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png", ".bmp", ".tiff"}))
		fileDialog.Show()
	})
	
	return container.NewVBox(dropLabel, browseButton)
}

func (gui *GUI) createSettingsPanel() *fyne.Container {
	// Engine selection
	gui.engineSelect = widget.NewSelect([]string{"opencv", "onnx", "tensorflow", "tflite"}, nil)
	gui.engineSelect.Selected = "opencv"
	
	// Mode selection
	gui.modeSelect = widget.NewSelect([]string{"cpu", "gpu", "mps"}, nil)
	gui.modeSelect.Selected = "cpu"
	
	// Scale selection
	gui.scaleSelect = widget.NewSelect([]string{"2x", "4x", "8x", "Custom"}, nil)
	gui.scaleSelect.Selected = "4x"
	gui.scaleSelect.OnChanged = gui.onScaleChanged
	
	// Model path
	gui.modelEntry = widget.NewEntry()
	gui.modelEntry.SetText("models/Real-ESRGAN-x4plus.onnx")
	modelBrowse := widget.NewButton("Browse", gui.onBrowseModel)
	modelContainer := container.NewBorder(nil, nil, nil, modelBrowse, gui.modelEntry)
	
	// Custom dimensions
	gui.widthEntry = widget.NewEntry()
	gui.widthEntry.SetText("0")
	gui.heightEntry = widget.NewEntry()
	gui.heightEntry.SetText("0")
	
	gui.customContainer = container.NewHBox(
		widget.NewLabel("W:"), gui.widthEntry,
		widget.NewLabel("H:"), gui.heightEntry,
	)
	gui.customContainer.Hide()
	
	// Settings form
	return container.NewVBox(
		container.NewHBox(widget.NewLabel("Engine:"), gui.engineSelect),
		container.NewHBox(widget.NewLabel("Mode:"), gui.modeSelect),
		container.NewHBox(widget.NewLabel("Scale:"), gui.scaleSelect),
		gui.customContainer,
		container.NewVBox(widget.NewLabel("Model:"), modelContainer),
	)
}

func (gui *GUI) createImageComparisonView() *container.Split {
	originalContainer := container.NewVBox(
		widget.NewLabelWithStyle("Original", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		gui.originalImage,
	)
	
	resultContainer := container.NewVBox(
		widget.NewLabelWithStyle("Super Resolution", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		gui.resultImage,
	)
	
	split := container.NewHSplit(originalContainer, resultContainer)
	split.SetOffset(0.5)
	return split
}

func (gui *GUI) createControlButtons() *fyne.Container {
	gui.processButton = widget.NewButton("🚀 Process Image", gui.onProcessImage)
	gui.processButton.Importance = widget.HighImportance
	
	gui.saveButton = widget.NewButton("💾 Save Result", gui.onSaveResult)
	gui.saveButton.Disable()
	
	clearButton := widget.NewButton("🗑️ Clear", gui.onClear)
	
	return container.NewHBox(gui.processButton, gui.saveButton, clearButton)
}

func (gui *GUI) createStatusBar() *fyne.Container {
	gui.statusLabel = widget.NewLabel("Ready - Select an image to get started")
	gui.statusLabel.TextStyle = fyne.TextStyle{Monospace: true}
	
	gui.progressBar = widget.NewProgressBar()
	gui.progressBar.Hide()
	
	return container.NewVBox(gui.statusLabel, gui.progressBar)
}

func (gui *GUI) onScaleChanged(selected string) {
	if selected == "Custom" {
		gui.customContainer.Show()
	} else {
		gui.customContainer.Hide()
	}
}

func (gui *GUI) onBrowseModel() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			gui.showError(err)
			return
		}
		if reader != nil {
			gui.modelEntry.SetText(reader.URI().Path())
			reader.Close()
		}
	}, gui.window)
	
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".onnx", ".pb", ".tflite"}))
	fileDialog.Show()
}

func (gui *GUI) onProcessImage() {
	if gui.currentImagePath == "" {
		dialog.ShowInformation("No Image", "Please select an image first", gui.window)
		return
	}
	
	if gui.isProcessing {
		return
	}
	
	// Start processing
	gui.processImageAsync()
}

func (gui *GUI) processImageAsync() {
	// Set initial state
	gui.isProcessing = true
	gui.processButton.Disable()
	gui.statusLabel.SetText("Processing...")
	gui.progressBar.Show()
	gui.progressBar.SetValue(0.3)
	
	// Create config
	config := gui.createConfig()
	
	// Start background processing
	go func() {
		// Create processor
		processor, err := NewSuperResolutionProcessor(config)
		if err != nil {
			// Schedule UI update
			time.AfterFunc(1*time.Millisecond, func() {
				gui.app.SendNotification(fyne.NewNotification("Error", fmt.Sprintf("Failed to create processor: %v", err)))
				gui.statusLabel.SetText("Processing failed")
				gui.finishProcessing()
			})
			return
		}
		defer processor.Close()
		
		// Process the image
		err = processor.ProcessImage()
		if err != nil {
			// Schedule UI update
			time.AfterFunc(1*time.Millisecond, func() {
				gui.app.SendNotification(fyne.NewNotification("Error", fmt.Sprintf("Processing failed: %v", err)))
				gui.statusLabel.SetText("Processing failed")
				gui.finishProcessing()
			})
			return
		}
		
		// Schedule successful completion UI update
		time.AfterFunc(1*time.Millisecond, func() {
			gui.currentResultPath = config.OutputPath
			gui.loadResultImage(config.OutputPath)
			gui.progressBar.SetValue(1.0)
			gui.statusLabel.SetText("Processing completed successfully!")
			gui.saveButton.Enable()
			gui.finishProcessing()
		})
	}()
}

func (gui *GUI) finishProcessing() {
	gui.isProcessing = false
	gui.processButton.Enable()
	gui.progressBar.Hide()
}

func (gui *GUI) createConfig() Config {
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
		fmt.Sscanf(gui.widthEntry.Text, "%d", &targetWidth)
		fmt.Sscanf(gui.heightEntry.Text, "%d", &targetHeight)
	}
	
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

func (gui *GUI) generateOutputPath() string {
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

func (gui *GUI) loadResultImage(path string) {
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

func (gui *GUI) onSaveResult() {
	if gui.currentResultPath == "" {
		dialog.ShowInformation("No Result", "No processed image to save", gui.window)
		return
	}
	
	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			gui.showError(err)
			return
		}
		if writer != nil {
			gui.copyFile(gui.currentResultPath, writer.URI().Path())
			writer.Close()
			dialog.ShowInformation("Saved", "Image saved successfully!", gui.window)
		}
	}, gui.window)
	
	saveDialog.SetFileName("super_resolution_result.jpg")
	saveDialog.Show()
}

func (gui *GUI) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (gui *GUI) onClear() {
	gui.originalImage.Resource = theme.DocumentIcon()
	gui.resultImage.Resource = theme.DocumentIcon()
	gui.originalImage.Refresh()
	gui.resultImage.Refresh()
	gui.currentImagePath = ""
	gui.currentResultPath = ""
	gui.statusLabel.SetText("Ready - Select an image to get started")
	gui.saveButton.Disable()
}

func (gui *GUI) showError(err error) {
	dialog.ShowError(err, gui.window)
	gui.statusLabel.SetText("Error: " + err.Error())
}

func (gui *GUI) loadImage(path string) {
	gui.currentImagePath = path
	
	resource, err := fyne.LoadResourceFromPath(path)
	if err != nil {
		gui.showError(fmt.Errorf("failed to load image: %v", err))
		return
	}
	
	gui.originalImage.Resource = resource
	gui.originalImage.Refresh()
	
	img := gocv.IMRead(path, gocv.IMReadColor)
	if !img.Empty() {
		gui.statusLabel.SetText(fmt.Sprintf("Loaded: %s (%dx%d)", filepath.Base(path), img.Cols(), img.Rows()))
		img.Close()
	}
}

func (gui *GUI) Run() {
	gui.window.ShowAndRun()
}

func main() {
	gui := NewGUI()
	gui.Run()
}