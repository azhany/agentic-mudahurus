package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	m := NewTokenManager("access-secret", "refresh-secret", 15*time.Minute, time.Hour)
	tid := uuid.New()
	tok, exp, err := m.IssueAccess(tid, "seller", "ali")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if exp.Before(time.Now()) {
		t.Fatal("expiry in the past")
	}
	claims, err := m.ParseAccess(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.TenantID != tid.String() || claims.Role != "seller" || claims.Username != "ali" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestRejectExpiredToken(t *testing.T) {
	m := NewTokenManager("s", "r", -time.Minute, time.Hour) // already expired
	tok, _, _ := m.IssueAccess(uuid.New(), "seller", "x")
	if _, err := m.ParseAccess(tok); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestRejectTamperedSecret(t *testing.T) {
	m1 := NewTokenManager("secret-a", "r", time.Minute, time.Hour)
	m2 := NewTokenManager("secret-b", "r", time.Minute, time.Hour)
	tok, _, _ := m1.IssueAccess(uuid.New(), "seller", "x")
	if _, err := m2.ParseAccess(tok); err == nil {
		t.Fatal("token signed with a different secret must be rejected")
	}
}

func TestRefreshTokenHashing(t *testing.T) {
	m := NewTokenManager("s", "r", time.Minute, time.Hour)
	tok, hash, exp, err := m.IssueRefresh()
	if err != nil {
		t.Fatalf("issue refresh: %v", err)
	}
	if tok == hash {
		t.Fatal("stored hash must differ from the opaque token")
	}
	if HashToken(tok) != hash {
		t.Fatal("hash not reproducible")
	}
	if exp.Before(time.Now()) {
		t.Fatal("refresh expiry in the past")
	}
}
