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

// LegacyPrefixes maps collection names to PocketBase's internal collection IDs.
// Existing files in S3 are stored under {pbCollectionID}/{recordID}/{filename}.
// This mapping allows looking up files by collection name.
var LegacyPrefixes = map[string]string{
	"schematics":        "ezzomjw4q1qibza",
	"temp_uploads":      "pbc_temp_uploads",
	"temp_upload_files": "pbc_temp_upload_files",
}

// CollectionPrefix returns the S3 prefix for a given collection name.
// It uses the legacy PocketBase collection ID if one exists, otherwise
// returns the collection name directly (for new collections).
func CollectionPrefix(collection string) string {
	if prefix, ok := LegacyPrefixes[collection]; ok {
		return prefix
	}
	return collection
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

// --- Raw key operations (for non-collection paths like _internal/*, _thumbs/*) ---

// UploadRaw stores data at an arbitrary S3 key (not scoped to a collection).
func (s *Service) UploadRaw(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
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

// UploadRawBytes is a convenience function to upload a byte slice at an arbitrary S3 key.
func (s *Service) UploadRawBytes(ctx context.Context, key string, data []byte, contentType string) error {
	reader := io.NopCloser(io.NewSectionReader(
		&byteReaderAt{data: data}, 0, int64(len(data)),
	))
	return s.UploadRaw(ctx, key, reader, int64(len(data)), contentType)
}

// DownloadRaw retrieves a file from S3 by arbitrary key.
func (s *Service) DownloadRaw(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", key, err)
	}
	return obj, nil
}

// ExistsRaw checks if an arbitrary key exists in S3.
func (s *Service) ExistsRaw(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("checking %s: %w", key, err)
	}
	return true, nil
}

// DeleteRaw removes an arbitrary key from S3.
func (s *Service) DeleteRaw(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("deleting %s: %w", key, err)
	}
	return nil
}

// Stat returns the object info (size, content-type, etc.) for a file in S3.
func (s *Service) Stat(ctx context.Context, collection, recordID, filename string) (minio.ObjectInfo, error) {
	key := objectKey(collection, recordID, filename)
	return s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
}

// StatRaw returns the object info for an arbitrary S3 key.
func (s *Service) StatRaw(ctx context.Context, key string) (minio.ObjectInfo, error) {
	return s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
}

// Bucket returns the configured bucket name.
func (s *Service) Bucket() string {
	return s.bucket
}
