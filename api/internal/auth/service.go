package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/notify"
)

type Service struct {
	store  *Store
	tokens *TokenManager
	mailer notify.Mailer
}

func NewService(store *Store, tokens *TokenManager, mailer notify.Mailer) *Service {
	return &Service{store: store, tokens: tokens, mailer: mailer}
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Tenant       *Tenant   `json:"tenant"`
}

func (s *Service) issuePair(ctx context.Context, t *Tenant) (*TokenPair, error) {
	access, exp, err := s.tokens.IssueAccess(t.ID, t.Role, t.Username)
	if err != nil {
		return nil, err
	}
	refresh, hash, rexp, err := s.tokens.IssueRefresh()
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveRefresh(ctx, t.ID, hash, rexp); err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresAt: exp, Tenant: t}, nil
}

type RegisterInput struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FullName  string `json:"full_name"`
	StoreName string `json:"store_name"`
	Phone     string `json:"phone"`
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*Tenant, error) {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if err := httpx.NewValidator().
		Require("username", in.Username).
		Require("email", in.Email).
		Require("password", in.Password).
		Check(len(in.Password) >= 8, "password", "must be at least 8 characters").
		Check(strings.Contains(in.Email, "@"), "email", "must be a valid email").
		Err(); err != nil {
		return nil, err
	}
	if _, err := s.store.TenantByUsername(ctx, in.Username); err == nil {
		return nil, httpx.Conflict("username already taken")
	}
	if _, err := s.store.TenantByEmail(ctx, in.Email); err == nil {
		return nil, httpx.Conflict("email already registered")
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	t := &Tenant{
		Username: in.Username, Email: in.Email, PasswordHash: hash,
		Role: "seller", FullName: in.FullName, StoreName: in.StoreName, Phone: in.Phone,
	}
	if err := s.store.CreateTenant(ctx, t); err != nil {
		return nil, err
	}
	// Email verification token (single-use, 24h).
	if err := s.sendToken(ctx, t, "verify_email", 24*time.Hour, "Verify your MUDAHURUS account"); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Login(ctx context.Context, usernameOrEmail, password string) (*TokenPair, error) {
	usernameOrEmail = strings.ToLower(strings.TrimSpace(usernameOrEmail))
	var t *Tenant
	var err error
	if strings.Contains(usernameOrEmail, "@") {
		t, err = s.store.TenantByEmail(ctx, usernameOrEmail)
	} else {
		t, err = s.store.TenantByUsername(ctx, usernameOrEmail)
	}
	if err != nil {
		return nil, httpx.Unauthorized("invalid credentials")
	}
	ok, err := VerifyPassword(password, t.PasswordHash)
	if err != nil || !ok {
		return nil, httpx.Unauthorized("invalid credentials")
	}
	return s.issuePair(ctx, t)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	hash := HashToken(refreshToken)
	tenantID, err := s.store.RotateRefresh(ctx, hash)
	if err != nil {
		return nil, httpx.Unauthorized("invalid refresh token")
	}
	t, err := s.store.TenantByID(ctx, tenantID)
	if err != nil {
		return nil, httpx.Unauthorized("invalid refresh token")
	}
	return s.issuePair(ctx, t)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.store.RevokeRefresh(ctx, HashToken(refreshToken))
}

func (s *Service) ChangePassword(ctx context.Context, tenantID uuid.UUID, oldPw, newPw string) error {
	if len(newPw) < 8 {
		return httpx.Validation(map[string]string{"new_password": "must be at least 8 characters"})
	}
	t, err := s.store.TenantByID(ctx, tenantID)
	if err != nil {
		return httpx.NotFound("account not found")
	}
	ok, err := VerifyPassword(oldPw, t.PasswordHash)
	if err != nil || !ok {
		return httpx.Unauthorized("current password is incorrect")
	}
	hash, err := HashPassword(newPw)
	if err != nil {
		return err
	}
	return s.store.UpdatePassword(ctx, tenantID, hash)
}

// ForgotPassword always returns nil to avoid leaking which emails exist.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	t, err := s.store.TenantByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil // do not reveal existence
	}
	return s.sendToken(ctx, t, "reset_password", time.Hour, "Reset your MUDAHURUS password")
}

func (s *Service) ResetPassword(ctx context.Context, token, newPw string) error {
	if len(newPw) < 8 {
		return httpx.Validation(map[string]string{"password": "must be at least 8 characters"})
	}
	tenantID, err := s.store.ConsumeAuthToken(ctx, "reset_password", HashToken(token))
	if err != nil {
		return httpx.BadRequest("invalid or expired reset token")
	}
	hash, err := HashPassword(newPw)
	if err != nil {
		return err
	}
	return s.store.UpdatePassword(ctx, tenantID, hash)
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	tenantID, err := s.store.ConsumeAuthToken(ctx, "verify_email", HashToken(token))
	if err != nil {
		return httpx.BadRequest("invalid or expired verification token")
	}
	return s.store.MarkEmailVerified(ctx, tenantID)
}

func (s *Service) UpdateProfile(ctx context.Context, tenantID uuid.UUID, fullName, storeName, phone string) (*Tenant, error) {
	if err := s.store.UpdateProfile(ctx, tenantID, fullName, storeName, phone); err != nil {
		return nil, err
	}
	return s.store.TenantByID(ctx, tenantID)
}

func (s *Service) Me(ctx context.Context, tenantID uuid.UUID) (*Tenant, error) {
	t, err := s.store.TenantByID(ctx, tenantID)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NotFound("account not found")
	}
	return t, err
}

// sendToken creates a single-use token and emails it (log sink in dev).
func (s *Service) sendToken(ctx context.Context, t *Tenant, purpose string, ttl time.Duration, subject string) error {
	raw := uuid.NewString() + "." + uuid.NewString()
	if err := s.store.SaveAuthToken(ctx, t.ID, purpose, HashToken(raw), time.Now().Add(ttl)); err != nil {
		return err
	}
	path := "/verify-email"
	if purpose == "reset_password" {
		path = "/reset"
	}
	return s.mailer.Send(ctx, notify.Message{
		To:      t.Email,
		Subject: subject,
		Body:    "Use this link: " + path + "?token=" + raw,
	})
}
