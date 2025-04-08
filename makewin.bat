@if "%1" == "cmd" goto cmd
@if "%1" == "generate" goto generate
@if "%1" == "test" goto test

@echo valid targets are  cmd, generate, test
@goto end

:cmd
go build -o bin/llxgen.exe ./cmd/llxgen/llxgen.go
@goto end

:generate
go generate ./examples/calc/internal
go generate ./examples/conf-edit/internal
go generate ./examples/style-check/internal
@goto end

:test
go test . ./internal/ints ./internal/queue  ./internal/bmap ./source ./lexer ./langdef ./parser/... ./tree
go test ./examples/calc/internal ./examples/conf-edit/internal ./examples/style-check/internal

:end
