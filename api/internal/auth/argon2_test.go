package auth

import "testing"

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password not hashed")
	}
	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil || !ok {
		t.Fatalf("verify should succeed: ok=%v err=%v", ok, err)
	}
	bad, err := VerifyPassword("wrong password", hash)
	if err != nil {
		t.Fatalf("verify err: %v", err)
	}
	if bad {
		t.Fatal("verify should fail for wrong password")
	}
}

func TestVerifyInvalidHash(t *testing.T) {
	if _, err := VerifyPassword("x", "not-a-hash"); err == nil {
		t.Fatal("expected error for malformed hash")
	}
}

func TestHashIsSalted(t *testing.T) {
	a, _ := HashPassword("samepw")
	b, _ := HashPassword("samepw")
	if a == b {
		t.Fatal("hashes of the same password must differ (random salt)")
	}
}
