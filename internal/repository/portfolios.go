package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/salt-ux/stock-bot/internal/repository/sqlc"
)

type PortfoliosRepository struct {
	q sqlc.Querier
}

func (r PortfoliosRepository) Create(ctx context.Context, arg sqlc.CreatePortfolioParams) (int64, error) {
	res, err := r.q.CreatePortfolio(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("create portfolio: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read portfolio id: %w", err)
	}
	return id, nil
}

func (r PortfoliosRepository) GetByID(ctx context.Context, id int64) (sqlc.Portfolio, error) {
	item, err := r.q.GetPortfolioByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return sqlc.Portfolio{}, ErrNotFound
		}
		return sqlc.Portfolio{}, fmt.Errorf("get portfolio by id: %w", err)
	}
	return item, nil
}

func (r PortfoliosRepository) ListByUser(ctx context.Context, userID int64) ([]sqlc.Portfolio, error) {
	items, err := r.q.ListPortfoliosByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list portfolios by user: %w", err)
	}
	return items, nil
}
