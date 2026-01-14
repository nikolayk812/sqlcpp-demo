# AGENTS.md

This file provides comprehensive guidance for AI agents working with this Go-based SQLC demo application.

## Current State

The repository currently contains:
- Domain models in `internal/domain/` (business entities with value objects)
- Port interfaces in `internal/port/` (repository interface definitions)
- SQL queries in `internal/db/queries/` for SQLC code generation
- Database schema migrations in `internal/migrations/`
- Complete SQLC configuration with type mappings
- Repository skill for guided implementation in `.crush/skills/repository/`
- **Generated repository files were deleted** - use `/generate-repository <domain>` to generate them

The `internal/repository/` and `internal/db/` directories are currently empty of Go files after cleanup.

**Key Technologies:**
- Go 1.25
- SQLC for database code generation
- PostgreSQL with pgx/v5 driver
- Testcontainers for integration testing
- Domain-driven design with hexagonal architecture
- Crush AI Assistant with repository skill for guided code generation

## Current Workflow

After cleanup, the recommended workflow is:

1. **Regenerate Repository Code**: Run `/generate-repository <domain>` to create repository files for any domain model
2. **Verify Generation**: Check that files are created in `internal/db/` and `internal/repository/`
3. **Build and Test**: Ensure `go build ./...` succeeds and `make test` passes
4. **Clean Dependencies**: Run `go mod tidy` to remove unused dependencies

## Essential Commands

### Code Generation
```bash
sqlc generate          # Generate Go code from SQL queries
make sqlc              # Same as above (via Makefile)
```

### Building and Testing
```bash
go build .             # Build the whole project
go build ./...         # Build all packages
make build             # Build via Makefile
make test              # Run all tests with race detection and coverage
go mod tidy            # Clean up dependencies
```

### GitHub Integration
```bash
gh api -H "Accept: application/vnd.github.raw" repos/nikolayk812/sqlcpp/contents/path/to/file.go
# Fetch reference files from the main sqlcpp repository
```

### Cleanup
```bash
./cleanup.sh           # Delete all generated Go files in internal/db and internal/repository
```

## Architecture and Code Organization

### Directory Structure
```
.crush/
└── skills/
    └── repository/        # Repository implementation skill and guidance
internal/
├── domain/            # Business models/entities (Cart, Money)
├── port/              # Repository interfaces (hexagonal architecture ports)
├── repository/        # Repository implementations  
├── db/                # SQLC-generated queries and models
│   └── queries/       # SQL query files for SQLC
└── migrations/        # Database schema migration files
```

### Core Components

**Domain Layer (`internal/domain/`)**
- Contains pure business logic and entities
- No dependencies on persistence or external concerns
- Uses value objects with proper type safety (e.g., Money with decimal.Decimal and currency.Unit)

**Port Layer (`internal/port/`)**  
- Defines repository interfaces
- Follows hexagonal architecture port concept
- Interface naming: `<Domain>Repository` pattern

**Database Layer (`internal/db/`)**
- Contains SQLC-generated code
- SQL queries live in `internal/db/queries/`
- Generated models are called "records" (different from domain models)
- Never edit generated files directly

**Repository Layer (`internal/repository/`)**
- Implements port interfaces
- Maps between SQLC records and domain models
- Handles database transactions when multiple queries needed
- Uses dependency injection pattern with constructors

**Migration Layer (`internal/migrations/`)**
- SQL schema migration files
- Named with incrementing numbers: `01_<table_name>.up.sql`
- Used by SQLC to understand database schema
- Applied automatically in test setup

**Skill Layer (`.crush/skills/repository/`)**
- Contains repository implementation guidance and patterns
- Provides structured approach to implementing repository layer
- Includes mapping conventions, transaction handling, and error patterns

## Development Patterns

### Repository Implementation Rules

1. **Constructor Pattern**: Create constructors for both `*pgxpool.Pool` and `pgx.Tx`
2. **Domain Model Mapping**: Repository methods must accept/return only domain models, never SQLC records
3. **Transaction Handling**: 
   - Single SQLC query: avoid `withTx` method
   - Multiple SQLC queries: use proper `withTxXYZ` method
4. **Struct Initialization**: Initialize structs with 4+ fields before calling methods
5. **Reference Implementation**: Follow patterns in `internal/repository/order_repository.go` from main sqlcpp repo

### Code Generation Workflow

When generating repository code:
1. Run `sqlc generate` first
2. Add generated SQLC files to context
3. Reference domain models and port interfaces
4. Use GitHub reference files for implementation patterns
5. Create both repository and test files
6. Ensure compilation with `go build ./...`
7. Verify tests pass with `make test`

### Testing Strategy

- **Integration Tests**: Use Testcontainers with real PostgreSQL
- **Test Database**: Automatically configured with schema migrations
- **Test Naming**: `<domain>_repository_test.go`
- **Reference**: Follow patterns from `order_repository_test.go` in main repo

## SQLC Configuration

### Type Mappings (sqlc.yaml)
```yaml
overrides:
  - db_type: "uuid" → github.com/google/uuid.UUID
  - db_type: "numeric" → github.com/shopspring/decimal.Decimal  
  - db_type: "timestamp" → time.Time
```

### SQL Query Patterns
- Use `-- name: QueryName :returntype` format
- `:many` for SELECT returning multiple rows
- `:exec` for INSERT/UPDATE/DELETE
- `:execrows` for DELETE with row count return
- Use `ON CONFLICT` for upsert operations

## Dependencies and Environment

### Key Dependencies
```
github.com/jackc/pgx/v5              # PostgreSQL driver
github.com/shopspring/decimal        # Decimal arithmetic
github.com/google/uuid               # UUID handling
github.com/testcontainers/testcontainers-go/modules/postgres  # Testing
github.com/stretchr/testify          # Test assertions
github.com/brianvoe/gofakeit/v7      # Test data generation
golang.org/x/text                   # Currency handling
```

### Environment Requirements
- PostgreSQL database (handled via testcontainers in tests)
- Docker (required for testcontainers)
- SQLC CLI tool (installed via `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)

### Test Environment
- Special Docker socket override: `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock`
- Race detection enabled in tests
- Coverage reporting enabled

## Coding Conventions

### Naming Conventions
- Domain models: PascalCase (e.g., `Order`, `Product`)
- Repository interfaces: `<Domain>Repository`
- Repository implementations: `<domain>Repository` 
- Test files: `<domain>_repository_test.go`
- SQL files: lowercase with underscores (e.g., `order.sql`)

### Code Style
- Use Go standard formatting (gofmt)
- Import ordering: standard library, third party, local
- Error handling: always handle errors, use descriptive messages
- Context: always accept `context.Context` as first parameter in repository methods

### Database Conventions
- Table names: snake_case (e.g., `order_items`)
- Column names: snake_case (e.g., `owner_id`, `product_id`)  
- Primary keys: composite where logical (e.g., `owner_id, product_id`)
- Indexes: descriptive names (e.g., `idx_order_items_owner`)
- Timestamps: use `TIMESTAMP DEFAULT CURRENT_TIMESTAMP`

## Special Commands and Workflows

### Repository Generation Command
Use Claude Code's custom command:
```
/generate-repository <domain>
```

This command (defined in `.claude/commands/generate-repository.md`):
1. Runs `sqlc generate` command to generate SQLC code
2. Adds generated SQLC files to context
3. Adds domain model and port interface files to context  
4. Creates or modifies `internal/repository/repository_test.go` for testcontainer setup
5. Creates `internal/repository/{domain}_repository_test.go` for integration tests
6. Creates `internal/repository/{domain}_repository.go` for implementation
7. Organizes imports with `goimports`
8. Ensures compilation with `go build ./...`
9. Verifies tests pass with `make test`
10. References implementation patterns from main sqlcpp repo

### Skill-Based Implementation
The repository skill (`.crush/skills/repository/`) provides:
- Core principles for repository implementation
- Constructor patterns and method structures
- Transaction handling guidance
- Error handling patterns
- Mapping conventions between SQLC records and domain models

### GitHub Reference Files
The project references implementation patterns from:
- `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/repository_test.go`
- `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/order_repository.go`
- `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/order_repository_test.go`

### Repository Skill
Access the repository skill for guided implementation:
- File: `.crush/skills/repository/SKILL.md`
- Provides step-by-step guidance for implementing repositories
- Includes mapping conventions and error handling patterns
- Follows hexagonal architecture principles

## Common Gotchas

### SQLC Generation
- Always run `sqlc generate` after changing SQL files or migrations
- Generated files should never be edited manually
- SQLC reads migration files to understand schema

### Repository Implementation  
- Don't mix domain models with SQLC records
- Always map between record and domain model types
- Use transactions only when calling multiple SQLC queries
- Repository constructors must support both pool and transaction

### Testing
- Tests require Docker for testcontainers
- Database schema applied automatically from migrations
- Integration tests run against real PostgreSQL instances
- Use `make test` not `go test` to get proper Docker socket configuration

### Dependencies
- Only add dependencies that exist in main sqlcpp repo
- Run `go mod tidy` after adding dependencies
- Check main repo's go.mod for version compatibility

## CI/CD Integration

GitHub Actions workflow (`.github/workflows/test.yml`):
- Runs on Ubuntu with Go 1.25
- Installs SQLC CLI tool
- Generates code before testing
- Runs full test suite with proper environment

The workflow ensures:
1. Code generation works
2. All tests pass
3. Build succeeds
4. Dependencies are correctly managed