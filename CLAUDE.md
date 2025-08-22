# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based demo application that showcases an AI-powered repository code generation on top of SQL queries and
domain models provided by a user and SQLC generated code.

## Key Commands

### Building and Running

- `go build .` - Build the whole project
- `go test ./...` - Run all tests in the project
- `gh api repos/nikolayk812/sqlcpp/contents/path/to/file.go --jq '.content' | base64 -d` - Fetch GitHub file content for https://github.com/nikolayk812/sqlcpp/blob/main/path/to/file.go)

### Code Generation

- `sqlc generate` - Generate Go code from SQL queries using the sqlc.yaml configuration

### Environment Requirements

- PostgreSQL database for testing (handled via testcontainers)

## Architecture

### Core Components

**Domain Models** (`internal/domain/`):

- `Cart`, `Money` domain models/entities separated from persistence concerns

**Database directory** (`internal/db/`):

- Hosts SQL queries for SQLC code generation in `internal/db/queries/`
- SQLC-generated queries and models, aka records which are different from domain models

**Database schema migration directory** (`internal/migrations/`):

- SQL scripts for database schema migrations
- Used by SQLC to generate models and queries

**Port directory** (`internal/port/`):

- Interfaces defining repository methods
- Corresponds to port concept in hexagonal architecture

**Repository Layer** (`internal/repository/`):

- Implements port interfaces defined in `internal/port/`
- Implemented methods delegate to SQLC-generated queries, it can call one or multiple queries
- If multiple queries are called in a single method, they have to be wrapped in a transaction
- Repository methods should accept and return only domain models
- As SQLC-generated queries return records, repository methods should map these records to domain models
- Repository New() function uses dependency injection pattern with constructors for both `*pgxpool.Pool` or `pgx.Tx`

### Testing Strategy

- Integration tests use Testcontainers for PostgreSQL
- Repository tests run against real database instances
- Migration scripts applied automatically in test setup

## Development Patterns

### Repository Implementation

- Follow the patterns established in `internal/repository/order_repository.go`
- Create constructors for both pgx pool and transaction
- Avoid using `withTx` method when delegating to a single SQLC generated query
- Use proper `withTxXYZ` method when calling multiple SQLC generated queries in a single repository method
- Initialize structs with 4 or more fields before calling a method which uses it