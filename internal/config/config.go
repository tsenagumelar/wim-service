package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DatabaseURL string
	DB          *sql.DB

	// API Config
	APIPort   string
	JWTSecret string

	// ANPR FTP Config
	ANPRFTPHost     string
	ANPRFTPUser     string
	ANPRFTPPass     string
	ANPRFTPDir      string
	ANPRFTPInterval time.Duration

	// AXLE FTP Config
	AxleFTPHost     string
	AxleFTPUser     string
	AxleFTPPass     string
	AxleFTPDir      string
	AxleFTPInterval time.Duration

	// MinIO Config for ANPR
	ANPRMinIOEndpoint string
	ANPRMinIOAccess   string
	ANPRMinIOSecret   string
	ANPRMinIOBucket   string
	ANPRMinIOUseSSL   bool

	// MinIO Config for AXLE
	AxleMinIOEndpoint string
	AxleMinIOAccess   string
	AxleMinIOSecret   string
	AxleMinIOBucket   string
	AxleMinIOUseSSL   bool

	// Vehicle Dimension Detection Config
	DimensionEnabled   bool    // Enable dimension detection
	DimensionModelPath string  // Path to detection model (if using ML model)
	DimensionThreshold float64 // Detection confidence threshold

	// Camera Calibration Parameters
	CameraFocalLength    float64 // Focal length in pixels
	CameraImageWidth     int     // Image width in pixels
	CameraImageHeight    int     // Image height in pixels
	CameraHeight         float64 // Camera height from ground in meters
	CameraTiltAngle      float64 // Camera tilt angle in degrees
	CameraRefPixelLength int     // Reference object length in pixels
	CameraRefRealLength  float64 // Reference object length in meters
	CameraRefDistance    float64 // Distance to reference object in meters
}

func Load() (*Config, error) {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("[CONFIG] .env file not found, using system environment")
	} else {
		log.Println("[CONFIG] .env file loaded successfully")
	}

	cfg := &Config{
		DatabaseURL: getEnv("DATABASE_URL", ""),

		// API Config
		APIPort:   getEnv("API_PORT", "3000"),
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),

		// ANPR FTP
		ANPRFTPHost:     getEnv("ANPR_FTP_HOST", "72.61.213.6:21"),
		ANPRFTPUser:     getEnv("ANPR_FTP_USER", "ftpuser"),
		ANPRFTPPass:     getEnv("ANPR_FTP_PASS", "ftpsecret123"),
		ANPRFTPDir:      getEnv("ANPR_FTP_DIR", "/"),
		ANPRFTPInterval: time.Duration(getEnvInt("ANPR_FTP_INTERVAL_SEC", 5)) * time.Second,

		// AXLE FTP
		AxleFTPHost:     getEnv("AXLE_FTP_HOST", "72.61.213.6:21"),
		AxleFTPUser:     getEnv("AXLE_FTP_USER", "ftpuser"),
		AxleFTPPass:     getEnv("AXLE_FTP_PASS", "ftpsecret123"),
		AxleFTPDir:      getEnv("AXLE_FTP_DIR", "/"),
		AxleFTPInterval: time.Duration(getEnvInt("AXLE_FTP_INTERVAL_SEC", 5)) * time.Second,

		// ANPR MinIO
		ANPRMinIOEndpoint: getEnv("ANPR_MINIO_ENDPOINT", "s3minio.activa.id"),
		ANPRMinIOAccess:   getEnv("ANPR_MINIO_ACCESS_KEY", "admin"),
		ANPRMinIOSecret:   getEnv("ANPR_MINIO_SECRET_KEY", "admin12345"),
		ANPRMinIOBucket:   getEnv("ANPR_MINIO_BUCKET", "anpr"),
		ANPRMinIOUseSSL:   getEnvBool("ANPR_MINIO_USE_SSL", true),

		// AXLE MinIO
		AxleMinIOEndpoint: getEnv("AXLE_MINIO_ENDPOINT", "s3minio.activa.id"),
		AxleMinIOAccess:   getEnv("AXLE_MINIO_ACCESS_KEY", "admin"),
		AxleMinIOSecret:   getEnv("AXLE_MINIO_SECRET_KEY", "admin12345"),
		AxleMinIOBucket:   getEnv("AXLE_MINIO_BUCKET", "axle"),
		AxleMinIOUseSSL:   getEnvBool("AXLE_MINIO_USE_SSL", true),

		// Vehicle Dimension Detection
		DimensionEnabled:   getEnvBool("DIMENSION_ENABLED", false),
		DimensionModelPath: getEnv("DIMENSION_MODEL_PATH", ""),
		DimensionThreshold: getEnvFloat("DIMENSION_THRESHOLD", 0.5),

		// Camera Calibration (default values - should be calibrated)
		CameraFocalLength:    getEnvFloat("CAMERA_FOCAL_LENGTH", 1000.0),
		CameraImageWidth:     getEnvInt("CAMERA_IMAGE_WIDTH", 1920),
		CameraImageHeight:    getEnvInt("CAMERA_IMAGE_HEIGHT", 1080),
		CameraHeight:         getEnvFloat("CAMERA_HEIGHT_METERS", 6.0),
		CameraTiltAngle:      getEnvFloat("CAMERA_TILT_ANGLE", 30.0),
		CameraRefPixelLength: getEnvInt("CAMERA_REF_PIXEL_LENGTH", 200),
		CameraRefRealLength:  getEnvFloat("CAMERA_REF_REAL_LENGTH", 5.0),
		CameraRefDistance:    getEnvFloat("CAMERA_REF_DISTANCE", 10.0),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	log.Println("[CONFIG] Connecting to database...")

	// Initialize database connection
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	log.Println("[CONFIG] Testing database connection...")

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	cfg.DB = db
	log.Println("[CONFIG] Database connection established")

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch v {
		case "1", "true", "TRUE":
			return true
		case "0", "false", "FALSE":
			return false
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
	}
	return def
}
