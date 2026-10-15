-- name: LockBudgetsForCategory :many
-- Locks the budgets an expense in this category counts toward (the overall
-- budget and that category's budget) until the surrounding DB transaction
-- ends. Always locked in id order so two requests can't deadlock each other.
SELECT id FROM budgets
WHERE user_id = sqlc.arg(user_id)
  AND (category_id IS NULL OR category_id = sqlc.arg(category_id)::uuid)
ORDER BY id
FOR UPDATE;

-- name: ListAffectedBudgets :many
-- The same budgets, with their spending for the month that contains the
-- transaction date.
SELECT b.id, b.amount, b.alert_threshold,
       COALESCE((
           SELECT SUM(t.amount) FROM transactions t
           WHERE t.user_id = b.user_id
             AND t.type = 'expense'
             AND t.occurred_on >= sqlc.arg(from_date)::date
             AND t.occurred_on <  sqlc.arg(to_date)::date
             AND (b.category_id IS NULL OR t.category_id = b.category_id)
       ), 0)::bigint AS spent
FROM budgets b
WHERE b.user_id = sqlc.arg(user_id)
  AND (b.category_id IS NULL OR b.category_id = sqlc.arg(category_id)::uuid)
ORDER BY b.id;

-- name: InsertAlert :execrows
-- ON CONFLICT DO NOTHING + the UNIQUE (budget_id, month, level) constraint
-- means a second attempt for the same budget/month/level silently does nothing.
INSERT INTO alerts (user_id, budget_id, month, level, amount, spent, percent_used)
VALUES (
    sqlc.arg(user_id), sqlc.arg(budget_id), sqlc.arg(month),
    sqlc.arg(level), sqlc.arg(amount), sqlc.arg(spent), sqlc.arg(percent_used)
)
ON CONFLICT (budget_id, month, level) DO NOTHING;

-- name: ListAlertsForUser :many
SELECT a.id, a.budget_id,
       COALESCE(c.name, 'All spending')::text AS category_name,
       a.month, a.level, a.amount, a.spent, a.percent_used,
       (a.read_at IS NOT NULL)::boolean AS is_read,
       a.created_at
FROM alerts a
JOIN budgets b ON b.id = a.budget_id
LEFT JOIN categories c ON c.id = b.category_id
WHERE a.user_id = sqlc.arg(user_id)
  AND (NOT sqlc.arg(unread_only)::boolean OR a.read_at IS NULL)
ORDER BY a.created_at DESC, a.id
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;

-- name: MarkAlertRead :execrows
UPDATE alerts SET read_at = COALESCE(read_at, now())
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);