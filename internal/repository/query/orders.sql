-- name: CreateOrder :execresult
INSERT INTO orders (
  portfolio_id, symbol, side, order_type, status, quantity, price, external_order_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = ?, filled_at = ?
WHERE id = ?;

-- name: GetOrderByID :one
SELECT id, portfolio_id, symbol, side, order_type, status, quantity, price,
       external_order_id, submitted_at, filled_at
FROM orders
WHERE id = ?
LIMIT 1;

-- name: ListOrdersByPortfolio :many
SELECT id, portfolio_id, symbol, side, order_type, status, quantity, price,
       external_order_id, submitted_at, filled_at
FROM orders
WHERE portfolio_id = ?
ORDER BY submitted_at DESC
LIMIT ?;
