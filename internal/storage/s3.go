package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/voice-chat-team/services.files/internal/config"
)

type Storage struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func New(cfg *config.Config) (*Storage, error) {
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	return &Storage{client: client, bucket: cfg.S3Bucket, publicURL: cfg.S3PublicURL}, nil
}

func (s *Storage) PresignUpload(ctx context.Context, objectName string, ttl time.Duration) (string, error) {
	url, err := s.client.PresignedPutObject(ctx, s.bucket, objectName, ttl)
	if err != nil {
		return "", fmt.Errorf("presign put: %w", err)
	}

	return url.String(), nil
}

func (s *Storage) PresignDownload(ctx context.Context, objectName string, ttl time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, ttl, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign get: %w", err)
	}

	return url.String(), nil
}

func (s *Storage) Stat(ctx context.Context, objectName string) (minio.ObjectInfo, error) {
	return s.client.StatObject(ctx, s.bucket, objectName, minio.StatObjectOptions{})
}

func (s *Storage) Remove(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *Storage) PublicURL(key string) string {
	if s.publicURL == "" {
		return ""
	}
	return strings.TrimRight(s.publicURL, "/") + "/" + key
}
