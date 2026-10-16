-- name: ReportCategoryTotals :many
-- Totals per currency, type (income/expense) and category for a date range.
-- Each currency is kept separate because amounts in different currencies must
-- never be added together. Biggest categories first.
SELECT a.currency,
       t.type,
       c.id   AS category_id,
       c.name AS category_name,
       SUM(t.amount)::bigint AS total,
       COUNT(*)::bigint      AS tx_count
FROM transactions t
JOIN accounts   a ON a.id = t.account_id
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = sqlc.arg(user_id)
  AND t.occurred_on >= sqlc.arg(from_date)::date
  AND t.occurred_on <  sqlc.arg(to_date)::date
GROUP BY a.currency, t.type, c.id, c.name
ORDER BY a.currency, t.type, total DESC, c.name;