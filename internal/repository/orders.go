package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/salt-ux/stock-bot/internal/repository/sqlc"
)

type OrdersRepository struct {
	q sqlc.Querier
}

func (r OrdersRepository) Create(ctx context.Context, arg sqlc.CreateOrderParams) (int64, error) {
	res, err := r.q.CreateOrder(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("create order: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read order id: %w", err)
	}
	return id, nil
}

func (r OrdersRepository) GetByID(ctx context.Context, id int64) (sqlc.Order, error) {
	item, err := r.q.GetOrderByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return sqlc.Order{}, ErrNotFound
		}
		return sqlc.Order{}, fmt.Errorf("get order by id: %w", err)
	}
	return item, nil
}

func (r OrdersRepository) ListByPortfolio(ctx context.Context, portfolioID int64, limit int32) ([]sqlc.Order, error) {
	items, err := r.q.ListOrdersByPortfolio(ctx, sqlc.ListOrdersByPortfolioParams{
		PortfolioID: portfolioID,
		Limit:       limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list orders by portfolio: %w", err)
	}
	return items, nil
}

func (r OrdersRepository) UpdateStatus(ctx context.Context, arg sqlc.UpdateOrderStatusParams) error {
	if err := r.q.UpdateOrderStatus(ctx, arg); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return nil
}
