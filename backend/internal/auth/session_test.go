package auth

import "testing"

func TestManagerTokenRoundTrip(t *testing.T) {
	manager := NewManager("admin@example.com", "secret", "session-secret")

	token, err := manager.CreateToken()
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	user, err := manager.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if user.Email != "admin@example.com" {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestValidateCredentials(t *testing.T) {
	manager := NewManager("admin@example.com", "secret", "session-secret")

	if !manager.ValidateCredentials("admin@example.com", "secret") {
		t.Fatal("expected credentials to validate")
	}
	if manager.ValidateCredentials("admin@example.com", "bad") {
		t.Fatal("expected invalid password to fail")
	}
}
