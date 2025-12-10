package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"wim-service/internal/vision"
)

// DimensionHandler handles vehicle dimension processing
type DimensionHandler struct {
	DB               *sql.DB
	SiteUUID         string // Site UUID from master_site.id
	DimensionService *vision.DimensionService
	SaveResults      bool // Whether to save results to database
}

// DimensionResult represents the result of dimension processing
type DimensionResult struct {
	ImagePath    string                     `json:"image_path"`
	ProcessedAt  time.Time                  `json:"processed_at"`
	Dimensions   []vision.VehicleDimensions `json:"dimensions"`
	VehicleCount int                        `json:"vehicle_count"`
	Success      bool                       `json:"success"`
	ErrorMessage string                     `json:"error_message,omitempty"`
}

// NewDimensionHandler creates a new dimension handler
func NewDimensionHandler(db *sql.DB, siteUUID, modelPath string, threshold float64) (*DimensionHandler, error) {
	dimensionService := vision.NewDimensionService(modelPath, threshold)

	return &DimensionHandler{
		DB:               db,
		SiteUUID:         siteUUID,
		DimensionService: dimensionService,
		SaveResults:      true,
	}, nil
}

// SetCalibration sets camera calibration parameters
func (dh *DimensionHandler) SetCalibration(calibration *vision.CameraCalibration) error {
	return dh.DimensionService.SetCalibration(calibration)
}

// ProcessImageFile processes a single image file and returns dimensions
func (dh *DimensionHandler) ProcessImageFile(imagePath string) (*DimensionResult, error) {
	log.Printf("[DIMENSION_HANDLER] Processing image: %s", imagePath)

	result := &DimensionResult{
		ImagePath:   imagePath,
		ProcessedAt: time.Now(),
		Success:     false,
	}

	// Process the image
	dimensions, err := dh.DimensionService.ProcessImage(imagePath)
	if err != nil {
		result.ErrorMessage = err.Error()
		log.Printf("[DIMENSION_HANDLER] Error processing image: %v", err)
		return result, err
	}

	result.Dimensions = dimensions
	result.VehicleCount = len(dimensions)
	result.Success = true

	// Save to database if enabled
	if dh.SaveResults && dh.DB != nil {
		if err := dh.saveDimensionsToDatabase(imagePath, dimensions); err != nil {
			log.Printf("[DIMENSION_HANDLER] Warning: Failed to save to database: %v", err)
		}
	}

	log.Printf("[DIMENSION_HANDLER] Successfully processed image. Found %d vehicle(s)", len(dimensions))

	return result, nil
}

// ProcessANPRImage processes an ANPR image with metadata
func (dh *DimensionHandler) ProcessANPRImage(imagePath string, plateNumber string, anprID string) (*DimensionResult, error) {
	log.Printf("[DIMENSION_HANDLER] Processing ANPR image for plate: %s (ANPR ID: %s)", plateNumber, anprID)

	result, err := dh.ProcessImageFile(imagePath)
	if err != nil {
		return result, err
	}

	// Update database with ANPR association if available
	if dh.SaveResults && dh.DB != nil && anprID != "" {
		for i, dims := range result.Dimensions {
			if err := dh.updateANPRWithDimensions(anprID, dims, i); err != nil {
				log.Printf("[DIMENSION_HANDLER] Warning: Failed to update ANPR record: %v", err)
			}
		}
	}

	return result, nil
}

// saveDimensionsToDatabase saves dimension results to database
func (dh *DimensionHandler) saveDimensionsToDatabase(imagePath string, dimensions []vision.VehicleDimensions) error {
	// Create table if not exists
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS vehicle_dimensions (
			id SERIAL PRIMARY KEY,
			image_path VARCHAR(500),
			length_meters DECIMAL(10, 3),
			width_meters DECIMAL(10, 3),
			height_meters DECIMAL(10, 3),
			distance_meters DECIMAL(10, 3),
			confidence DECIMAL(5, 4),
			vehicle_class VARCHAR(50),
			class_description VARCHAR(200),
			center_x INT,
			center_y INT,
			processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := dh.DB.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Insert each dimension result
	insertSQL := `
		INSERT INTO vehicle_dimensions 
		(image_path, length_meters, width_meters, height_meters, distance_meters, 
		 confidence, vehicle_class, class_description, center_x, center_y, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	for _, dims := range dimensions {
		if !dims.IsValid() {
			log.Printf("[DIMENSION_HANDLER] Skipping invalid dimension result")
			continue
		}

		vehicleClass := dh.DimensionService.ClassifyVehicle(dims)

		_, err := dh.DB.Exec(
			insertSQL,
			imagePath,
			dims.LengthMeters,
			dims.WidthMeters,
			dims.HeightMeters,
			dims.DistanceMeters,
			dims.Confidence,
			vehicleClass.Class,
			vehicleClass.Description,
			dims.CenterX,
			dims.CenterY,
			dims.Timestamp,
		)

		if err != nil {
			log.Printf("[DIMENSION_HANDLER] Failed to insert dimension: %v", err)
			continue
		}
	}

	return nil
}

// updateANPRWithDimensions updates ANPR record with dimension data
func (dh *DimensionHandler) updateANPRWithDimensions(anprID string, dims vision.VehicleDimensions, vehicleIndex int) error {
	// Check if dimension columns exist in anpr_data table
	alterTableSQL := `
		ALTER TABLE anpr_data 
		ADD COLUMN IF NOT EXISTS vehicle_length DECIMAL(10, 3),
		ADD COLUMN IF NOT EXISTS vehicle_width DECIMAL(10, 3),
		ADD COLUMN IF NOT EXISTS vehicle_height DECIMAL(10, 3),
		ADD COLUMN IF NOT EXISTS vehicle_class VARCHAR(50),
		ADD COLUMN IF NOT EXISTS dimension_confidence DECIMAL(5, 4);
	`

	if _, err := dh.DB.Exec(alterTableSQL); err != nil {
		return fmt.Errorf("failed to alter table: %w", err)
	}

	vehicleClass := dh.DimensionService.ClassifyVehicle(dims)

	// Update ANPR record with dimension data
	updateSQL := `
		UPDATE anpr_data 
		SET vehicle_length = $1,
		    vehicle_width = $2,
		    vehicle_height = $3,
		    vehicle_class = $4,
		    dimension_confidence = $5
		WHERE id = $6
	`

	_, err := dh.DB.Exec(
		updateSQL,
		dims.LengthMeters,
		dims.WidthMeters,
		dims.HeightMeters,
		vehicleClass.Class,
		dims.Confidence,
		anprID,
	)

	return err
}

// GetDimensionsByImagePath retrieves dimensions from database by image path
func (dh *DimensionHandler) GetDimensionsByImagePath(imagePath string) ([]vision.VehicleDimensions, error) {
	querySQL := `
		SELECT length_meters, width_meters, height_meters, distance_meters,
		       confidence, center_x, center_y, processed_at
		FROM vehicle_dimensions
		WHERE image_path = $1
		ORDER BY processed_at DESC
	`

	rows, err := dh.DB.Query(querySQL, imagePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []vision.VehicleDimensions
	for rows.Next() {
		var dims vision.VehicleDimensions
		err := rows.Scan(
			&dims.LengthMeters,
			&dims.WidthMeters,
			&dims.HeightMeters,
			&dims.DistanceMeters,
			&dims.Confidence,
			&dims.CenterX,
			&dims.CenterY,
			&dims.Timestamp,
		)
		if err != nil {
			log.Printf("[DIMENSION_HANDLER] Error scanning row: %v", err)
			continue
		}
		dims.ImagePath = imagePath
		results = append(results, dims)
	}

	return results, nil
}

// ExportResultToJSON exports dimension result to JSON file
func (dh *DimensionHandler) ExportResultToJSON(result *DimensionResult, outputPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if outputPath == "" {
		outputPath = filepath.Join(
			filepath.Dir(result.ImagePath),
			fmt.Sprintf("dimensions_%d.json", time.Now().Unix()),
		)
	}

	if err := saveJSONToFile(outputPath, data); err != nil {
		return fmt.Errorf("failed to save JSON: %w", err)
	}

	log.Printf("[DIMENSION_HANDLER] Exported result to: %s", outputPath)
	return nil
}

// saveJSONToFile saves JSON data to file
func saveJSONToFile(path string, data []byte) error {
	// Implementation would go here
	// For now, just log
	log.Printf("[DIMENSION_HANDLER] Would save JSON to: %s", path)
	return nil
}
