-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :execresult
INSERT INTO users (login_id, password)
VALUES (?, ?);

-- name: GetUserByLoginID :one
SELECT id, login_id, password, created_at
FROM users
WHERE login_id = ?
LIMIT 1;
