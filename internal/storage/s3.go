// Package storage provides a direct Minio/S3 client for file operations,
// replacing PocketBase's filesystem API.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds S3/Minio connection configuration.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// Service wraps a Minio client for file storage operations.
type Service struct {
	client *minio.Client
	bucket string
}

// New creates a new storage service connected to Minio/S3.
func New(cfg Config) (*Service, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating minio client: %w", err)
	}

	return &Service{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// objectKey builds the S3 key for a PocketBase-compatible file path.
// Format: {collection}/{recordId}/{filename}
// This maintains backward compatibility with existing file URLs.
func objectKey(collection, recordID, filename string) string {
	return collection + "/" + recordID + "/" + filename
}

// Upload stores a file in S3.
func (s *Service) Upload(ctx context.Context, collection, recordID, filename string, reader io.Reader, size int64, contentType string) error {
	key := objectKey(collection, recordID, filename)

	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, opts)
	if err != nil {
		return fmt.Errorf("uploading %s: %w", key, err)
	}

	return nil
}

// Download retrieves a file from S3.
func (s *Service) Download(ctx context.Context, collection, recordID, filename string) (io.ReadCloser, error) {
	key := objectKey(collection, recordID, filename)

	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", key, err)
	}

	return obj, nil
}

// Delete removes a file from S3.
func (s *Service) Delete(ctx context.Context, collection, recordID, filename string) error {
	key := objectKey(collection, recordID, filename)

	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("deleting %s: %w", key, err)
	}

	return nil
}

// PresignedURL generates a presigned URL for temporary direct access.
func (s *Service) PresignedURL(ctx context.Context, collection, recordID, filename string, expires time.Duration) (string, error) {
	key := objectKey(collection, recordID, filename)

	reqParams := make(url.Values)
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL for %s: %w", key, err)
	}

	return u.String(), nil
}

// Exists checks if a file exists in S3.
func (s *Service) Exists(ctx context.Context, collection, recordID, filename string) (bool, error) {
	key := objectKey(collection, recordID, filename)

	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		// Check if it's a "not found" error
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("checking %s: %w", key, err)
	}

	return true, nil
}

// UploadBytes is a convenience function to upload a byte slice.
func (s *Service) UploadBytes(ctx context.Context, collection, recordID, filename string, data []byte, contentType string) error {
	reader := io.NopCloser(io.NewSectionReader(
		&byteReaderAt{data: data}, 0, int64(len(data)),
	))
	return s.Upload(ctx, collection, recordID, filename, reader, int64(len(data)), contentType)
}

type byteReaderAt struct {
	data []byte
}

func (b *byteReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
