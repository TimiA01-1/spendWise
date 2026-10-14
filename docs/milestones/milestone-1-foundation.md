# Milestone 1 — Project Foundation

## 1. Overview

The first milestone of SpendWise focused on establishing a reliable backend development environment and project architecture.

The goal was to prepare the application for future features such as authentication, financial account management, transactions, budgets, and reporting.

**Status:** Completed

## 2. Technologies Used

- **Go (Golang):** Backend programming language.
- **Gin:** HTTP web framework.
- **PostgreSQL:** Relational database.
- **Docker and Docker Compose:** Containerized database and application services.
- **pgx:** PostgreSQL driver and connection pooling.
- **sqlc:** Generates type-safe Go code from SQL queries.
- **golang-migrate:** Manages database schema migrations.
- **Git and GitHub:** Version control and source-code hosting.

## 3. Project Initialization

Initialized the SpendWise project using Go modules and Git.

```bash
go mod init spendWise
git init
```

Created a modular project structure to separate responsibilities and support future development.

The application uses a layered architecture:

**HTTP Handler → Service → Repository → PostgreSQL**

This separates HTTP request handling, business logic, and database operations.

## 4. Environment Configuration

Implemented environment-based configuration to manage application settings.

Configuration includes:

- Application environment.
- HTTP server port.
- PostgreSQL connection URL.
- JWT secret and token expiration settings.

A `.env.example` file provides a template for local setup.

Sensitive environment variables are kept outside version control.

## 5. PostgreSQL and Docker Setup

Configured PostgreSQL 16 using Docker Compose.

During development, the PostgreSQL host port was changed from `5432` to `5433` to resolve a local port conflict.

The resulting configuration uses:

- **Host connection:** `localhost:5433`
- **Container connection:** `postgres:5432`

Configured a PostgreSQL health check and persistent Docker volume.

Verified database availability using Docker Compose and `psql`.

## 6. Database Migrations

Integrated `golang-migrate` to manage database schema changes.

Created the initial users-table migration with fields including:

- User ID (UUID).
- Name.
- Email.
- Password hash.
- Currency.
- Creation and update timestamps.

Added a case-insensitive unique email index to prevent duplicate email addresses.

Migration files support applying and reversing schema changes.

## 7. PostgreSQL Connection Pooling

Integrated `pgxpool` for managing PostgreSQL connections.

Configured connection limits and connection lifetimes to support efficient database access.

Verified that the application could connect successfully to the running PostgreSQL instance.

## 8. SQL Code Generation

Installed and configured `sqlc`.

Used SQL query definitions to generate type-safe Go database access code.

This approach keeps SQL explicit while reducing repetitive database-access code.

## 9. HTTP Server and Health Check

Configured the Gin HTTP server.

Implemented the health-check endpoint:

| Method | Endpoint | Description |
|---|---|---|
| GET | `/health` | Check API health |

The health-check endpoint provides a basic way to verify that the API server is running.

## 10. Challenges Encountered

### PostgreSQL Port Conflict

**Problem:** The application could not connect reliably to PostgreSQL using the original host port.

**Solution:** Changed the Docker host port mapping to `5433:5432` and updated the local database connection string.

### Dependency Installation Issues

**Problem:** Installing `sqlc` initially failed because of a network or Go module proxy resolution issue.

**Solution:** Verified the Go proxy configuration and connectivity, then successfully installed `sqlc`.

### Database Connection Errors

**Problem:** Encountered connection-refused and authentication errors while configuring the database.

**Solution:** Verified the running PostgreSQL container, connection port, database credentials, and environment configuration.

## 11. What I Learned

During this milestone, I gained practical experience with:

- Initializing and organizing Go backend projects.
- Managing Go dependencies.
- Configuring environment variables.
- Running PostgreSQL with Docker Compose.
- Understanding Docker host and container port mappings.
- Creating and applying database migrations.
- Using PostgreSQL connection pools.
- Generating Go database code with `sqlc`.
- Structuring backend applications for maintainability.
- Debugging database connectivity issues.

## 12. Milestone Outcome

Successfully established the foundation for the SpendWise backend.

The application has a working Go HTTP server, PostgreSQL database, migration system, environment configuration, and database access layer.

This foundation supports the authentication and financial management features implemented in subsequent milestones.