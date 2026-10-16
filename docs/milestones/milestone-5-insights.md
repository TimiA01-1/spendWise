# Milestone 5 — Insights

## Overview

Milestone 5 introduces financial insights through a dashboard and reporting endpoints. Users can review account balances, monthly income and expenses, recent transactions, budget utilization, alerts, and category-based financial reports.

All endpoints require JWT authentication and return data belonging to the authenticated user.

## Step 22 — Dashboard

**Endpoint:** `GET /api/v1/dashboard`

Optional query parameter: `month=YYYY-MM`.

The dashboard provides:

- Current account balances grouped by currency.
- Monthly income, expenses, and net income.
- The 10 most recent transactions.
- Monthly budget utilization, remaining amounts, and statuses.
- Number of unread budget alerts.

Financial amounts are calculated separately for each currency.

The dashboard uses a read-only PostgreSQL `REPEATABLE READ` transaction to retrieve a consistent snapshot.

## Step 23 — Reports

### Monthly report

**Endpoint:** `GET /api/v1/reports/monthly?month=2026-10`

Returns:

- Income, expenses, net income, and transaction counts per currency.
- Previous-month expenses.
- Absolute and percentage changes in spending.
- Spending breakdown by category.
- Income breakdown by category.

When the previous month has no spending, the percentage change is returned as `null`.

### Category report

**Endpoint:** `GET /api/v1/reports/categories?from=2026-01-01&to=2026-03-31&type=expense`

Supports:

- Custom inclusive date ranges.
- Expense and income filtering.
- Category totals and transaction counts.
- Category percentages rounded down to one decimal place.
- Separate totals for each currency.

The default transaction type is `expense`.

## Implementation

The milestone introduces:

- `internal/dashboard/` — Dashboard repository, service, handler, and tests.
- `internal/report/` — Reporting repository, service, handler, and tests.
- `pkg/period/` — Shared monthly date-range utility.
- `db/queries/dashboard.sql` — Dashboard aggregation queries.
- `db/queries/reports.sql` — Category aggregation queries.

Database access uses sqlc-generated queries. Monthly reports use a read-only PostgreSQL `REPEATABLE READ` transaction for consistent comparisons.

## Validation

The following commands completed successfully:

```bash
go build ./...
go vet ./...
go test ./...
```

Manual API testing verified authentication, financial totals, currency separation, month boundaries, category percentages, budget and alert integration, input validation, user isolation, and cleanup of test transactions.

## Outcome

Milestone 5 provides authenticated financial dashboards and reporting capabilities, completing the Insights phase of the SpendWise API.