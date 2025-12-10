package vision

import (
	"fmt"
	"log"
)

// DimensionService handles vehicle dimension calculations
type DimensionService struct {
	Detector    *VehicleDetector
	Calibration *CameraCalibration
}

// NewDimensionService creates a new dimension service
func NewDimensionService(modelPath string, threshold float64) *DimensionService {
	return &DimensionService{
		Detector:    NewVehicleDetector(modelPath, threshold),
		Calibration: NewCameraCalibration(),
	}
}

// SetCalibration sets the camera calibration parameters
func (ds *DimensionService) SetCalibration(calibration *CameraCalibration) error {
	if err := calibration.Validate(); err != nil {
		return fmt.Errorf("invalid calibration: %w", err)
	}
	ds.Calibration = calibration
	return nil
}

// ProcessImage processes an image and returns vehicle dimensions
func (ds *DimensionService) ProcessImage(imagePath string) ([]VehicleDimensions, error) {
	log.Printf("[DIMENSION] Processing image: %s", imagePath)

	// Detect vehicles in the image
	boxes, err := ds.Detector.DetectVehicle(imagePath)
	if err != nil {
		return nil, fmt.Errorf("vehicle detection failed: %w", err)
	}

	if len(boxes) == 0 {
		log.Printf("[DIMENSION] No vehicles detected in image")
		return []VehicleDimensions{}, nil
	}

	log.Printf("[DIMENSION] Detected %d vehicle(s)", len(boxes))

	// Calculate dimensions for each detected vehicle
	var results []VehicleDimensions
	for i, box := range boxes {
		log.Printf("[DIMENSION] Calculating dimensions for vehicle %d (score: %.2f)", i+1, box.Score)

		dims, err := ds.Calibration.CalculateGroundDimensions(box)
		if err != nil {
			log.Printf("[DIMENSION] Warning: Failed to calculate dimensions for vehicle %d: %v", i+1, err)
			continue
		}

		// Set additional metadata
		dims.ImagePath = imagePath

		// Adjust confidence based on detection score
		dims.Confidence = dims.Confidence * box.Score

		results = append(results, *dims)

		log.Printf("[DIMENSION] Vehicle %d dimensions: L=%.2fm W=%.2fm H=%.2fm (distance: %.2fm, confidence: %.2f)",
			i+1, dims.LengthMeters, dims.WidthMeters, dims.HeightMeters, dims.DistanceMeters, dims.Confidence)
	}

	return results, nil
}

// ClassifyVehicle classifies vehicle type based on dimensions
func (ds *DimensionService) ClassifyVehicle(dims VehicleDimensions) VehicleClass {
	// Vehicle classification based on typical dimensions
	// These thresholds should be adjusted based on your requirements

	length := dims.LengthMeters
	width := dims.WidthMeters

	// Motorcycle: small vehicles
	if length < 2.5 && width < 1.5 {
		return VehicleClass{
			Class:       "motorcycle",
			Confidence:  dims.Confidence,
			Description: "Sepeda Motor / Kendaraan Kecil",
		}
	}

	// Sedan/Car: medium-sized passenger vehicles
	if length >= 2.5 && length < 5.5 && width < 2.0 {
		return VehicleClass{
			Class:       "sedan",
			Confidence:  dims.Confidence,
			Description: "Mobil Penumpang / Sedan",
		}
	}

	// SUV/Minivan: larger passenger vehicles
	if length >= 4.0 && length < 6.0 && width >= 1.8 && width < 2.2 {
		return VehicleClass{
			Class:       "suv",
			Confidence:  dims.Confidence,
			Description: "SUV / Minivan",
		}
	}

	// Truck: medium to large cargo vehicles
	if length >= 5.5 && length < 12.0 {
		return VehicleClass{
			Class:       "truck",
			Confidence:  dims.Confidence,
			Description: "Truk / Kendaraan Barang",
		}
	}

	// Bus: long passenger vehicles
	if length >= 7.0 && width >= 2.0 {
		return VehicleClass{
			Class:       "bus",
			Confidence:  dims.Confidence,
			Description: "Bus / Kendaraan Penumpang Besar",
		}
	}

	// Default: unknown
	return VehicleClass{
		Class:       "unknown",
		Confidence:  dims.Confidence * 0.5, // Lower confidence for unknown
		Description: "Kendaraan Tidak Teridentifikasi",
	}
}

// String returns a formatted string representation of vehicle dimensions
func (vd VehicleDimensions) String() string {
	return fmt.Sprintf(
		"Vehicle Dimensions:\n"+
			"  Length: %.2f m\n"+
			"  Width: %.2f m\n"+
			"  Height: %.2f m (estimated)\n"+
			"  Distance: %.2f m\n"+
			"  Confidence: %.2f%%\n"+
			"  Timestamp: %s",
		vd.LengthMeters,
		vd.WidthMeters,
		vd.HeightMeters,
		vd.DistanceMeters,
		vd.Confidence*100,
		vd.Timestamp.Format("2006-01-02 15:04:05"),
	)
}

// IsValid checks if the dimensions are within reasonable ranges
func (vd VehicleDimensions) IsValid() bool {
	// Check if dimensions are within reasonable ranges for vehicles
	if vd.LengthMeters < 1.0 || vd.LengthMeters > 20.0 {
		return false
	}
	if vd.WidthMeters < 0.5 || vd.WidthMeters > 3.5 {
		return false
	}
	if vd.HeightMeters < 0.5 || vd.HeightMeters > 5.0 {
		return false
	}
	if vd.Confidence < 0.3 { // Minimum confidence threshold
		return false
	}
	return true
}
