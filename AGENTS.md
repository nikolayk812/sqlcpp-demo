# AGENTS.md

This file provides comprehensive guidance for AI agents working with this Go-based SQLC demo application.

## Current State

The repository currently contains:
- Domain models for `Cart` and `Money` entities
- Port interface (`CartRepository`) defining repository methods
- SQL queries for cart operations in SQLC format
- Database schema migration for cart_items table
- Complete SQLC configuration with type mappings
- **No generated repository files yet** - use `/generate-repository cart` to create them

The `internal/repository/` and `internal/db/` directories are currently empty of Go files.

**Key Technologies:**
- Go 1.25
- SQLC for database code generation
- PostgreSQL with pgx/v5 driver
- Testcontainers for integration testing
- Domain-driven design with hexagonal architecture

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
- Examples: `Cart`, `Money` structs
- Uses value objects (Money with decimal.Decimal and currency.Unit)

**Port Layer (`internal/port/`)**  
- Defines repository interfaces
- Follows hexagonal architecture port concept
- Interface naming: `<Domain>Repository` (e.g., `CartRepository`)

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
- Named with incrementing numbers: `01_cart_items.up.sql`
- Used by SQLC to understand database schema
- Applied automatically in test setup

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
- Domain models: PascalCase (`Cart`, `CartItem`)
- Repository interfaces: `<Domain>Repository`
- Repository implementations: `<domain>Repository` 
- Test files: `<domain>_repository_test.go`
- SQL files: lowercase with underscores (`cart.sql`)

### Code Style
- Use Go standard formatting (gofmt)
- Import ordering: standard library, third party, local
- Error handling: always handle errors, use descriptive messages
- Context: always accept `context.Context` as first parameter in repository methods

### Database Conventions
- Table names: snake_case (`cart_items`)
- Column names: snake_case (`owner_id`, `product_id`)  
- Primary keys: composite where logical (`owner_id, product_id`)
- Indexes: descriptive names (`idx_cart_items_owner`)
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
7. Ensures compilation with `go build ./...`
8. Verifies tests pass with `make test`
9. References implementation patterns from main sqlcpp repo

### GitHub Reference Files
The project references implementation patterns from:
- `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/repository_test.go`
- `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/order_repository.go`
- `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/order_repository_test.go`

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