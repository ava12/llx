.PHONY: help cmd generate test

help:
	@echo valid targets are  cmd, generate, test

cmd:
	go build -o bin/llxgen ./cmd/llxgen/llxgen.go

generate:
	go generate ./examples/calc/internal
	go generate ./examples/conf-edit/internal
	go generate ./examples/style-check/internal

test:
	go test . ./internal/ints ./internal/queue ./internal/bmap ./source ./lexer ./langdef ./parser/... ./tree
	go test ./examples/calc/internal ./examples/conf-edit/internal ./examples/style-check/internal
