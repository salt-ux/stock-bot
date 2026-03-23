package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/salt-ux/stock-bot/internal/repository/sqlc"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Users() UsersRepository {
	return UsersRepository{q: sqlc.New(s.db)}
}

func (s *Store) Portfolios() PortfoliosRepository {
	return PortfoliosRepository{q: sqlc.New(s.db)}
}

func (s *Store) Positions() PositionsRepository {
	return PositionsRepository{q: sqlc.New(s.db)}
}

func (s *Store) Orders() OrdersRepository {
	return OrdersRepository{q: sqlc.New(s.db)}
}

func (s *Store) Fills() FillsRepository {
	return FillsRepository{q: sqlc.New(s.db)}
}

func (s *Store) StrategyRuns() StrategyRunsRepository {
	return StrategyRunsRepository{q: sqlc.New(s.db)}
}

func (s *Store) InTx(ctx context.Context, fn func(txStore *TxStore) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txStore := &TxStore{q: sqlc.New(tx)}
	if err := fn(txStore); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

type TxStore struct {
	q *sqlc.Queries
}

func (t *TxStore) Users() UsersRepository {
	return UsersRepository{q: t.q}
}

func (t *TxStore) Portfolios() PortfoliosRepository {
	return PortfoliosRepository{q: t.q}
}

func (t *TxStore) Positions() PositionsRepository {
	return PositionsRepository{q: t.q}
}

func (t *TxStore) Orders() OrdersRepository {
	return OrdersRepository{q: t.q}
}

func (t *TxStore) Fills() FillsRepository {
	return FillsRepository{q: t.q}
}

func (t *TxStore) StrategyRuns() StrategyRunsRepository {
	return StrategyRunsRepository{q: t.q}
}
