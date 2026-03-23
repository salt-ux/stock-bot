package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrIDRequired         = errors.New("id is required")
	ErrPasswordTooShort   = errors.New("password must be at least 5 characters")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type CredentialsStore interface {
	Register(id, password string) error
	Authenticate(id, password string) error
}

type FileStore struct {
	path string
	mu   sync.Mutex
}

type userRecord struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Register(id, password string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrIDRequired
	}
	if len(password) < 5 {
		return ErrPasswordTooShort
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists, err := s.readLocked()
	if err != nil {
		return err
	}
	if exists {
		return ErrUserAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	record = userRecord{ID: id, Password: string(hashed)}
	if err := s.writeLocked(record); err != nil {
		return err
	}
	return nil
}

func (s *FileStore) Authenticate(id, password string) error {
	id = strings.TrimSpace(id)
	if id == "" || strings.TrimSpace(password) == "" {
		return ErrInvalidCredentials
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists, err := s.readLocked()
	if err != nil {
		return err
	}
	if !exists {
		return ErrInvalidCredentials
	}
	if record.ID != id {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(record.Password), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (s *FileStore) readLocked() (userRecord, bool, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return userRecord{}, false, nil
		}
		return userRecord{}, false, fmt.Errorf("read auth store: %w", err)
	}

	var record userRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return userRecord{}, false, fmt.Errorf("decode auth store: %w", err)
	}
	if strings.TrimSpace(record.ID) == "" {
		return userRecord{}, false, nil
	}
	return record, true, nil
}

func (s *FileStore) writeLocked(record userRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create auth store directory: %w", err)
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode auth store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write auth store: %w", err)
	}
	return nil
}
