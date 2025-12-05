package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"wim-service/internal/ftpwatcher"
	"wim-service/internal/handler"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("[ENV] .env file not found, using system environment")
	} else {
		log.Println("[ENV] .env file loaded successfully")
	}
}

func main() {
	ftpHost := getEnv("FTP_HOST", "72.61.213.6:21")
	ftpUser := getEnv("FTP_USER", "ftpuser")
	ftpPass := getEnv("FTP_PASS", "ftpsecret123")
	ftpDir := getEnv("FTP_DIR", "/")
	ftpInterval := getEnvInt("FTP_INTERVAL_SEC", 5)

	minioEndpoint := getEnv("MINIO_ENDPOINT", "s3minio.activa.id")
	minioAccess := getEnv("MINIO_ACCESS_KEY", "admin")
	minioSecret := getEnv("MINIO_SECRET_KEY", "admin12345")
	minioBucket := getEnv("MINIO_BUCKET", "axle")
	minioUseSSL := getEnvBool("MINIO_USE_SSL", true)

	log.Println("WIM AXLE watcher starting...")
	log.Println("FTP AXLE:", ftpHost, "dir:", ftpDir)
	log.Println("MinIO endpoint:", minioEndpoint, "bucket:", minioBucket)

	processor, err := handler.NewAxleProcessor(
		ftpDir,
		minioEndpoint,
		minioAccess,
		minioSecret,
		minioBucket,
		minioUseSSL,
	)
	if err != nil {
		log.Fatal("init axle processor:", err)
	}

	w := ftpwatcher.New(
		ftpHost,
		ftpUser,
		ftpPass,
		ftpDir,
		time.Duration(ftpInterval)*time.Second,
		processor.HandleNewFileAXLE,
	)

	ctx := context.Background()
	if err := w.Start(ctx); err != nil {
		log.Fatal(err)
	}
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
