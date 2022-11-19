.PHONY: help generate test

help:
	@echo valid targets are  cmd, generate, test

cmd : cmd/llxgen/llxgen.go cmd/llxgen/version
	go build -o bin/llxgen ./cmd/llxgen/llxgen.go

generate:
	go generate ./examples/calc/internal
	go generate ./examples/conf-edit/internal
	go generate ./examples/style-check/internal

test:
	go test . ./internal/ints ./internal/queue ./source ./lexer ./langdef ./parser ./tree
	go test ./examples/calc/internal ./examples/conf-edit/internal ./examples/style-check/internal
