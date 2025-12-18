package vision

import (
	"fmt"
	"math"
)

// CameraCalibration holds camera calibration parameters
type CameraCalibration struct {
	// Intrinsic parameters
	FocalLengthPixels float64 // Focal length in pixels
	ImageWidth        int     // Image width in pixels
	ImageHeight       int     // Image height in pixels
	PrincipalPointX   float64 // Principal point X (usually image_width/2)
	PrincipalPointY   float64 // Principal point Y (usually image_height/2)

	// Extrinsic parameters
	CameraHeightMeters float64 // Camera height from ground in meters
	TiltAngleDegrees   float64 // Camera tilt angle in degrees (0 = horizontal, 90 = looking down)

	// Reference calibration
	ReferencePixelLength int     // Length in pixels of reference object
	ReferenceRealLength  float64 // Real length in meters of reference object
	ReferenceDistanceM   float64 // Distance from camera to reference object in meters

	// Computed values
	PixelToMeterRatio float64 // Conversion ratio at reference distance
}

// NewCameraCalibration creates a new camera calibration with default values
func NewCameraCalibration() *CameraCalibration {
	return &CameraCalibration{
		// Default values - should be configured
		FocalLengthPixels:    1000.0,
		ImageWidth:           1920,
		ImageHeight:          1080,
		PrincipalPointX:      960.0,
		PrincipalPointY:      540.0,
		CameraHeightMeters:   6.0,
		TiltAngleDegrees:     30.0,
		ReferencePixelLength: 200,
		ReferenceRealLength:  5.0,
		ReferenceDistanceM:   10.0,
	}
}

// LoadFromConfig loads calibration parameters from configuration
func (cc *CameraCalibration) LoadFromConfig(
	focalLength float64,
	imageWidth, imageHeight int,
	cameraHeight, tiltAngle float64,
	refPixelLen int,
	refRealLen, refDistance float64,
) {
	cc.FocalLengthPixels = focalLength
	cc.ImageWidth = imageWidth
	cc.ImageHeight = imageHeight
	cc.PrincipalPointX = float64(imageWidth) / 2.0
	cc.PrincipalPointY = float64(imageHeight) / 2.0
	cc.CameraHeightMeters = cameraHeight
	cc.TiltAngleDegrees = tiltAngle
	cc.ReferencePixelLength = refPixelLen
	cc.ReferenceRealLength = refRealLen
	cc.ReferenceDistanceM = refDistance

	cc.ComputePixelToMeterRatio()
}

// ComputePixelToMeterRatio calculates the pixel to meter conversion ratio
func (cc *CameraCalibration) ComputePixelToMeterRatio() {
	if cc.ReferencePixelLength > 0 {
		cc.PixelToMeterRatio = cc.ReferenceRealLength / float64(cc.ReferencePixelLength)
	}
}

// PixelsToMeters converts pixel measurements to meters at a given distance
func (cc *CameraCalibration) PixelsToMeters(pixels int, distanceMeters float64) float64 {
	// Simple linear approximation using reference calibration
	// For more accuracy, use perspective transformation
	if cc.ReferenceDistanceM > 0 {
		// Adjust ratio based on distance (objects farther appear smaller)
		distanceRatio := distanceMeters / cc.ReferenceDistanceM
		adjustedRatio := cc.PixelToMeterRatio * distanceRatio
		return float64(pixels) * adjustedRatio
	}

	// Fallback to direct ratio
	return float64(pixels) * cc.PixelToMeterRatio
}

// EstimateDistance estimates distance from camera to object based on its position in image
func (cc *CameraCalibration) EstimateDistance(pixelY int) float64 {
	// Calculate distance using camera height and tilt angle
	// This is a simplified model assuming flat ground

	tiltRad := cc.TiltAngleDegrees * math.Pi / 180.0

	// Calculate the angle from camera to pixel
	pixelOffsetY := float64(pixelY) - cc.PrincipalPointY
	angleToPixel := math.Atan(pixelOffsetY / cc.FocalLengthPixels)

	// Calculate ground distance
	totalAngle := tiltRad - angleToPixel
	if math.Abs(math.Tan(totalAngle)) < 0.001 {
		return cc.ReferenceDistanceM // Avoid division by near-zero
	}

	distance := cc.CameraHeightMeters / math.Tan(totalAngle)

	// Clamp to reasonable values
	if distance < 1.0 {
		distance = 1.0
	}
	if distance > 100.0 {
		distance = 100.0
	}

	return distance
}

// CalculateGroundDimensions calculates real-world dimensions from bounding box
func (cc *CameraCalibration) CalculateGroundDimensions(bbox BoundingBox) (*VehicleDimensions, error) {
	// Calculate bottom center of bounding box (closest point to camera)
	bottomY := bbox.Y + bbox.Height
	centerX := bbox.X + bbox.Width/2

	// Estimate distance to vehicle
	distance := cc.EstimateDistance(bottomY)

	// Convert pixel dimensions to meters
	// PENTING: Untuk camera ANPR yang menghadap kendaraan dari depan/belakang:
	// - bbox.Width (horizontal) = panjang kendaraan (tampak dari depan)
	// - bbox.Height (vertical) = tinggi kendaraan (tampak dari depan)
	//
	// Jadi kita swap length dan width untuk mendapatkan dimensi yang benar
	length := cc.PixelsToMeters(bbox.Width, distance) // Width pixel → Length (panjang kendaraan)
	width := cc.PixelsToMeters(bbox.Height, distance) // Height pixel → Width (lebar kendaraan)

	// CORRECTION: Apply vertical scale factor for width
	// Vertical measurements need different scaling due to camera perspective
	// Empirically determined: vertical measurements are ~1.47x oversized (2.64m / 1.8m)
	// Correction factor: 0.68 (to scale 2.64m down to 1.8m)
	verticalScaleFactor := 0.68
	width = width * verticalScaleFactor

	// Height estimation (simplified - assumes vehicle height proportional to length)
	// For better accuracy, need side-view camera or 3D reconstruction
	height := length * 0.4 // Rough estimate: vehicles are typically 40% as tall as long

	return &VehicleDimensions{
		LengthMeters:   length,
		WidthMeters:    width,
		HeightMeters:   height,
		DistanceMeters: distance,
		CenterX:        centerX,
		CenterY:        bottomY,
		Confidence:     0.7, // Medium confidence without side view
	}, nil
}

// Validate checks if calibration parameters are reasonable
func (cc *CameraCalibration) Validate() error {
	if cc.FocalLengthPixels <= 0 {
		return fmt.Errorf("focal length must be positive")
	}
	if cc.ImageWidth <= 0 || cc.ImageHeight <= 0 {
		return fmt.Errorf("image dimensions must be positive")
	}
	if cc.CameraHeightMeters <= 0 {
		return fmt.Errorf("camera height must be positive")
	}
	if cc.TiltAngleDegrees < 0 || cc.TiltAngleDegrees > 90 {
		return fmt.Errorf("tilt angle must be between 0 and 90 degrees")
	}
	if cc.ReferenceRealLength <= 0 {
		return fmt.Errorf("reference length must be positive")
	}

	return nil
}

// GetCalibrationInfo returns a string with calibration information
func (cc *CameraCalibration) GetCalibrationInfo() string {
	return fmt.Sprintf(
		"Camera Calibration:\n"+
			"  Resolution: %dx%d\n"+
			"  Focal Length: %.2f pixels\n"+
			"  Height: %.2f m\n"+
			"  Tilt Angle: %.2f°\n"+
			"  Reference: %d pixels = %.2f m at %.2f m distance\n"+
			"  Pixel-to-Meter Ratio: %.6f m/pixel",
		cc.ImageWidth, cc.ImageHeight,
		cc.FocalLengthPixels,
		cc.CameraHeightMeters,
		cc.TiltAngleDegrees,
		cc.ReferencePixelLength, cc.ReferenceRealLength, cc.ReferenceDistanceM,
		cc.PixelToMeterRatio,
	)
}
