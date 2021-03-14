package parser

import (
	err "github.com/ava12/llx/errors"
	"github.com/ava12/llx/lexer"
)

const (
	ErrUnexpectedEof = iota + 201
	ErrUnexpectedToken
)

func unexpectedEofError (t *lexer.Token, expected string) *err.Error {
	return err.FormatPos(t, ErrUnexpectedEof, "unexpected end of file, expecting %s", expected)
}

func unexpectedTokenError (t *lexer.Token, expected string) *err.Error {
	return err.FormatPos(t, ErrUnexpectedToken, "unexpected token $%s, expecting %s", t.TypeName(), expected)
}
