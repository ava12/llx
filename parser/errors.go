package parser

import (
	err "github.com/ava12/llx/errors"
	"github.com/ava12/llx/lexer"
)

const (
	UnexpectedEofError = iota + 201
	UnexpectedTokenError
	UnexpectedGroupError
)

func unexpectedEofError (t *lexer.Token, expected string) *err.Error {
	return err.FormatPos(t, UnexpectedEofError, "unexpected end of file, expecting %s", expected)
}

func unexpectedTokenError (t *lexer.Token, expected string) *err.Error {
	return err.FormatPos(t, UnexpectedTokenError, "unexpected token $%s, expecting %s", t.TypeName(), expected)
}

func unexpectedGroupError (t *lexer.Token, group int) *err.Error {
	return err.FormatPos(t, UnexpectedGroupError, "expecting token group %d, got %s token", group, t.TypeName())
}
