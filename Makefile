bin/llxgen : llxgen/llxgen.go llxgen/version
	go build -o bin/llxgen ./llxgen/llxgen.go

.PHONY: generate test

generate:
	go generate ./examples/calc/internal
	go generate ./examples/conf-edit/internal
	go generate ./examples/style-check/internal

test:
	go test ./internal/ints ./source ./lexer ./langdef ./parser ./tree
	go test ./examples/calc/internal ./examples/conf-edit/internal ./examples/style-check/internal
