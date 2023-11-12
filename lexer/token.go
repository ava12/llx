package lexer

import (
	"github.com/ava12/llx/source"
)

// Token represents a lexeme, either fetched from a source file or "external" one.
// Contains token type, text, and source and starting position (if known).
// Immutable.
type Token struct {
	tokenType int
	typeName  string
	content   []byte
	text      string
	pos       source.Pos
}

// Type returns token type.
func (t *Token) Type() int {
	return t.tokenType
}

// TypeName returns token type name.
func (t *Token) TypeName() string {
	return t.typeName
}

// Content returns token content.
func (t *Token) Content() []byte {
	return t.content
}

// Text returns lexeme body converted to string.
// Conversion occurs on first call, resulting string is stored and reused to minimize number of allocations.
func (t *Token) Text() string {
	if t.text == "" && len(t.content) > 0 {
		t.text = string(t.content)
	}
	return t.text
}

// Pos returns captured source position.
func (t *Token) Pos() source.Pos {
	return t.pos
}

// Source returns captured source. Returns nil if source is not known.
func (t *Token) Source() *source.Source {
	return t.pos.Source()
}

// SourceName returns source file name. Returns empty string if source is not known.
func (t *Token) SourceName() string {
	return t.pos.SourceName()
}

// Line returns 1-based line number of the first byte of the token.
// Returns 0 if source is not known.
func (t *Token) Line() int {
	return t.pos.Line()
}

// Col returns 1-based column number of the first byte of the token.
// Returns 0 if source is not known.
func (t *Token) Col() int {
	return t.pos.Col()
}

// NewToken creates a token.
// Expects zero value for sp if token source is not known.
func NewToken(tokenType int, typeName string, content []byte, sp source.Pos) *Token {
	return &Token{
		tokenType: tokenType,
		typeName:  typeName,
		content:   content,
		pos:       sp,
	}
}

const (
	// EofTokenType is a fake token indicating the end of source file.
	// Line and column (if present) mark the position right after the last rune of source file.
	EofTokenType = -2

	// EofTokenName is the type name for EofTokenType
	EofTokenName = "-end-of-file-"

	// EoiTokenType is a fake token indicating absence of queued sources (i.e. all sources are processed).
	EoiTokenType = -3

	// EoiTokenName is the type name for EoiTokenType
	EoiTokenName = "-end-of-input-"

	LowestTokenType = -3
)

// EofToken creates a token of EofTokenType.
// s may be nil.
func EofToken(s *source.Source) *Token {
	return &Token{tokenType: EofTokenType, typeName: EofTokenName, pos: source.NewPos(s, s.Len())}
}

// EoiToken creates a token of EoiTokenType.
func EoiToken() *Token {
	return &Token{tokenType: EoiTokenType, typeName: EoiTokenName}
}
