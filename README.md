# SQLC++ Demo

A Go project demonstrating SQLC for type-safe database operations with domain-driven design.

> **Note**: This is a demo project inspired by [sqlcpp](https://github.com/nikolayk812/sqlcpp).

## What is this?

This project shows how to build Go applications with:
- SQLC for database code generation
- PostgreSQL with pgx driver
- Domain-driven design patterns
- Repository pattern implementation
- Integration testing with Testcontainers

## Structure

```
internal/
├── domain/      # Business models (Cart, Money)
├── port/        # Repository interfaces  
├── repository/  # Repository implementations
├── db/          # Generated SQLC code
└── migrations/  # Database schema
```

## Key Files

- `internal/domain/cart.go` - Main business entity
- `internal/port/cart_port.go` - Repository interface
- `internal/repository/cart_repository.go` - Database implementation
- `internal/db/queries/cart.sql` - SQL queries for SQLC
- `sqlc.yaml` - SQLC configuration

## Database

Uses PostgreSQL with custom type mappings:
- UUID fields → `github.com/google/uuid`
- Decimal fields → `github.com/shopspring/decimal`
- Timestamps → `time.Time`

## Testing

Integration tests use Testcontainers with real PostgreSQL instances. Tests automatically set up database schema and run migrations.

## Usage

1. Define domain models in `internal/domain/`
2. Add SQL migrations in `internal/migrations/`
3. Write SQL queries in `internal/db/queries/`
4. Use Claude Code's `/generate-repository cart` command to auto-generate repository and tests for the cart domain