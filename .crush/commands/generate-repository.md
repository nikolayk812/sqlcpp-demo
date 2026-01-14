Generate repository file and integration tests file for domain model: $ARGUMENTS

Context = agentic tool context. Content = full content of a file.

You can use repository skill.

Follow these steps:

- Run `sqlc generate` command to generate the SQLC code in `internal/db/` directory. 
- Add content of files `internal/db/"$ARGUMENTS".sql.go` and `internal/db/db.go` with generated SQLC files to the context. 
- Add content of file `domain/"$ARGUMENTS".go` with domain model to the context, also resolve referred files and add them to the context. 
- Add content of file `internal/port/"$ARGUMENTS"_port.go` with port (repository interface) to the context. 
- Use content of file at GitHub as reference `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/order_repository_test.go` for testing principles and style inspiration for the next step.
- Create file `internal/repository/"$ARGUMENTS"_repository_test.go` for generated integration tests.
- Use file content at GitHub as reference `https://github.com/nikolayk812/sqlcpp/blob/main/internal/repository/order_repository.go` for implementation principles and style inspiration for the next step.
- Create file `internal/repository/"$ARGUMENTS"_repository.go` for generated implementation code, it has to satisfy the port interface defined in `internal/port/"$ARGUMENTS".go`. 
- Add only direct dependencies from `https://github.com/nikolayk812/sqlcpp/blob/main/go.mod` to local `go.mod` file, if not yet present, run `go mod tidy`.
- Make sure all generated files compile, run `go build ./...` to check, if not then fix compilation errors.
- Run `goimports -w ./internal/repository/*.go` to organize imports in generated files the same way as GoLand.
- Run `make test` to check if all tests pass, if not fix errors.