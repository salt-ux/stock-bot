package auth

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestRegisterAndAuthenticate(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "user.json"))

	if err := store.Register("admin", "12345"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := store.Authenticate("admin", "12345"); err != nil {
		t.Fatalf("authenticate failed: %v", err)
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "user.json"))

	err := store.Register("admin", "1234")
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestRegisterOnlyOneUser(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "user.json"))

	if err := store.Register("admin", "12345"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	err := store.Register("other", "67890")
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestAuthenticateInvalid(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "user.json"))

	if err := store.Register("admin", "12345"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	err := store.Authenticate("admin", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
