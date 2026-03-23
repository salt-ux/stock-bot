package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/salt-ux/stock-bot/internal/repository/sqlc"
)

type UsersRepository struct {
	q sqlc.Querier
}

func (r UsersRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountUsers(ctx)
}

func (r UsersRepository) Create(ctx context.Context, loginID, password string) (int64, error) {
	res, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		LoginID:  loginID,
		Password: password,
	})
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read user id: %w", err)
	}
	return id, nil
}

func (r UsersRepository) GetByLoginID(ctx context.Context, loginID string) (sqlc.User, error) {
	user, err := r.q.GetUserByLoginID(ctx, loginID)
	if err != nil {
		if err == sql.ErrNoRows {
			return sqlc.User{}, ErrNotFound
		}
		return sqlc.User{}, fmt.Errorf("get user by login id: %w", err)
	}
	return user, nil
}
