-- name: InsertBudget :one
-- A category_id of all zeros means "no category": an overall budget (stored as NULL).
INSERT INTO budgets (user_id, category_id, amount, alert_threshold)
VALUES (
    sqlc.arg(user_id),
    NULLIF(sqlc.arg(category_id)::uuid, '00000000-0000-0000-0000-000000000000'),
    sqlc.arg(amount),
    sqlc.arg(alert_threshold)
)
RETURNING id;

-- name: ListBudgetsWithSpending :many
-- "spent" is this user's expenses in the given month: all expenses for an
-- overall budget, or only that category's for a category budget.
SELECT b.id,
       COALESCE(b.category_id, '00000000-0000-0000-0000-000000000000'::uuid)::uuid AS category_id,
       COALESCE(c.name, '')::text AS category_name,
       b.amount,
       b.alert_threshold,
       b.created_at,
       COALESCE((
           SELECT SUM(t.amount) FROM transactions t
           WHERE t.user_id = b.user_id
             AND t.type = 'expense'
             AND t.occurred_on >= sqlc.arg(from_date)::date
             AND t.occurred_on <  sqlc.arg(to_date)::date
             AND (b.category_id IS NULL OR t.category_id = b.category_id)
       ), 0)::bigint AS spent
FROM budgets b
LEFT JOIN categories c ON c.id = b.category_id
WHERE b.user_id = sqlc.arg(user_id)
ORDER BY (b.category_id IS NOT NULL), lower(c.name), b.id;

-- name: GetBudgetWithSpending :one
SELECT b.id,
       COALESCE(b.category_id, '00000000-0000-0000-0000-000000000000'::uuid)::uuid AS category_id,
       COALESCE(c.name, '')::text AS category_name,
       b.amount,
       b.alert_threshold,
       b.created_at,
       COALESCE((
           SELECT SUM(t.amount) FROM transactions t
           WHERE t.user_id = b.user_id
             AND t.type = 'expense'
             AND t.occurred_on >= sqlc.arg(from_date)::date
             AND t.occurred_on <  sqlc.arg(to_date)::date
             AND (b.category_id IS NULL OR t.category_id = b.category_id)
       ), 0)::bigint AS spent
FROM budgets b
LEFT JOIN categories c ON c.id = b.category_id
WHERE b.id = sqlc.arg(id) AND b.user_id = sqlc.arg(user_id);

-- name: UpdateBudget :execrows
UPDATE budgets
SET amount = sqlc.arg(amount), alert_threshold = sqlc.arg(alert_threshold), updated_at = now()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteBudget :execrows
DELETE FROM budgets
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);