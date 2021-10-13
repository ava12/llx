@if "%1" == "generate" goto generate
@if "%1" == "test" goto test

go build -o bin/llxgen.exe ./llxgen/llxgen.go
@goto end

:generate
go generate ./examples/calc/internal
go generate ./examples/conf-edit/internal
go generate ./examples/style-check/internal
@goto end

:test
go test ./internal/ints ./source ./lexer ./langdef ./parser ./tree
go test ./examples/calc/internal ./examples/conf-edit/internal ./examples/style-check/internal

:end
