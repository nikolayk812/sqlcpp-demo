Generate repository file and integration tests file for domain model: $ARGUMENTS.

Follow these steps:

- Run `sqlc generate` command to generate the SQLC code in `internal/db/` directory. 
- Add files `internal/db/$ARGUMENTS.sql.go` and `internal/db/db.go` with generated SQLC files to the context. 
- Add file `domain/$ARGUMENTS.go` with domain model to context, also resolve referred files and add them to the context. 
- Add file `internal/port/$ARGUMENTS.go` with port interface to the context. 
- Use file content at GitHub as reference `repos/nikolayk812/sqlcpp/contents/internal/repository/repository_test.go` for the next step.
- Create file or modify `internal/repository/repository_test.go`, use paths to SQL schema migrations in `internal/migrations` dir to configure Postgres inside Testcontainers.
- Use file content at GitHub as reference `repos/nikolayk812/sqlcpp/contents/internal/repository/order_repository_test.go` for testing principles and style inspiration for the next step.
- Create file `internal/repository/$ARGUMENTS_repository_test.go` for generated integration tests.
- Use file content at GitHub as reference `repos/nikolayk812/sqlcpp/contents/internal/repository/order_repository` for testing principles and style inspiration for the next step.
- Create file `internal/repository/$ARGUMENTS_repository.go` for generated implementation code, it has to satisfy the port interface defined in `internal/port/$ARGUMENTS.go`. 
- Add only direct dependencies from `repos/nikolayk812/sqlcpp/contents/go.mod` to `go.mod` file, if not already present, run `go mod tidy`.
- Make sure all generated files compile, run `go build ./...` to check, if not fix compilation errors.
- Run `go test ./...` to check if all tests pass, if not fix errors.