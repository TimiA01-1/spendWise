# Milestone 4 — Budget Engine

## Overview

Milestone 4 introduces monthly budget management, spending utilization calculations, and automatic budget alerts to the SpendWise API.

The milestone consists of two implementation steps:

-  Budgets and monthly utilization
-  Budget alerts and read tracking

## 1. Budget Management

Authenticated users can create and manage monthly spending budgets.

Each budget defines a spending limit associated with a category and month.

### Features

- Create budgets with spending limits.
- List and retrieve budgets.
- Update existing budgets.
- Delete budgets.
- Calculate spending against monthly budget limits.
- Calculate budget utilization percentages.
- Determine budget status based on spending thresholds.
- Enforce user ownership and validate requests.

### Budget API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/budgets` | Create a budget |
| GET | `/api/v1/budgets` | List budgets and utilization |
| PATCH | `/api/v1/budgets/:id` | Update a budget |
| DELETE | `/api/v1/budgets/:id` | Delete a budget |

All budget endpoints require authentication.

## 2. Budget Utilization

Budget utilization is calculated by comparing the amount spent during a month against the configured budget limit.

**Formula:**

`Utilization (%) = (Monthly Spending / Budget Limit) × 100`

For example, if a user has a monthly budget of ₦10,000:

| Spending | Utilization | Interpretation |
|---|---|---|
| ₦5,000 | 50% | Below warning threshold |
| ₦8,000 | 80% | Warning threshold reached |
| ₦10,500 | 105% | Budget limit exceeded |

Calculations are based on the applicable budget category and month.

## 3. Budget Alerts

Budget alerts notify users when spending reaches configured thresholds.

### Alert Levels

- **Warning:** Spending reaches the warning threshold.
- **Limit reached:** Spending reaches or exceeds the budget limit.

### Alert Behavior

- Alerts are generated during relevant transaction operations.
- Alert creation is integrated with the database transaction.
- Budget rows are locked during threshold evaluation to help prevent concurrent duplicate alerts.
- Each budget, month, and alert level has a unique alert record.
- Repeated spending beyond a previously triggered threshold does not generate duplicate alerts.
- Alerts contain spending information captured when the threshold was reached.
- Users can retrieve their alerts and mark them as read.
- Users cannot access or modify another user's alerts.
- Deleting a budget removes its associated alerts through database cascading.

### Alert API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/alerts` | List alerts, including unread filtering |
| PATCH | `/api/v1/alerts/:id/read` | Mark an alert as read |

Both endpoints require authentication.

## 4. Database Changes

### Migration 000005 — Budgets

Introduced the budgets table for storing monthly spending limits and their associated ownership and category information.

### Migration 000006 — Alerts

Introduced the alerts table for storing threshold notifications.

The alerts table includes:

- Budget association
- Alert month and level
- Spending snapshot
- Read status
- Database uniqueness constraint for budget, month, and alert level
- Foreign key relationship with cascading deletion

Database queries are maintained using SQL files and generated Go code through `sqlc`.

## 5. Project Structure

The implementation includes the following components:

```text
internal/
├── budget/
│   ├── model.go
│   ├── repository.go
│   ├── service.go
│   ├── handler.go
│   ├── alerts.go
│   └── model_test.go
├── alert/
│   ├── model.go
│   ├── repository.go
│   ├── service.go
│   ├── handler.go
│   └── model_test.go
└── transaction/
    └── repository.go

db/
└── queries/
    ├── budgets.sql
    └── alerts.sql

migrations/
├── 000005_create_budgets.up.sql
├── 000005_create_budgets.down.sql
├── 000006_create_alerts.up.sql
└── 000006_create_alerts.down.sql
```

## 6. Verification and Testing

The following project-wide commands completed successfully:

```bash
go build ./...
go vet ./...
go test ./...
```

### Manual API Verification

The following scenarios were tested:

- Budget creation and management
- Monthly spending utilization
- Budget validation and ownership restrictions
- Warning alert generation at 80% utilization
- Limit-reached alert generation above 100% utilization
- Prevention of duplicate threshold alerts
- Separate alerts for different months
- Alert generation after transaction updates
- Listing and filtering unread alerts
- Marking alerts as read
- User isolation and authorization
- Invalid alert identifiers and query parameters
- Cascading deletion of alerts when a budget is deleted

Budget and alert model tests passed. Other packages without test files were compiled, and transaction-alert integration was verified through manual API testing.

