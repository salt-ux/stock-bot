-- name: CreateStrategyRun :execresult
INSERT INTO strategy_runs (portfolio_id, strategy_name, status)
VALUES (?, ?, 'RUNNING');

-- name: FinishStrategyRun :exec
UPDATE strategy_runs
SET status = ?, finished_at = CURRENT_TIMESTAMP, error_message = ?
WHERE id = ?;

-- name: ListStrategyRunsByPortfolio :many
SELECT id, portfolio_id, strategy_name, status, started_at, finished_at, error_message
FROM strategy_runs
WHERE portfolio_id = ?
ORDER BY started_at DESC
LIMIT ?;
