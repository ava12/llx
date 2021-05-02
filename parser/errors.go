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
	UnknownTokenTypeError
	UnknownTokenLiteralError
	UnknownNonTermError
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

func unknownTokenTypeError (typeName string) *err.Error {
	return err.Format(UnknownTokenTypeError, "unknown token type key: %q", typeName)
}

func unknownTokenLiteralError (text string) *err.Error {
	return err.Format(UnknownTokenLiteralError, "unknown literal token key: %q", text)
}

func unknownNonTermError (name string) *err.Error {
	return err.Format(UnknownNonTermError, "unknown non-terminal key: %q", name)
}
