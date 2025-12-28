package parser

import (
	"github.com/ava12/llx"
	"github.com/ava12/llx/lexer"
)

// Syntax error codes used by parser:
const (
	// expecting token, got end of input
	UnexpectedEoiError = llx.SyntaxErrors + iota
	// cannot match grammar rule for incoming token
	UnexpectedTokenError
)

// Other error codes used by parser:
const (
	// trying to emit token of unknown type, a literal, or an error token
	EmitWrongTokenError = llx.ParserErrors + iota
	// token hook for unknown token type name
	UnknownTokenTypeError
	// node hook for unknown node
	UnknownNodeError
	// trying to register layer template again
	LayerRegisteredError
	// trying to use unknown layer template
	UnknownLayerError
	// non-side tokens left in source file(s) after parsing is done
	RemainingSourceError
)

func unexpectedEofError(t *lexer.Token, expected string) *llx.Error {
	return llx.FormatErrorPos(t, UnexpectedEoiError, "unexpected end of input, expecting %s", expected)
}

func unexpectedTokenError(t *lexer.Token, expected string) *llx.Error {
	text := t.Text()
	if len(text) > 10 {
		text = text[:7] + "..."
	}
	return llx.FormatErrorPos(t, UnexpectedTokenError, "unexpected %q token (%q), expecting %s", t.TypeName(), text, expected)
}

func emitWrongTokenError(t *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(t, EmitWrongTokenError, "cannot emit %q token (type %d)", t.TypeName(), t.Type())
}

func unknownTokenTypeError(typeName string) *llx.Error {
	return llx.FormatError(UnknownTokenTypeError, "unknown token type key: %q", typeName)
}

func unknownNodeError(name string) *llx.Error {
	return llx.FormatError(UnknownNodeError, "unknown node key: %q", name)
}

func layerRegisteredError(name string) *llx.Error {
	return llx.FormatError(LayerRegisteredError, "layer %q already registered", name)
}

func unknownLayerError(name string) *llx.Error {
	return llx.FormatError(UnknownLayerError, "layer %q is not registered", name)
}

func remainingSourceError(t *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(t, RemainingSourceError, "got %q token after end of source", t.TypeName())
}
