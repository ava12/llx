package parser

import (
	err "github.com/ava12/llx/errors"
	"github.com/ava12/llx/lexer"
)

const (
	UnexpectedEofError = iota + 201
	UnexpectedTokenError
	UnexpectedGroupError
	EmitWrongTokenError
)

func unexpectedEofError (t *lexer.Token, expected string) *err.Error {
	return err.FormatPos(t, UnexpectedEofError, "unexpected end of file, expecting %s", expected)
}

func unexpectedTokenError (t *lexer.Token, expected string) *err.Error {
	text := t.Text()
	if len(text) > 10 {
		text = text[: 7] + "..."
	}
	return err.FormatPos(t, UnexpectedTokenError, "unexpected token %s (%q), expecting %s", t.TypeName(), text, expected)
}

func unexpectedGroupError (t *lexer.Token, group int) *err.Error {
	return err.FormatPos(t, UnexpectedGroupError, "expecting token group %d, got %s token", group, t.TypeName())
}

func emitWrongTokenError (t *lexer.Token) *err.Error {
	return err.FormatPos(t, EmitWrongTokenError, "cannot emit %s token (type %d)", t.TypeName(), t.Type())
}
