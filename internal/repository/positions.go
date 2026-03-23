package repository

import (
	"context"
	"fmt"

	"github.com/salt-ux/stock-bot/internal/repository/sqlc"
)

type PositionsRepository struct {
	q sqlc.Querier
}

func (r PositionsRepository) Upsert(ctx context.Context, arg sqlc.UpsertPositionParams) error {
	if err := r.q.UpsertPosition(ctx, arg); err != nil {
		return fmt.Errorf("upsert position: %w", err)
	}
	return nil
}

func (r PositionsRepository) ListByPortfolio(ctx context.Context, portfolioID int64) ([]sqlc.Position, error) {
	items, err := r.q.ListPositionsByPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("list positions by portfolio: %w", err)
	}
	return items, nil
}
