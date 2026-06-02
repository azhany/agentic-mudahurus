package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWT access tokens carry tenant_id + role (MH-102). Refresh tokens are opaque
// random strings; only their SHA-256 hash is stored server-side (rotation +
// logout invalidation).

type Claims struct {
	TenantID string `json:"tid"`
	Role     string `json:"role"`
	Username string `json:"usr"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewTokenManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

var ErrInvalidToken = errors.New("invalid or expired token")

func (m *TokenManager) IssueAccess(tenantID uuid.UUID, role, username string) (string, time.Time, error) {
	exp := time.Now().Add(m.accessTTL)
	claims := Claims{
		TenantID: tenantID.String(),
		Role:     role,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   tenantID.String(),
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "mudahurus",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(m.accessSecret)
	return s, exp, err
}

func (m *TokenManager) ParseAccess(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.accessSecret, nil
	})
	if err != nil || !tok.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// IssueRefresh returns the opaque token (given to the client) and its hash
// (stored server-side) plus the expiry.
func (m *TokenManager) IssueRefresh() (token, hash string, exp time.Time, err error) {
	token = uuid.NewString() + "." + uuid.NewString()
	hash = HashToken(token)
	exp = time.Now().Add(m.refreshTTL)
	return token, hash, exp, nil
}

// HashToken returns the hex SHA-256 of a token (refresh/reset/verify tokens).
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
