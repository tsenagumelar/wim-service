package handler

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type AttachmentHandler struct {
	MinioClient *minio.Client
	Bucket      string
}

// NewAttachmentHandler creates a new attachment handler
func NewAttachmentHandler(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*AttachmentHandler, error) {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &AttachmentHandler{
		MinioClient: mc,
		Bucket:      bucket,
	}, nil
}

type UploadResponse struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path"`
	Message  string `json:"message,omitempty"`
}

// UploadImage handles image upload to MinIO
func (h *AttachmentHandler) UploadImage(c *fiber.Ctx) error {
	log.Println("[ATTACHMENT] Upload request received")

	// Get uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		log.Printf("[ATTACHMENT] No file in request: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Success: false,
			Message: "No image file provided",
		})
	}

	log.Printf("[ATTACHMENT] File received: %s, Size: %d bytes", file.Filename, file.Size)

	// Validate file type (only images) - case insensitive
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	if !allowedExts[ext] {
		log.Printf("[ATTACHMENT] Invalid file type: %s", ext)
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid file type '%s'. Only image files are allowed (jpg, jpeg, png, gif, webp)", ext),
		})
	}

	// Open uploaded file
	fileReader, err := file.Open()
	if err != nil {
		log.Printf("[ATTACHMENT] Failed to open uploaded file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Success: false,
			Message: "Failed to process uploaded file",
		})
	}
	defer fileReader.Close()

	// Generate unique filename without folder structure
	// Format: uuid-originalname.ext
	uniqueID := uuid.New().String()
	objectName := fmt.Sprintf("%s-%s", uniqueID, file.Filename)

	// Determine content type based on extension
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	// Upload to MinIO
	ctx := context.Background()
	_, err = h.MinioClient.PutObject(ctx, h.Bucket, objectName, fileReader, file.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		log.Printf("[ATTACHMENT] Failed to upload to MinIO: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Success: false,
			Message: "Failed to upload file to storage",
		})
	}

	// Construct file path
	filePath := fmt.Sprintf("%s/%s", h.Bucket, objectName)

	log.Printf("[ATTACHMENT] Successfully uploaded: %s", filePath)

	return c.JSON(UploadResponse{
		Success:  true,
		FilePath: filePath,
		Message:  "File uploaded successfully",
	})
}
