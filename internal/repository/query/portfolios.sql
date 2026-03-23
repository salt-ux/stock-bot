-- name: CreatePortfolio :execresult
INSERT INTO portfolios (user_id, name, cash_balance)
VALUES (?, ?, ?);

-- name: ListPortfoliosByUser :many
SELECT id, user_id, name, cash_balance, created_at, updated_at
FROM portfolios
WHERE user_id = ?
ORDER BY id DESC;

-- name: GetPortfolioByID :one
SELECT id, user_id, name, cash_balance, created_at, updated_at
FROM portfolios
WHERE id = ?
LIMIT 1;
