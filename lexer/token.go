package lexer

import (
	"github.com/ava12/llx/source"
)

type Token struct {
	tokenType int
	typeName  string
	text      string
	source    *source.Source
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
	if t.source == nil {
		return ""
	} else {
		return t.source.Name()
	}
}

func (t *Token) Line () int {
	return t.line
}

func (t *Token) Col () int {
	return t.col
}

type SourcePos interface {
	Source () *source.Source
	Line () int
	Col () int
}

func NewToken (tokenType int, typeName, text string, sp SourcePos) *Token {
	if sp == nil {
		return &Token{tokenType, typeName, text, nil, 0, 0}
	} else {
		return &Token{tokenType, typeName, text, sp.Source(), sp.Line(), sp.Col()}
	}
}

const (
	EofTokenType = -2
	EofTokenName = "-eof-"
)

func EofToken (s *source.Source) *Token {
	line := 0
	col := 0
	if s != nil {
		line, col = s.LineCol(s.Len())
	}
	return &Token{tokenType: EofTokenType, typeName: EofTokenName, source: s, line: line, col: col}
}
