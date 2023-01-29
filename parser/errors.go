package parser

import (
	"github.com/ava12/llx"
	"github.com/ava12/llx/lexer"
)

// Error codes used by parser:
const (
	// expecting token, got end of input
	UnexpectedEoiError = iota + 201

	// cannot match grammar rule for incoming token
	UnexpectedTokenError

	// trying to emit token of unknown type, a literal, or an error token
	EmitWrongTokenError

	// token hook for unknown token type name
	UnknownTokenTypeError

	// literal hook for unknown literal
	UnknownTokenLiteralError

	// non-terminal hook for unknown non-terminal
	UnknownNonTermError
)

func unexpectedEofError (t *lexer.Token, expected string) *llx.Error {
	return llx.FormatErrorPos(t, UnexpectedEoiError, "unexpected end of input, expecting %s", expected)
}

func unexpectedTokenError (t *lexer.Token, expected string) *llx.Error {
	text := t.Text()
	if len(text) > 10 {
		text = text[: 7] + "..."
	}
	return llx.FormatErrorPos(t, UnexpectedTokenError, "unexpected %q token (%q), expecting %s", t.TypeName(), text, expected)
}

func emitWrongTokenError (t *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(t, EmitWrongTokenError, "cannot emit %q token (type %d)", t.TypeName(), t.Type())
}

func unknownTokenTypeError (typeName string) *llx.Error {
	return llx.FormatError(UnknownTokenTypeError, "unknown token type key: %q", typeName)
}

func unknownTokenLiteralError (text string) *llx.Error {
	return llx.FormatError(UnknownTokenLiteralError, "unknown literal key: %q", text)
}

func unknownNonTermError (name string) *llx.Error {
	return llx.FormatError(UnknownNonTermError, "unknown non-terminal key: %q", name)
}
