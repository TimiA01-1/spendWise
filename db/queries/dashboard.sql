-- name: DashboardBalances :many
-- Current balance per currency. Different currencies are never added together.
SELECT currency, COALESCE(SUM(balance), 0)::bigint AS balance
FROM accounts
WHERE user_id = sqlc.arg(user_id)
GROUP BY currency
ORDER BY currency;

-- name: DashboardMonthTotals :many
-- Income and expenses per currency for one month. The currency belongs to the
-- account the transaction was recorded on.
SELECT a.currency,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'), 0)::bigint  AS income,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0)::bigint AS expenses
FROM transactions t
JOIN accounts a ON a.id = t.account_id
WHERE t.user_id = sqlc.arg(user_id)
  AND t.occurred_on >= sqlc.arg(from_date)::date
  AND t.occurred_on <  sqlc.arg(to_date)::date
GROUP BY a.currency
ORDER BY a.currency;

-- name: CountUnreadAlerts :one
SELECT COUNT(*)::bigint FROM alerts
WHERE user_id = sqlc.arg(user_id) AND read_at IS NULL;