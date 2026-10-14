# Milestone 2 — Authentication

## 1. Overview

The second milestone of SpendWise focused on implementing secure user authentication and protecting API endpoints.

The goal was to allow users to create accounts, log in securely, and access their own information using JSON Web Tokens (JWTs).

**Status:** Completed

## 2. Technologies Used

- **Go:** Backend development.
- **Gin:** HTTP routing and request handling.
- **PostgreSQL:** Persistent user storage.
- **pgx and sqlc:** Database access.
- **bcrypt:** Secure password hashing and verification.
- **JWT:** Token-based authentication.
- **Go testing package:** Automated tests.

## 3. User Registration

Implemented the registration endpoint:

**POST `/api/v1/auth/register`**

The registration process:

1. Receives user details from an HTTP request.
2. Validates the registration input.
3. Hashes the password using bcrypt.
4. Stores the user's information in PostgreSQL.
5. Returns the registered user's information without exposing the password hash.

The database enforces case-insensitive email uniqueness, preventing duplicate accounts with the same email address.

## 4. User Login

Implemented the login endpoint:

**POST `/api/v1/auth/login`**

The login process:

1. Receives the user's email and password.
2. Retrieves the user from PostgreSQL.
3. Compares the supplied password against the stored bcrypt hash.
4. Generates a signed JWT when authentication succeeds.
5. Returns the token to the client.

Invalid credentials are rejected.

## 5. Password Security

Used bcrypt to hash user passwords before database storage.

Passwords are never stored as plaintext.

During login, bcrypt compares the submitted password with the stored hash.

Password hashes are excluded from public API responses.

## 6. JWT Authentication

Implemented JWT-based authentication to protect private API endpoints.

JWTs identify the authenticated user and have a configured expiration time.

Clients include the token in the HTTP Authorization header:

`Authorization: Bearer <access-token>`

The server validates the token before allowing access to protected resources.

## 7. Authentication Middleware

Implemented authentication middleware for protected routes.

The middleware:

- Extracts the Bearer token from the request.
- Validates the token.
- Rejects missing, invalid, or expired tokens.
- Makes authenticated user information available to protected handlers.

This avoids duplicating authentication logic across endpoints.

## 8. Authenticated User Profile

Implemented the protected endpoint:

**GET `/api/v1/users/me`**

The endpoint retrieves the currently authenticated user's profile.

A valid JWT is required to access it.

## 9. Error Handling and Validation

Implemented error handling for authentication failures, including:

- Invalid registration input.
- Duplicate email addresses.
- Incorrect login credentials.
- Missing authentication tokens.
- Invalid or expired JWTs.

The API returns appropriate HTTP error responses without exposing password hashes or other sensitive authentication details.

## 10. Automated Testing

Added automated tests for authentication components using Go's built-in testing package.

Verified the authentication tests with:

`go test ./...`

The authentication package tests passed successfully.

Additional integration and end-to-end test coverage can be expanded during Milestone 7 — Production Readiness.

## 11. Docker Integration

Added a multi-stage Docker build for the Go API.

This separates application compilation from the final runtime image, supporting a more streamlined deployment process.

The application can be started using Docker Compose.

## 12. What I Learned

During this milestone, I gained practical experience with:

- Building registration and login endpoints in Go.
- Hashing and verifying passwords using bcrypt.
- Understanding JWT-based authentication.
- Implementing Gin middleware.
- Protecting API routes.
- Using request context to share authenticated user information.
- Handling authentication and validation errors.
- Writing and running automated Go tests.
- Building Go applications with Docker.

## 13. Milestone Outcome

Successfully implemented the SpendWise authentication system.

Users can register, log in, obtain JWT access tokens, and retrieve their authenticated profiles.

The authentication middleware also provides the security foundation for account, category, transaction, and future budget management features.