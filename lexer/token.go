package lexer

import (
	"github.com/ava12/llx/source"
)

type Token struct {
	tokenType int
	typeName  string
	text      string
	pos       source.Pos
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

func (t *Token) Pos () source.Pos {
	return t.pos
}

func (t *Token) Source () *source.Source {
	return t.pos.Source()
}

func (t *Token) SourceName () string {
	return t.pos.SourceName()
}

func (t *Token) Line () int {
	return t.pos.Line()
}

func (t *Token) Col () int {
	return t.pos.Col()
}

func NewToken (tokenType int, typeName, text string, sp source.Pos) *Token {
	return &Token{tokenType, typeName, text, sp}
}

const (
	EofTokenType = -2
	EoiTokenType = -3
	LowestTokenType = -3
	EofTokenName = "-end-of-file-"
	EoiTokenName = "-end-of-input-"
)

func EofToken (s *source.Source) *Token {
	return &Token{tokenType: EofTokenType, typeName: EofTokenName, pos: source.NewPos(s, s.Len())}
}

func EoiToken () *Token {
	return &Token{tokenType: EoiTokenType, typeName: EoiTokenName}
}
