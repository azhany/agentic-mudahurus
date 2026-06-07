package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// LocalStorage stores objects on the local filesystem and serves them through
// the API via HMAC-signed, time-limited URLs. Default backend in dev/test.
type LocalStorage struct {
	root      string
	baseURL   string // e.g. http://localhost:8080/files
	signerKey []byte
}

func NewLocal(root, baseURL, signerKey string) (*LocalStorage, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalStorage{root: root, baseURL: baseURL, signerKey: []byte(signerKey)}, nil
}

func (s *LocalStorage) Put(ctx context.Context, tenantID uuid.UUID, prefix, filename, contentType string, r io.Reader, size int64) (PutResult, error) {
	key := keyFor(tenantID, prefix, filename)
	full := filepath.Join(s.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return PutResult{}, err
	}
	f, err := os.Create(full)
	if err != nil {
		return PutResult{}, err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return PutResult{}, err
	}
	if n > MaxUploadBytes {
		_ = os.Remove(full)
		return PutResult{}, fmt.Errorf("object exceeds max size %d bytes", MaxUploadBytes)
	}
	// persist content type alongside
	_ = os.WriteFile(full+".ct", []byte(contentType), 0o644)
	return PutResult{Key: key, Size: n}, nil
}

func (s *LocalStorage) SignedGetURL(ctx context.Context, key string, ttlSeconds int) (string, error) {
	exp := time.Now().Add(time.Duration(ttlSeconds) * time.Second).Unix()
	sig := s.sign(key, exp)
	q := url.Values{}
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", sig)
	// key contains '/' (tenant/prefix/file); served via a wildcard route, so it
	// is kept raw in the path (not percent-escaped).
	return fmt.Sprintf("%s/%s?%s", s.baseURL, key, q.Encode()), nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	full := filepath.Join(s.root, filepath.FromSlash(key))
	_ = os.Remove(full + ".ct")
	err := os.Remove(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, string, error) {
	full := filepath.Join(s.root, filepath.FromSlash(key))
	f, err := os.Open(full)
	if err != nil {
		return nil, "", err
	}
	ct := "application/octet-stream"
	if b, err := os.ReadFile(full + ".ct"); err == nil && len(b) > 0 {
		ct = string(b)
	} else if t := mime.TypeByExtension(filepath.Ext(key)); t != "" {
		ct = t
	}
	return f, ct, nil
}

func (s *LocalStorage) sign(key string, exp int64) string {
	mac := hmac.New(sha256.New, s.signerKey)
	fmt.Fprintf(mac, "%s:%d", key, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks an HMAC signature + expiry for the local serve route.
func (s *LocalStorage) Verify(key, sig string, exp int64) bool {
	if time.Now().Unix() > exp {
		return false
	}
	want := s.sign(key, exp)
	return hmac.Equal([]byte(want), []byte(sig))
}
