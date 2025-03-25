/*
Package llx is a general-purpose LL(*) parser library.

Consists of subpackages:
  - cmd/llxgen: console utility converting grammar description to Go source file containing grammar definition structure;
  - grammar: defines structure that contains definition of lexemes and finite state machine used by parser;
  - langdef: converts grammar description (written in EBNF-like language) to grammar definition;
  - lexer: lexical analyzer;
  - parser: defines parser;
  - source: defines source file and source queue used by lexer;
  - tree: types and functions to create, traverse, and modify syntax trees.

Typical usage is:

1. Describe grammar in EBNF-like language. Description does not contain Go code,
the same grammar can be used for different purposes (translators, linters, formatters, etc.).

2. Parse grammar description using either langdef subpackage "on the fly"
or llxgen utility to generate Go file.

3. Define hooks to handle tokens and/or syntax tree nodes emitted by parser.

4. Create new parser for desired grammar and feed it source files and hooks.
*/
package llx

import (
	"fmt"
)

// Error classes used by subpackages, each class contains up to 99 error codes:
const (
	LangDefErrors = 1   // used by langdef
	LexicalErrors = 101 // used by lexer
	SyntaxErrors  = 201 // used by parser
	ParserErrors  = 301 // used by parser
	LayerErrors   = 401 // used by standard hook layers
)

// Error is the error type used by llx subpackages.
type Error struct {
	// Code contains non-zero error code.
	Code int

	// Message contains non-empty error message including source name and position information if provided.
	Message string

	// SourceName contains source name that caused this error or empty string.
	SourceName string

	// Line contains line number in source file or 0.
	Line int

	// Col contains column number in source file or 0.
	Col int
}

// SourcePos is used to retrieve source name and position information when constructing an error;
// source.Pos and lexer.Token implement this interface.
type SourcePos interface {
	// SourceName returns source file name or empty string.
	SourceName() string
	// Line returns line number or 0.
	Line() int
	// Col returns column number or 0.
	Col() int
}

// NewError creates new Error structure.
// name, line, and col will be added to error message if provided (non-zero).
func NewError(code int, msg, name string, line, col int) *Error {
	if name != "" && line != 0 && col != 0 {
		msg += fmt.Sprintf(" in %s at line %d col %d", name, line, col)
	}
	return &Error{code, msg, name, line, col}
}

// Error simply returns Error.Message.
func (e *Error) Error() string {
	return e.Message
}

// FormatError creates Error structure with no source and position information.
// params will be added to error message using fmt.Sprintf function.
func FormatError(code int, msg string, params ...any) *Error {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewError(code, msg, "", 0, 0)
}

// FormatErrorPos creates Error structure with source and position information.
// pos must not be nil.
// params will be added to error message using fmt.Sprintf function.
func FormatErrorPos(pos SourcePos, code int, msg string, params ...any) *Error {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewError(code, msg, pos.SourceName(), pos.Line(), pos.Col())
}
