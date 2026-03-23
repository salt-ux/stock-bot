package repository

import (
	"context"
	"fmt"

	"github.com/salt-ux/stock-bot/internal/repository/sqlc"
)

type FillsRepository struct {
	q sqlc.Querier
}

func (r FillsRepository) Create(ctx context.Context, arg sqlc.CreateFillParams) (int64, error) {
	res, err := r.q.CreateFill(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("create fill: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read fill id: %w", err)
	}
	return id, nil
}

func (r FillsRepository) ListByOrder(ctx context.Context, orderID int64) ([]sqlc.Fill, error) {
	items, err := r.q.ListFillsByOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("list fills by order: %w", err)
	}
	return items, nil
}
