package lexer

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/ava12/llx"
	"github.com/ava12/llx/source"
)

const (
	ErrorTokenType = LowestTokenType - 1
	ErrorTokenName = "-error-"
)

const (
	_ = iota + 101
	ErrWrongChar
	ErrBadToken
)


type TokenType struct {
	Type     int
	TypeName string
}

type Lexer struct {
	types []TokenType
	re    *regexp.Regexp
}

func New (re *regexp.Regexp, types []TokenType) *Lexer {
	return &Lexer{types: types, re: re}
}


func wrongCharError (s *source.Source, content []byte, line, col int) *llx.Error {
	r, _ := utf8.DecodeRune(content)
	msg := fmt.Sprintf("wrong char \"%c\" (u+%x)", r, r)
	return llx.NewError(ErrWrongChar, msg, s.Name(), line, col)
}

func wrongTokenError (t *Token) *llx.Error {
	return llx.FormatErrorPos(t, ErrBadToken, "bad token %q", t.Text())
}

func (l *Lexer) matchToken (src *source.Source, content []byte, pos int) (*Token, int, error) {
	content = content[pos :]
	match := l.re.FindSubmatchIndex(content)
	if len(match) == 0 || match[0] != 0 || match[1] <= match[0] {
		line, col := src.LineCol(pos)
		return nil, 0, wrongCharError(src, content, line, col)
	}

	for i := 2; i < len(match); i += 2 {
		if match[i] >= 0 && match[i + 1] >= 0 {
			line, col := src.LineCol(pos + match[i])
			tokenType := ErrorTokenType
			typeName := ErrorTokenName
			if len(l.types) >= (i >> 1) {
				tokenType = l.types[(i >> 1) - 1].Type
				typeName = l.types[(i >> 1) - 1].TypeName
			}
			token := &Token{
				tokenType,
				typeName,
				string(content[match[i] : match[i + 1]]),
				src,
				line,
				col,
			}
			if tokenType == ErrorTokenType {
				return nil, 0, wrongTokenError(token)
			}

			return token, match[1], nil
		}
	}

	return nil, match[1], nil
}

func (l *Lexer) fetch (q *source.Queue) (*Token, error) {
	content, pos := q.ContentPos()
	src := q.Source()
	if len(content) - pos <= 0 {
		if src == nil {
			return EoiToken(), nil
		} else {
			q.NextSource()
			return EofToken(src), nil
		}
	}

	tok, advance, e := l.matchToken(src, content, pos)
	q.Skip(advance)
	return tok, e
}

func (l *Lexer) Next (q *source.Queue) (*Token, error) {
	for {
		t, e := l.fetch(q)
		if t != nil || e != nil {
			return t, e
		}
	}
}

func (l *Lexer) Shrink (q *source.Queue, tok *Token) *Token {
	if tok == nil || len(tok.text) <= 1 {
		return nil
	}

	src := q.Source()
	if src == nil || src != tok.source {
		return nil
	}

	currentPos := q.Pos()
	q.Seek(tok.source.Pos(tok.line, tok.col))
	content, pos := q.ContentPos()
	content = content[: pos + len(tok.Text()) - 1]
	result, advance, _ := l.matchToken(q.Source(), content, pos)
	if result == nil {
		q.Seek(currentPos)
	} else {
		q.Skip(advance)
	}
	return result
}
