package lexer

import (
	"fmt"
	"unicode/utf8"

	"github.com/ava12/llx"
	"github.com/ava12/llx/source"
)

const (
	ErrorTokenType = -3
	ErrorTokenName = "-error-"
)

const (
	ErrNoSource = iota + 101
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
	eof   bool
}

func New (re Regexp, types []TokenType, queue *source.Queue) *Lexer {
	return &Lexer{types: types, re: re, queue: queue}
}

func (l *Lexer) Eof () bool {
	return l.queue.IsEmpty()
}

func (l *Lexer) Source () *source.Source {
	return l.queue.Source()
}

func (l *Lexer) Advance (size int) {
	l.queue.Skip(size)
}

func noSourceError () *llx.Error {
	return llx.NewError(ErrNoSource, "no source code", "", 0, 0)
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
			return nil, noSourceError()
		} else {
			return EofToken(src), nil
		}
	}

	return l.matchToken(src, content, pos)
}

func (l *Lexer) Next () (*Token, error) {
	for {
		t, e := l.fetch()
		if t != nil || e != nil {
			if t != nil && t.Type() == EofTokenType {
				if l.eof {
					t = nil
				} else {
					l.eof = true
				}
			}
			return t, e
		}
	}
}

func (l *Lexer) Shrink (tok *Token) (*Token, error) {
	if tok == nil || len(tok.text) <= 1 {
		return nil ,nil
	}

	if l.queue.Source() != tok.source {
		l.queue.Prepend(tok.source)
	}
	l.queue.Seek(tok.source.Pos(tok.line, tok.col))

	content, pos := l.queue.ContentPos()
	content = content[: pos + len(tok.Text()) - 1]
	return l.matchToken(l.queue.Source(), content, pos)
}
