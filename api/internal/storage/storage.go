// Package storage abstracts object storage (product images, payment proofs,
// invoices) behind a small interface (MH-201). Keys are namespaced by tenant_id.
// Two backends: MinIO/S3 (prod) and a local-disk backend served via signed,
// time-limited URLs (dev/test, no external deps).
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// PutResult is returned after a successful upload.
type PutResult struct {
	Key  string
	Size int64
}

// Storage is the object-storage contract used by the domain.
type Storage interface {
	// Put stores an object under a tenant-namespaced key and returns the key.
	Put(ctx context.Context, tenantID uuid.UUID, prefix, filename, contentType string, r io.Reader, size int64) (PutResult, error)
	// SignedGetURL returns a time-limited download URL for a key.
	SignedGetURL(ctx context.Context, key string, ttlSeconds int) (string, error)
	// Delete removes an object (best-effort, e.g. on image replace).
	Delete(ctx context.Context, key string) error
	// Open returns a reader for a key (used by the local HTTP serve route).
	Open(ctx context.Context, key string) (io.ReadCloser, string, error)
}

// Limits enforced at the boundary (NFR security): allowed content types & max size.
var (
	MaxUploadBytes   int64 = 8 << 20 // 8 MiB
	AllowedImageMIME       = map[string]bool{
		"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true,
	}
	AllowedProofMIME = map[string]bool{
		"image/jpeg": true, "image/png": true, "image/webp": true, "application/pdf": true,
	}
)

func keyFor(tenantID uuid.UUID, prefix, filename string) string {
	return fmt.Sprintf("%s/%s/%s-%s", tenantID.String(), prefix, uuid.NewString(), sanitize(filename))
}

func sanitize(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "file"
	}
	return string(out)
}
