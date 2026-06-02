package storage

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage is the MinIO/S3 backend (prod). Signed URLs are presigned GETs.
type S3Storage struct {
	client *minio.Client
	bucket string
}

func NewS3(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Storage, error) {
	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := cl.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := cl.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	return &S3Storage{client: cl, bucket: bucket}, nil
}

func (s *S3Storage) Put(ctx context.Context, tenantID uuid.UUID, prefix, filename, contentType string, r io.Reader, size int64) (PutResult, error) {
	key := keyFor(tenantID, prefix, filename)
	info, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return PutResult{}, err
	}
	return PutResult{Key: key, Size: info.Size}, nil
}

func (s *S3Storage) SignedGetURL(ctx context.Context, key string, ttlSeconds int) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, time.Duration(ttlSeconds)*time.Second, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *S3Storage) Open(ctx context.Context, key string) (io.ReadCloser, string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	stat, err := obj.Stat()
	ct := "application/octet-stream"
	if err == nil && stat.ContentType != "" {
		ct = stat.ContentType
	}
	return obj, ct, nil
}
