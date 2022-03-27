package lexer

import (
	"fmt"
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


type Regexp interface {
	FindSubmatchIndex (content []byte) []int
}

type TokenType struct {
	Type     int
	TypeName string
}

type Lexer struct {
	types []TokenType
	re    Regexp
	queue *source.Queue
}

func New (re Regexp, types []TokenType, queue *source.Queue) *Lexer {
	return &Lexer{types: types, re: re, queue: queue}
}

func (l *Lexer) Source () *source.Source {
	return l.queue.Source()
}

func (l *Lexer) Advance (size int) {
	l.queue.Skip(size)
}

func wrongCharError (s *source.Source, content []byte, line, col int) *llx.Error {
	r, _ := utf8.DecodeRune(content)
	msg := fmt.Sprintf("wrong char \"%c\" (u+%x)", r, r)
	return llx.NewError(ErrWrongChar, msg, s.Name(), line, col)
}

func wrongTokenError (t *Token) *llx.Error {
	return llx.FormatErrorPos(t, ErrBadToken, "bad token %q", t.Text())
}

func (l *Lexer) matchToken (src *source.Source, content []byte, pos int) (*Token, error) {
	content = content[pos :]
	match := l.re.FindSubmatchIndex(content)
	if len(match) == 0 || match[0] != 0 || match[1] <= match[0] {
		line, col := l.queue.LineCol(pos)
		return nil, wrongCharError(src, content, line, col)
	}

	for i := 2; i < len(match); i += 2 {
		if match[i] >= 0 && match[i + 1] >= 0 {
			line, col := l.queue.LineCol(pos + match[i])
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
				return nil, wrongTokenError(token)
			}

			l.Advance(match[1])
			return token, nil
		}
	}

	l.Advance(match[1])
	return nil, nil
}

func (l *Lexer) fetch () (*Token, error) {
	content, pos := l.queue.ContentPos()
	src := l.queue.Source()
	if len(content) - pos <= 0 {
		if src == nil {
			return EoiToken(), nil
		} else {
			l.queue.NextSource()
			return EofToken(src), nil
		}
	}

	return l.matchToken(src, content, pos)
}

func (l *Lexer) Next () (*Token, error) {
	for {
		t, e := l.fetch()
		if t != nil || e != nil {
			return t, e
		}
	}
}

func (l *Lexer) Shrink (tok *Token) *Token {
	if tok == nil || len(tok.text) <= 1 {
		return nil
	}

	src := l.queue.Source()
	if src == nil || src != tok.source {
		return nil
	}

	currentPos := l.queue.Pos()
	l.queue.Seek(tok.source.Pos(tok.line, tok.col))
	content, pos := l.queue.ContentPos()
	content = content[: pos + len(tok.Text()) - 1]
	result, _ := l.matchToken(l.queue.Source(), content, pos)
	if result == nil {
		l.queue.Seek(currentPos)
	}
	return result
}
