package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mudahurus/api/internal/db"
)

var ErrNotFound = errors.New("not found")

type Tenant struct {
	ID            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	Role          string    `json:"role"`
	FullName      string    `json:"full_name"`
	StoreName     string    `json:"store_name"`
	Phone         string    `json:"phone"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

type Store struct{ pool *db.Pool }

func NewStore(pool *db.Pool) *Store { return &Store{pool: pool} }

func (s *Store) CreateTenant(ctx context.Context, t *Tenant) error {
	return s.pool.QueryRow(ctx, `
		INSERT INTO tenants (username, email, password_hash, role, full_name, store_name, phone)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at`,
		t.Username, t.Email, t.PasswordHash, t.Role, t.FullName, t.StoreName, t.Phone,
	).Scan(&t.ID, &t.CreatedAt)
}

func (s *Store) tenantByQuery(ctx context.Context, where string, arg any) (*Tenant, error) {
	t := &Tenant{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, role, full_name, store_name, phone, email_verified, created_at
		FROM tenants WHERE `+where, arg,
	).Scan(&t.ID, &t.Username, &t.Email, &t.PasswordHash, &t.Role, &t.FullName, &t.StoreName, &t.Phone, &t.EmailVerified, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) TenantByUsername(ctx context.Context, username string) (*Tenant, error) {
	return s.tenantByQuery(ctx, "lower(username) = lower($1)", username)
}
func (s *Store) TenantByEmail(ctx context.Context, email string) (*Tenant, error) {
	return s.tenantByQuery(ctx, "lower(email) = lower($1)", email)
}
func (s *Store) TenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	return s.tenantByQuery(ctx, "id = $1", id)
}

func (s *Store) UpdateProfile(ctx context.Context, id uuid.UUID, fullName, storeName, phone string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenants SET full_name=$2, store_name=$3, phone=$4, updated_at=now() WHERE id=$1`,
		id, fullName, storeName, phone)
	return err
}

func (s *Store) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE tenants SET password_hash=$2, updated_at=now() WHERE id=$1`, id, hash)
	return err
}

func (s *Store) MarkEmailVerified(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE tenants SET email_verified=true, updated_at=now() WHERE id=$1`, id)
	return err
}

// ---- refresh tokens ----

func (s *Store) SaveRefresh(ctx context.Context, tenantID uuid.UUID, hash string, exp time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (tenant_id, token_hash, expires_at) VALUES ($1,$2,$3)`,
		tenantID, hash, exp)
	return err
}

// RotateRefresh validates a refresh token hash (unexpired, unrevoked), revokes
// it, and returns the owning tenant_id. Returns ErrNotFound if invalid.
func (s *Store) RotateRefresh(ctx context.Context, hash string) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
		RETURNING tenant_id`, hash).Scan(&tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return tenantID, err
}

func (s *Store) RevokeRefresh(ctx context.Context, hash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`, hash)
	return err
}

// ---- single-use auth tokens (verify email / reset password) ----

func (s *Store) SaveAuthToken(ctx context.Context, tenantID uuid.UUID, purpose, hash string, exp time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO auth_tokens (tenant_id, purpose, token_hash, expires_at) VALUES ($1,$2,$3,$4)`,
		tenantID, purpose, hash, exp)
	return err
}

// ConsumeAuthToken atomically marks a valid token used and returns its tenant_id.
func (s *Store) ConsumeAuthToken(ctx context.Context, purpose, hash string) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		UPDATE auth_tokens SET used_at = now()
		WHERE token_hash = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
		RETURNING tenant_id`, hash, purpose).Scan(&tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return tenantID, err
}
