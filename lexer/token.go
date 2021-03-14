package lexer

import (
	"github.com/ava12/llx/source"
)

type Token struct {
	tokenType int
	typeName, text string
	source *source.Source
	line, col int
}

func (t *Token) Type () int {
	return t.tokenType
}

func (t *Token) TypeName () string {
	return t.typeName
}

func (t *Token) Text () string {
	return t.text
}

func (t *Token) Source () *source.Source {
	return t.source
}

func (t *Token) SourceName () string {
	return t.source.Name()
}

func (t *Token) Line () int {
	return t.line
}

func (t *Token) Col () int {
	return t.col
}

const (
	EofTokenType = -1
	EofTokenName = "-eof-"
)

func EofToken (s *source.Source) *Token {
	line, col := s.LineCol(s.Len())
	return &Token{tokenType: EofTokenType, typeName: EofTokenName, source: s, line: line, col: col}
}
