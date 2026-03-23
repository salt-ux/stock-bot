-- name: UpsertPosition :exec
INSERT INTO positions (portfolio_id, symbol, quantity, avg_price, last_price)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  quantity = VALUES(quantity),
  avg_price = VALUES(avg_price),
  last_price = VALUES(last_price),
  updated_at = CURRENT_TIMESTAMP;

-- name: ListPositionsByPortfolio :many
SELECT id, portfolio_id, symbol, quantity, avg_price, last_price, updated_at
FROM positions
WHERE portfolio_id = ?
ORDER BY symbol ASC;
