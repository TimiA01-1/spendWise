# Milestone 6 — Automation

## Overview

Milestone 6 introduces recurring transaction automation to the SpendWise API. Users can configure scheduled income and expense transactions, while a background worker automatically processes due transactions and maintains account balances.

## Features Implemented

### 1. Recurring Transaction Management

- Create recurring income and expense rules.
- List recurring transactions belonging to the authenticated user.
- Update transaction amounts, notes, end dates, and statuses.
- Pause and resume active recurring rules.
- Delete recurring rules without deleting previously recorded transactions.
- Automatically mark completed recurring rules as finished.

### 2. Recurrence Scheduling

Supported frequencies:

- Daily
- Weekly
- Monthly
- Yearly

The scheduling engine handles month-end dates, leap years, and missed occurrences without permanently shifting the original schedule.

### 3. Background Worker

- Automatically checks for due recurring transactions.
- Uses a configurable `RECURRING_INTERVAL` environment variable.
- Processes overdue occurrences when the application starts.
- Creates financial transactions and updates account balances.
- Advances recurring schedules after successful processing.
- Uses database transactions and uniqueness constraints to prevent duplicate processing.

### 4. Security and Validation

- JWT authentication protects recurring transaction endpoints.
- Users cannot access, update, or delete another user's recurring rules.
- Invalid amounts, frequencies, dates, and transaction types are rejected.
- Recurring rules must use categories compatible with their transaction type.
- Finished rules cannot be modified.

## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/recurring-transactions` | Create a recurring rule |
| GET | `/api/v1/recurring-transactions` | List recurring rules |
| PATCH | `/api/v1/recurring-transactions/:id` | Update, pause, or resume a rule |
| DELETE | `/api/v1/recurring-transactions/:id` | Delete a recurring rule |

All endpoints require authentication.

## Database Changes

Migration: `000007_create_recurring`

Introduced the `recurring_transactions` table, scheduling metadata, transaction associations, and a uniqueness index to prevent duplicate scheduled transactions.

## Testing and Verification

The following checks were completed successfully:

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- Recurring transaction validation tests.
- Daily recurring transaction catch-up processing.
- Monthly scheduling across months with different numbers of days.
- Duplicate prevention after restarting the worker.
- Pause, resume, and recurring rule updates.
- Automatic completion of rules with end dates.
- Prevention of modifications to finished rules.
- User authorization and ownership checks.
- Rule deletion with preservation of transaction history.
- Account balance restoration after deleting test transactions.

## Configuration

The background worker uses the following environment variable:

```env
RECURRING_INTERVAL=1m
```

The worker checks for due recurring transactions every minute while the API is running.

## Outcome

Milestone 6 delivers a working recurring transaction automation system integrated with SpendWise's existing financial transaction infrastructure.

The implementation has passed unit tests, project-wide build and static analysis checks, and manual API and database integration tests.
