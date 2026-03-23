-- name: CreateFill :execresult
INSERT INTO fills (order_id, fill_price, fill_quantity, fee)
VALUES (?, ?, ?, ?);

-- name: ListFillsByOrder :many
SELECT id, order_id, fill_price, fill_quantity, fee, filled_at
FROM fills
WHERE order_id = ?
ORDER BY filled_at ASC;
