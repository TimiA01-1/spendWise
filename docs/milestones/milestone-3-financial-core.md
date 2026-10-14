# Milestone 3 — Financial Core

## 1. Overview

Milestone 3 focused on implementing the core financial management features of SpendWise.

The goal was to enable authenticated users to manage financial accounts, organize income and expenses into categories, and record transactions while maintaining accurate account balances.

**Status:** Implementation completed and validated.

## 2. Technologies Used

- **Go and Gin:** API development and HTTP request handling.
- **PostgreSQL:** Persistent financial data storage.
- **pgx and pgxpool:** Database operations and connection pooling.
- **sqlc:** Type-safe database queries.
- **JWT middleware:** Authentication and user identification.
- **Docker Compose:** Local development infrastructure.

## 3. Account Management

Implemented the following endpoints:

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/accounts` | Create an account |
| GET | `/api/v1/accounts` | List the user's accounts |
| GET | `/api/v1/accounts/:id` | Retrieve an account |
| PATCH | `/api/v1/accounts/:id` | Update account information |
| DELETE | `/api/v1/accounts/:id` | Delete an eligible account |

Accounts belong to individual authenticated users.

The implementation supports account creation with an opening balance and prevents users from accessing another user's accounts.

Account deletion is restricted when associated transactions exist, protecting financial history.

## 4. Category Management

Implemented two category endpoints:

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/categories` | List available categories |
| POST | `/api/v1/categories` | Create a custom category |

Categories organize financial activity into income and expense types.

The implementation supports built-in categories and user-owned categories.

Transaction validation checks category ownership and ensures the category type matches the transaction type.

## 5. Transaction Management

Implemented the following endpoints:

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/transactions` | Create a transaction |
| GET | `/api/v1/transactions` | List transactions |
| GET | `/api/v1/transactions/:id` | Retrieve a transaction |
| PATCH | `/api/v1/transactions/:id` | Update a transaction |
| DELETE | `/api/v1/transactions/:id` | Delete a transaction |

Transactions represent income or expenses associated with a financial account.

The implementation includes transaction history retrieval, filtering, pagination, input validation, and user ownership checks.

## 6. Account Balance Calculations

Transaction operations automatically update account balances.

The balance effect depends on the transaction type:

- **Income:** Increases the account balance.
- **Expense:** Decreases the account balance.

### Creating a Transaction

When a transaction is created, its signed amount is applied to the associated account balance.

For example, recording an expense of ₦15,000 reduces the account balance by ₦15,000.

### Updating a Transaction

Updating a transaction requires reversing the original financial effect before applying the new effect.

For example:

1. An expense of ₦15,000 reduces the balance by ₦15,000.
2. The expense is updated to ₦20,000.
3. The original ₦15,000 deduction is reversed.
4. The new ₦20,000 deduction is applied.

This results in the correct net balance change.

### Moving a Transaction Between Accounts

When a transaction moves between accounts:

1. Reverse its original effect on the previous account.
2. Apply its effect to the destination account.
3. Save the updated transaction.

Both account balance adjustments occur within the same database transaction.

### Deleting a Transaction

Deleting a transaction reverses its financial effect.

For example, deleting a ₦20,000 expense restores ₦20,000 to the associated account balance.

## 7. Database Transactions and Concurrency

Used PostgreSQL database transactions to keep financial operations atomic.

A transaction update or deletion must update both the financial record and the affected account balances consistently.

If an operation fails, its database changes are rolled back.

### Row-Level Locking

Used PostgreSQL row-level locks to coordinate concurrent operations.

Transaction updates lock the existing transaction record before modifying it.

Affected account records are also locked before balance changes.

### Consistent Lock Ordering

When two accounts are involved, the implementation locks account rows in a consistent UUID order.

This reduces the risk of deadlocks when concurrent requests move transactions between accounts.

## 8. Validation and Security

Implemented validation for:

- Positive transaction amounts and maximum allowed amounts.
- Valid income and expense types.
- Transaction dates.
- Transaction notes.
- Category ownership and type compatibility.
- Account ownership.
- Transaction ownership.

Users cannot retrieve, update, or delete another user's financial transactions.

Unauthorized access to another user's transaction returns a not-found response.

## 9. Manual Testing

Performed manual API tests using curl.

### Transaction Updates

Verified that:

- Increasing an expense amount adjusts the balance correctly.
- Updating a note does not change the balance.
- Changing a transaction from expense to income applies the correct financial effect.
- Moving a transaction between accounts restores the original account's balance and updates the destination account.
- Invalid category and transaction-type combinations are rejected.
- Attempts to move transactions into another user's account are rejected.

### Transaction Deletion

Verified that:

- Deleting a transaction reverses its balance effect.
- Deleting another user's transaction is rejected.
- Deleting an already-deleted transaction returns a not-found response.

### Concurrent Operations

Created ten expense transactions totaling ₦550.

Moved the transactions between two accounts using concurrent requests.

Verified that the combined account balance remained consistent.

After deleting the transactions, both account balances returned to their original values.

These checks provide useful manual validation but do not replace automated concurrency and integration tests.

## 10. Build and Automated Test Results

Successfully executed:

```bash
go build ./...
go vet ./...
go test ./...
```

The application compiled successfully, Go's static analysis completed without reported issues, and the existing authentication tests passed.

At the end of Milestone 3, the transaction package did not yet contain automated test files.

Additional automated financial-core tests are planned for Milestone 7 — Production Readiness.

## 11. Challenges and Solutions

### Maintaining Accurate Balances

**Challenge:** Editing or deleting a transaction could produce incorrect balances if the original financial effect was not reversed.

**Solution:** Implemented balance reversal before applying updates or completing deletions.

### Handling Concurrent Requests

**Challenge:** Multiple requests could attempt to update the same accounts simultaneously.

**Solution:** Used PostgreSQL transactions and row-level locking to coordinate financial updates.

### Avoiding Deadlocks

**Challenge:** Concurrent transactions moving between different accounts could acquire locks in conflicting orders.

**Solution:** Acquired account locks in a consistent UUID order.

### Preventing Unauthorized Access

**Challenge:** Users must not access or modify another user's financial records.

**Solution:** Applied authenticated-user ownership checks to account, category, and transaction operations.

## 12. What I Learned

During this milestone, I gained practical experience with:

- Designing RESTful CRUD endpoints in Go.
- Organizing handlers, services, and repositories.
- Writing SQL queries and generating Go code with sqlc.
- Implementing financial business rules.
- Maintaining account balances consistently.
- Using PostgreSQL transactions and row-level locks.
- Understanding concurrency and deadlock prevention.
- Validating user input and resource ownership.
- Testing APIs with curl.
- Debugging financial data consistency problems.

## 13. Milestone Outcome

Successfully implemented the Financial Core of SpendWise.

Authenticated users can manage financial accounts, organize transactions into categories, and create, retrieve, update, or delete income and expense records.

Account balances are updated automatically, with database transactions and locking used to maintain consistency.

