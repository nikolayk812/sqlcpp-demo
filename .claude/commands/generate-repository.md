Generate repository file and integration tests file for domain model: $ARGUMENTS.

Follow these steps:

- Run `sqlc generate` command to generate the SQLC code in `internal/db/` directory. 
- Add the full content of files `internal/db/$ARGUMENTS.sql.go` and `internal/db/db.go` with generated SQLC files to the agentic tool context. 
- Add file `domain/$ARGUMENTS.go` with domain model to context, also resolve referred files and add them to the context. 
- Add file `internal/port/$ARGUMENTS_port.go` with port interface to the context. 
- Use file content at GitHub as reference `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/repository_test.go` for the next step.
- Create file or modify `internal/repository/repository_test.go`, use paths to SQL schema migrations in `internal/migrations` dir to configure Postgres inside Testcontainers.
- Use file content at GitHub as reference `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/order_repository.go` for implementation principles and style inspiration for the next step.
- Create file `internal/repository/$ARGUMENTS_repository.go` for generated implementation code, it has to satisfy the port interface defined in `internal/port/$ARGUMENTS.go`. 
- Add only direct dependencies from `https://github.com/nikolayk812/sqlcpp/blob/main/go.mod` to local `go.mod` file, if not already present, run `go mod tidy`.
- Make sure all generated files compile, run `go build ./...` to check, if not then fix compilation errors.
- Run `goimports -w ./internal/repository/*.go` to organize imports in generated files the same way as GoLand.
- Run `make test` to check if all tests pass, if not fix errors.