package repository

import (
	"context"
	"fmt"

	"github.com/salt-ux/stock-bot/internal/repository/sqlc"
)

type StrategyRunsRepository struct {
	q sqlc.Querier
}

func (r StrategyRunsRepository) Create(ctx context.Context, arg sqlc.CreateStrategyRunParams) (int64, error) {
	res, err := r.q.CreateStrategyRun(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("create strategy run: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read strategy run id: %w", err)
	}
	return id, nil
}

func (r StrategyRunsRepository) Finish(ctx context.Context, arg sqlc.FinishStrategyRunParams) error {
	if err := r.q.FinishStrategyRun(ctx, arg); err != nil {
		return fmt.Errorf("finish strategy run: %w", err)
	}
	return nil
}

func (r StrategyRunsRepository) ListByPortfolio(ctx context.Context, portfolioID int64, limit int32) ([]sqlc.StrategyRun, error) {
	items, err := r.q.ListStrategyRunsByPortfolio(ctx, sqlc.ListStrategyRunsByPortfolioParams{
		PortfolioID: portfolioID,
		Limit:       limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list strategy runs by portfolio: %w", err)
	}
	return items, nil
}
