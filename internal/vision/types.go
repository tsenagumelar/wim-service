package vision

import "time"

// BoundingBox represents a detected vehicle's bounding box
type BoundingBox struct {
	X      int     // Top-left X coordinate
	Y      int     // Top-left Y coordinate
	Width  int     // Width of bounding box
	Height int     // Height of bounding box
	Label  string  // Vehicle type label
	Score  float64 // Confidence score
}

// VehicleDimensions represents the calculated dimensions of a vehicle
type VehicleDimensions struct {
	LengthMeters   float64   // Vehicle length in meters
	WidthMeters    float64   // Vehicle width in meters
	HeightMeters   float64   // Vehicle height in meters (estimated)
	DistanceMeters float64   // Distance from camera in meters
	CenterX        int       // Center X coordinate in image
	CenterY        int       // Center Y coordinate in image
	Confidence     float64   // Confidence score (0-1)
	Timestamp      time.Time // When the measurement was taken
	ImagePath      string    // Path to the source image
}

// VehicleClass represents vehicle classification based on dimensions
type VehicleClass struct {
	Class       string  // e.g., "sedan", "truck", "bus", "motorcycle"
	Confidence  float64 // Classification confidence
	Description string  // Human-readable description
}
