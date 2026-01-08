package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"movie-backend/internal/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

type MinIOService struct {
	client    *minio.Client
	bucket    string
	publicURL string
	logger    *logrus.Logger
}

func NewMinIOService(cfg *config.MinIOConfig, logger *logrus.Logger) (*MinIOService, error) {
	endpoint := cfg.Endpoint
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
		"bucket":   cfg.BucketName,
		"useSSL":   cfg.UseSSL,
	}).Info("MinIO client initialized successfully")

	service := &MinIOService{
		client:    minioClient,
		bucket:    cfg.BucketName,
		publicURL: cfg.PublicURL,
		logger:    logger,
	}

	if err := service.ensureBucket(context.Background()); err != nil {
		logger.WithError(err).Warn("Failed to configure bucket, but continuing...")
	}

	return service, nil
}

func (s *MinIOService) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		s.logger.WithField("bucket", s.bucket).Info("Bucket created successfully")
	}

	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, s.bucket)

	if err := s.client.SetBucketPolicy(ctx, s.bucket, policy); err != nil {
		return fmt.Errorf("failed to set bucket policy: %w", err)
	}

	s.logger.WithField("bucket", s.bucket).Info("Bucket policy set to public read")
	return nil
}

func (s *MinIOService) GeneratePresignedURL(filename, contentType string) (string, string, error) {
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	uniqueFilename := fmt.Sprintf("%s_%s%s", nameWithoutExt, uuid.New().String()[:8], ext)

	objectPath := uniqueFilename

	// Set expiration time (15 minutes)
	expiry := time.Duration(15) * time.Minute

	presignedURL, err := s.client.PresignedPutObject(
		context.Background(),
		s.bucket,
		objectPath,
		expiry,
	)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate presigned URL")
		return "", "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	publicBase := strings.TrimPrefix(s.publicURL, "https://")
	publicBase = strings.TrimPrefix(publicBase, "http://")

	if idx := strings.Index(publicBase, "/"); idx != -1 {
		publicBase = publicBase[:idx]
	}

	protocol := "http://"
	if strings.Contains(s.publicURL, "https://") {
		protocol = "https://"
	}

	publicURL := fmt.Sprintf("%s%s/%s/%s", protocol, publicBase, s.bucket, objectPath)

	s.logger.WithFields(logrus.Fields{
		"filename":   filename,
		"objectPath": objectPath,
		"expiry":     expiry,
	}).Info("Generated presigned URL")

	return presignedURL.String(), publicURL, nil
}

func (s *MinIOService) DeleteFile(objectPath string) error {
	if strings.Contains(objectPath, "http") {
		parts := strings.Split(objectPath, "/")
		if len(parts) > 0 {
			objectPath = parts[len(parts)-1]
		}
	}

	objectPath = strings.TrimPrefix(objectPath, s.bucket+"/")

	err := s.client.RemoveObject(
		context.Background(),
		s.bucket,
		objectPath,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		s.logger.WithError(err).WithField("objectPath", objectPath).Error("Failed to delete file")
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.WithField("objectPath", objectPath).Info("File deleted successfully from MinIO")
	return nil
}
