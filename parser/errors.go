package parser

import (
	"github.com/ava12/llx"
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
	IncludeUnresolvedError
)

func unexpectedEofError (t *lexer.Token, expected string) *llx.Error {
	return llx.FormatErrorPos(t, UnexpectedEofError, "unexpected end of file, expecting %s", expected)
}

func unexpectedTokenError (t *lexer.Token, expected string) *llx.Error {
	text := t.Text()
	if len(text) > 10 {
		text = text[: 7] + "..."
	}
	return llx.FormatErrorPos(t, UnexpectedTokenError, "unexpected token %s (%q), expecting %s", t.TypeName(), text, expected)
}

func unexpectedGroupError (t *lexer.Token, group int) *llx.Error {
	return llx.FormatErrorPos(t, UnexpectedGroupError, "expecting token group %d, got %s token", group, t.TypeName())
}

func emitWrongTokenError (t *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(t, EmitWrongTokenError, "cannot emit %s token (type %d)", t.TypeName(), t.Type())
}

func unknownTokenTypeError (typeName string) *llx.Error {
	return llx.FormatError(UnknownTokenTypeError, "unknown token type key: %q", typeName)
}

func unknownTokenLiteralError (text string) *llx.Error {
	return llx.FormatError(UnknownTokenLiteralError, "unknown literal token key: %q", text)
}

func unknownNonTermError (name string) *llx.Error {
	return llx.FormatError(UnknownNonTermError, "unknown non-terminal key: %q", name)
}

func includeUnresolvedError (ntName, sourceName string) *llx.Error {
	return llx.FormatError(IncludeUnresolvedError, "cannot include %q source: resolving ambiguity for %q non-terminal", sourceName, ntName)
}
