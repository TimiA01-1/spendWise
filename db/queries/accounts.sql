-- name: CreateAccount :one 
INSERT INTO accounts(user_id, name, type, currency, balance)
VALUES(sqlc.arg(user_id), sqlc.arg(name), sqlc.arg(type), sqlc.arg(currency), sqlc.arg(balance))
RETURNING *;

-- name: ListAccountsByUser :many
SELECT * FROM accounts
WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at, id;

-- name: GetAccountForUser :one 
SELECT * FROM accounts
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: UpdateAccount :one 
UPDATE accounts
SET name = sqlc.arg(name), type = sqlc.arg(type), updated_at = now()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: DeleteAccount :execrows 
DELETE FROM accounts
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);