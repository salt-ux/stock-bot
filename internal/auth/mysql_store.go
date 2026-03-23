package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"

	"github.com/salt-ux/stock-bot/internal/config"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(cfg config.DBConfig) (*MySQLStore, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	store := &MySQLStore{db: db}
	if err := store.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) Register(id, password string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrIDRequired
	}
	if len(password) < 5 {
		return ErrPasswordTooShort
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return ErrUserAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO users (login_id, password) VALUES (?, ?)`,
		id,
		string(hashed),
	); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *MySQLStore) Authenticate(id, password string) error {
	id = strings.TrimSpace(id)
	if id == "" || strings.TrimSpace(password) == "" {
		return ErrInvalidCredentials
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var storedPassword string
	err := s.db.QueryRowContext(ctx,
		`SELECT password FROM users WHERE login_id = ? LIMIT 1`,
		id,
	).Scan(&storedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("select user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (s *MySQLStore) ensureSchema(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  login_id VARCHAR(100) NOT NULL UNIQUE,
  password VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("ensure auth schema: %w", err)
	}
	return nil
}
