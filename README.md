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







API Endpoints:

The following endpoints are implemented:

Method  Endpoint  Description  Authentication

GET  /health  Check API health   No

POST /api/v1/auth/register  Register a user No

POST /api/v1/auth/login Log in and receive a JWT  No

GET /api/v1/users/me Retrieve the current user's profile  Yes






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