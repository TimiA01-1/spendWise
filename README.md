SpendWise — Personal Finance Management API

SpendWise is a RESTful backend API built with Go to help users manage their personal finances, including financial accounts, income, expenses, budgets, and financial insights.

The project is being developed incrementally, with each milestone introducing new backend functionality and engineering practices.

Tech Stack:

Language: Go (Golang)
Web Framework: Gin
Database: PostgreSQL
Database Driver: pgx
SQL Code Generation: sqlc
Authentication: JWT and bcrypt
Database Migrations: golang-migrate
Containerization: Docker and Docker Compose
Testing: Go's built-in testing package


Project Architecture:

SpendWise follows a layered backend architecture:
HTTP Handler → Service → Repository → PostgreSQL

Handlers: Process HTTP requests and return JSON responses.
Services: Implement business logic and validation.
Repositories: Handle database operations through sqlc-generated queries.
Database: Stores persistent application data.



Milestone 1 — Project Foundation:

Initialized the Go project and Git repository.
Organized the application into reusable packages.
Configured environment variables.
Set up PostgreSQL using Docker Compose.
Implemented database migrations.
Configured pgx connection pooling.
Integrated sqlc for type-safe database queries.
Added an HTTP server and health-check endpoint.



Milestone 2 — Authentication:

User registration with input validation.
Secure password hashing using bcrypt.
User login and credential verification.
JWT access-token generation and validation.
Authentication middleware for protected endpoints.
Authenticated user profile retrieval.
Duplicate-email handling.
Automated tests for authentication components.
Multi-stage Docker build for the API.


Milestone 3 - Financial core
Implemented the core financial management features of SpendWise.

### Account Management
- Create financial accounts with an opening balance.
- List and retrieve accounts belonging to the authenticated user.
- Update account information.
- Delete accounts that have no associated transactions.
- Prevent users from accessing or modifying another user's accounts.

### Category Management
- List available income and expense categories.
- Create custom categories.
- Support built-in categories and user-owned categories.
- Validate category ownership and transaction type compatibility.

### Transaction Management
- Create income and expense transactions.
- Retrieve individual transactions.
- List transaction history with filtering and pagination.
- Update transaction details, including amounts, categories, and accounts.
- Delete transactions and reverse their financial effects.
- Automatically adjust account balances when transactions are created, updated, or deleted.

### Financial Data Integrity
- Use PostgreSQL transactions to ensure atomic balance updates.
- Use row-level locking to prevent lost updates during concurrent requests.
- Reverse the original balance effect before applying transaction changes.
- Support moving transactions between accounts safely.
- Validate transaction ownership, amounts, categories, and dates.
- Prevent unauthorized access to financial records.

### Validation and Testing
- Tested account ownership and transaction authorization.
- Tested invalid amounts, dates, and category combinations.
- Verified balance adjustments when changing transaction amounts and types.
- Verified balance restoration after transaction deletion.
- Tested concurrent transaction movements between accounts.
- Successfully ran `go build ./...`, `go vet ./...`, and `go test ./...`.


## Milestone 4 — Budget Engine

Implemented monthly budget management, spending utilization calculations, and automatic budget alerts.

### Budget Management
- Create monthly budgets with category-specific spending limits.
- List, update, and delete budgets.
- Calculate monthly spending and budget utilization percentages.
- Determine budget status based on spending thresholds.
- Validate budget data and enforce user ownership.

### Budget Alerts
- Generate warning alerts when spending reaches 80% of a budget.
- Generate limit-reached alerts when spending reaches or exceeds 100%.
- Prevent duplicate alerts for the same budget, month, and threshold.
- Generate alerts during relevant transaction creation and updates.
- Store alerts atomically with the associated transaction changes.
- Support listing alerts, filtering unread alerts, and marking alerts as read.
- Prevent users from accessing another user's alerts.
- Automatically remove associated alerts when a budget is deleted.

### Validation and Testing
- Verified budget utilization and monthly spending calculations.
- Tested warning and limit-reached alert generation.
- Verified duplicate prevention across repeated threshold crossings.
- Tested alert read tracking, ownership, and invalid requests.
- Verified alert deletion through database cascading.
- Successfully ran `go build ./...`, `go vet ./...`, and `go test ./...`.



## Milestone 5 — Insights

Implemented a financial dashboard and reporting features to help users understand their income, expenses, account balances, spending patterns, and budget performance.

### Financial Dashboard
- Display current account balances grouped by currency.
- Calculate monthly income, expenses, and net income.
- Retrieve the 10 most recent transactions.
- Display monthly budget utilization, remaining amounts, and budget status.
- Show the number of unread budget alerts.
- Support optional month filtering using `YYYY-MM`.
- Use PostgreSQL read-only `REPEATABLE READ` transactions for consistent dashboard data.

### Monthly Financial Reports
- Generate monthly income, expense, and net income summaries.
- Display transaction counts for each currency.
- Compare current-month expenses with the previous month.
- Calculate absolute and percentage changes in spending.
- Provide income and expense breakdowns by category.
- Calculate category percentages to one decimal place, rounded down.
- Return `null` for percentage changes when previous-month spending is zero.

### Category Reports
- Generate category-based reports for custom date ranges.
- Support income and expense filtering.
- Include both start and end dates in the reporting period.
- Calculate category totals, transaction counts, and spending percentages.
- Keep financial totals separated by currency.
- Validate date formats, date ranges, and transaction types.

### Validation and Testing
- Verified dashboard income, expenses, balances, and currency separation.
- Tested recent transaction limits and budget alert integration.
- Verified monthly reports and previous-month comparisons.
- Tested category percentages and inclusive date ranges.
- Confirmed financial totals agree between dashboard and reports.
- Tested invalid requests and authenticated user isolation.
- Successfully ran `go build ./...`, `go vet ./...`, and `go test ./...`.





API Endpoints:

All endpoints except registration, login, and health checks require a JWT access token.

| Method | Endpoint | Description | Authentication |
|---|---|---|---|
| GET | `/health` | Check API health | No |
| POST | `/api/v1/auth/register` | Register a new user | No |
| POST | `/api/v1/auth/login` | Log in and receive a JWT | No |
| GET | `/api/v1/users/me` | Retrieve authenticated user profile | Yes |
| POST | `/api/v1/accounts` | Create a financial account | Yes |
| GET | `/api/v1/accounts` | List user's accounts | Yes |
| GET | `/api/v1/accounts/:id` | Retrieve an account | Yes |
| PATCH | `/api/v1/accounts/:id` | Update an account | Yes |
| DELETE | `/api/v1/accounts/:id` | Delete an account | Yes |
| GET | `/api/v1/categories` | List available categories | Yes |
| POST | `/api/v1/categories` | Create a category | Yes |
| POST | `/api/v1/transactions` | Create a transaction | Yes |
| GET | `/api/v1/transactions` | List transactions | Yes |
| GET | `/api/v1/transactions/:id` | Retrieve a transaction | Yes |
| PATCH | `/api/v1/transactions/:id` | Update a transaction | Yes |
| DELETE | `/api/v1/transactions/:id` | Delete a transaction | Yes |
| POST | `/api/v1/budgets` | Create a monthly budget | Yes |
| GET | `/api/v1/budgets` | List budgets and utilization | Yes |
| PATCH | `/api/v1/budgets/:id` | Update a budget | Yes |
| DELETE | `/api/v1/budgets/:id` | Delete a budget | Yes |
| GET | `/api/v1/alerts` | List budget alerts | Yes |
| PATCH | `/api/v1/alerts/:id/read` | Mark an alert as read | Yes |
| GET | `/api/v1/dashboard` | Retrieve financial dashboard with optional month filter | Yes |
| GET | `/api/v1/reports/monthly` | Retrieve monthly financial report and spending comparison | Yes |
| GET | `/api/v1/reports/categories` | Retrieve category breakdown for a date range | Yes |



### Authentication

For protected endpoints, include the JWT access token in the request header:

```http
Authorization: Bearer <your-access-token>






Getting Started:

Prerequisites:

Git
Docker and Docker Compose
Go (for local development)
sqlc (when regenerating database code)


Installation:
Clone the repository:

git clone https://github.com/TimiA01-1/spendWise.git
cd spendWise

Create your local environment file:
   cp .env.example .env

Configure the environment variables, including a strong JWT_SECRET of at least 32 characters.

Never commit your .env file or production secrets.

Run with Docker:
  docker compose up -d --build

The API is configured to run at:
    http://localhost:8080

Check the health endpoint:
  curl http://localhost:8080/health

Stop the containers:
   docker compose down

Run Tests:
  go test ./...

Compile the Application:
  go build ./...






## Project Documentation

Detailed implementation notes for each completed milestone:

| Milestone | Documentation | Status |
|---|---|---|
| 1 — Project Foundation | [Foundation](docs/milestones/milestone-1-foundation.md) | Complete |
| 2 — Authentication | [Authentication](docs/milestones/milestone-2-authentication.md) | Complete |
| 3 — Financial Core | [Financial Core](docs/milestones/milestone-3-financial-core.md) | Complete |
| 4 — Budget Engine | [Budget Engine](docs/milestones/milestone-4-budget-engine.md) | Complete |
| 5 — Insights | [Insights](docs/milestones/milestone-5-insights.md) | Complete |

