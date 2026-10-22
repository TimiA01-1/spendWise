-- name: InsertRecurring :one
-- An end_date of 0001-01-01 means "no end date" (stored as NULL).
INSERT INTO recurring_transactions
    (user_id, account_id, category_id, type, amount, note, frequency, start_date, end_date, next_run_on)
VALUES (
    sqlc.arg(user_id), sqlc.arg(account_id), sqlc.arg(category_id), sqlc.arg(type),
    sqlc.arg(amount), sqlc.arg(note), sqlc.arg(frequency), sqlc.arg(start_date),
    NULLIF(sqlc.arg(end_date)::date, '0001-01-01'),
    sqlc.arg(start_date)
)
RETURNING id;

-- name: ListRecurringForUser :many
SELECT r.id, r.account_id, r.category_id, c.name AS category_name, r.type, r.amount, r.note, r.frequency,
       r.start_date, COALESCE(r.end_date, '0001-01-01'::date)::date AS end_date,
       r.next_index, r.next_run_on, r.status, r.runs, r.last_error, r.created_at
FROM recurring_transactions r
JOIN categories c ON c.id = r.category_id
WHERE r.user_id = sqlc.arg(user_id)
ORDER BY r.created_at, r.id;

-- name: GetRecurringForUser :one
SELECT r.id, r.account_id, r.category_id, c.name AS category_name, r.type, r.amount, r.note, r.frequency,
       r.start_date, COALESCE(r.end_date, '0001-01-01'::date)::date AS end_date,
       r.next_index, r.next_run_on, r.status, r.runs, r.last_error, r.created_at
FROM recurring_transactions r
JOIN categories c ON c.id = r.category_id
WHERE r.id = sqlc.arg(id) AND r.user_id = sqlc.arg(user_id);

-- name: LockRecurringForUser :one
-- Same columns as GetRecurringForUser, but locks the rule until the surrounding
-- DB transaction ends, so an edit can't collide with the background worker.
SELECT r.id, r.account_id, r.category_id, c.name AS category_name, r.type, r.amount, r.note, r.frequency,
       r.start_date, COALESCE(r.end_date, '0001-01-01'::date)::date AS end_date,
       r.next_index, r.next_run_on, r.status, r.runs, r.last_error, r.created_at
FROM recurring_transactions r
JOIN categories c ON c.id = r.category_id
WHERE r.id = sqlc.arg(id) AND r.user_id = sqlc.arg(user_id)
FOR UPDATE OF r;

-- name: UpdateRecurring :execrows
UPDATE recurring_transactions
SET amount      = sqlc.arg(amount),
    note        = sqlc.arg(note),
    end_date    = NULLIF(sqlc.arg(end_date)::date, '0001-01-01'),
    status      = sqlc.arg(status),
    next_index  = sqlc.arg(next_index),
    next_run_on = sqlc.arg(next_run_on),
    last_error  = '',
    retry_at    = now(),
    updated_at  = now()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteRecurring :execrows
DELETE FROM recurring_transactions
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ClaimDueRecurring :one
-- The worker's heart. Picks ONE rule that is due and locks it. SKIP LOCKED means
-- that if another worker (or another server instance) already holds a rule, we
-- skip it instead of waiting, so two workers never process the same rule.
SELECT id, user_id, account_id, category_id, type, amount, note, frequency,
       start_date, COALESCE(end_date, '0001-01-01'::date)::date AS end_date,
       next_index, next_run_on, runs
FROM recurring_transactions
WHERE status = 'active'
  AND next_run_on <= sqlc.arg(today)::date
  AND (end_date IS NULL OR next_run_on <= end_date)
  AND retry_at <= now()
ORDER BY next_run_on, id
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: AdvanceRecurring :exec
UPDATE recurring_transactions
SET next_index  = sqlc.arg(next_index),
    next_run_on = sqlc.arg(next_run_on),
    status      = sqlc.arg(status),
    runs        = runs + 1,
    last_error  = '',
    updated_at  = now()
WHERE id = sqlc.arg(id);

-- name: LinkTransactionToRecurring :exec
UPDATE transactions
SET recurring_id = sqlc.arg(recurring_id)::uuid, recurring_for = sqlc.arg(recurring_for)::date
WHERE id = sqlc.arg(id);

-- name: FailRecurring :exec
-- Remember why a rule failed and keep it out of the worker's way for 15 minutes,
-- so one broken rule can never block all the others.
UPDATE recurring_transactions
SET last_error = sqlc.arg(last_error),
    retry_at   = now() + interval '15 minutes',
    updated_at = now()
WHERE id = sqlc.arg(id);