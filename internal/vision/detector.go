package vision

import (
"fmt"
"image"
"image/color"
"image/draw"
"image/jpeg"
"image/png"
"os"
"path/filepath"
"strings"
)

// VehicleDetector handles vehicle detection in images
type VehicleDetector struct {
	ModelPath string
	Threshold float64
}

// NewVehicleDetector creates a new vehicle detector instance
func NewVehicleDetector(modelPath string, threshold float64) *VehicleDetector {
	return &VehicleDetector{
		ModelPath: modelPath,
		Threshold: threshold,
	}
}

// DetectVehicle detects vehicles in an image and returns bounding boxes
func (vd *VehicleDetector) DetectVehicle(imagePath string) ([]BoundingBox, error) {
	img, err := loadImage(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Mock detection - replace with real detection
	mockBoxes := []BoundingBox{
		{
			X:      width / 4,
			Y:      height / 3,
			Width:  width / 2,
			Height: height / 2,
			Label:  "vehicle",
			Score:  0.95,
		},
	}

	return mockBoxes, nil
}

// DrawBoundingBoxes draws bounding boxes on image
func (vd *VehicleDetector) DrawBoundingBoxes(imagePath string, boxes []BoundingBox, outputPath string) error {
	img, err := loadImage(imagePath)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	for _, box := range boxes {
		drawRect(rgba, box.X, box.Y, box.Width, box.Height, red, 3)
	}

	return saveImage(rgba, outputPath)
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Decode(file)
	case ".png":
		return png.Decode(file)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}
}

func saveImage(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 95})
	case ".png":
		return png.Encode(file, img)
	default:
		return fmt.Errorf("unsupported image format: %s", ext)
	}
}

func drawRect(img *image.RGBA, x, y, width, height int, col color.Color, thickness int) {
	for t := 0; t < thickness; t++ {
		for i := x; i < x+width; i++ {
			if i >= 0 && i < img.Bounds().Dx() && y+t >= 0 && y+t < img.Bounds().Dy() {
				img.Set(i, y+t, col)
			}
		}
		for i := x; i < x+width; i++ {
			if i >= 0 && i < img.Bounds().Dx() && y+height-t >= 0 && y+height-t < img.Bounds().Dy() {
				img.Set(i, y+height-t, col)
			}
		}
		for i := y; i < y+height; i++ {
			if x+t >= 0 && x+t < img.Bounds().Dx() && i >= 0 && i < img.Bounds().Dy() {
				img.Set(x+t, i, col)
			}
		}
		for i := y; i < y+height; i++ {
			if x+width-t >= 0 && x+width-t < img.Bounds().Dx() && i >= 0 && i < img.Bounds().Dy() {
				img.Set(x+width-t, i, col)
			}
		}
	}
}
