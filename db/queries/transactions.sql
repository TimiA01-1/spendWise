-- name: LockAccountForUser :one
-- Locks the account row until the surrounding DB transaction ends, so two
-- simultaneous requests can't both read the same balance and overwrite each other.
SELECT id FROM accounts
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
FOR UPDATE;

-- name: GetCategoryForUser :one
SELECT id, name, type FROM categories
WHERE id = sqlc.arg(id)::uuid
  AND (user_id IS NULL OR user_id = sqlc.arg(user_id)::uuid);

-- name: InsertTransaction :one
INSERT INTO transactions (user_id, account_id, category_id, type, amount, note, occurred_on)
VALUES (
    sqlc.arg(user_id), sqlc.arg(account_id), sqlc.arg(category_id),
    sqlc.arg(type), sqlc.arg(amount), sqlc.arg(note), sqlc.arg(occurred_on)
)
RETURNING *;

-- name: AdjustAccountBalance :exec
UPDATE accounts
SET balance = balance + sqlc.arg(delta)::bigint, updated_at = now()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListTransactionsForUser :many
SELECT t.id, t.account_id, t.category_id, c.name AS category_name,
       t.type, t.amount, t.note, t.occurred_on, t.created_at
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = sqlc.arg(user_id)::uuid
  AND (sqlc.arg(account_id)::uuid  = '00000000-0000-0000-0000-000000000000' OR t.account_id  = sqlc.arg(account_id)::uuid)
  AND (sqlc.arg(category_id)::uuid = '00000000-0000-0000-0000-000000000000' OR t.category_id = sqlc.arg(category_id)::uuid)
  AND (sqlc.arg(type)::text = '' OR t.type = sqlc.arg(type)::text)
  AND t.occurred_on >= sqlc.arg(from_date)::date
  AND t.occurred_on <= sqlc.arg(to_date)::date
ORDER BY t.occurred_on DESC, t.created_at DESC, t.id
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;