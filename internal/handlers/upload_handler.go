package handlers

import (
	"movie-backend/internal/services"
	"movie-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type UploadHandler struct {
	minioService *services.MinIOService
	logger       *logrus.Logger
}

func NewUploadHandler(minioService *services.MinIOService, logger *logrus.Logger) *UploadHandler {
	return &UploadHandler{
		minioService: minioService,
		logger:       logger,
	}
}

// GetPresignedURL godoc
// @Summary Get presigned URL for file upload
// @Description Generate a presigned URL for uploading files to MinIO/S3
// @Tags Upload
// @Accept json
// @Produce json
// @Param filename query string true "Filename"
// @Param contentType query string false "Content Type" default(image/jpeg)
// @Success 200 {object} utils.StandardResponse
// @Failure 400 {object} utils.StandardResponse
// @Failure 500 {object} utils.StandardResponse
// @Router /upload/presign [get]
func (h *UploadHandler) GetPresignedURL(c *fiber.Ctx) error {
	filename := c.Query("filename")
	if filename == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "filename is required")
	}

	contentType := c.Query("contentType", "image/jpeg")

	presignedURL, publicURL, err := h.minioService.GeneratePresignedURL(filename, contentType)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate presigned URL")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate presigned URL")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Presigned URL generated successfully", fiber.Map{
		"presigned_url": presignedURL,
		"public_url":    publicURL,
	})
}
