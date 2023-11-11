// Package lexer defines lexical analyzer.
package lexer

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/ava12/llx"
	"github.com/ava12/llx/source"
)

const (
	// ErrorTokenType is the type for fake tokens capturing broken lexemes (e.g. incorrect string literals).
	// The purpose of these tokens is to generate more informative error messages.
	// Lexer will never return a token of this type, an error with message containing token text will be returned instead.
	ErrorTokenType = LowestTokenType - 1

	// ErrorTokenName is the type name for ErrorTokenType.
	ErrorTokenName = "-error-"
)

// Error codes used by lexer:
const (
	// WrongCharError indicates that lexer cannot fetch any token at current position.
	// Error message contains the rune at current source position.
	WrongCharError = llx.LexicalErrors + iota

	// BadTokenError indicates that lexer has fetched a token of ErrorTokenType.
	BadTokenError
)

// TokenType describes token type for specific capturing group of regular expression.
type TokenType struct {
	// Type contains token type, may be any value. ErrorTokenType is treated specially.
	Type int

	// TypeName contains token type name, may be any value.
	TypeName string
}

// Lexer performs lexical analysis of current source in source.Queue using regexp.Regexp.
// Lexer itself is immutable, stateless, and safe for concurrent use (i.e. the same Lexer instance
// may be used with different queues by different goroutines), but it affects queue state.
// Each token type that may be returned by lexer maps to its own regexp capturing group index.
// A match containing no captured groups is treated as insignificant lexeme (e.g. whitespace),
// in this case lexer tries to fetch a token again at new position.
// Every byte of source file must belong to some lexeme.
type Lexer struct {
	types []TokenType
	re    *regexp.Regexp
}

// New creates new Lexer.
// Each n-th element of types describes token type for (n+1)-th regexp capturing group.
// A group that has no description is treated as ErrorTokenType.
func New(re *regexp.Regexp, types []TokenType) *Lexer {
	return &Lexer{types: types, re: re}
}

func wrongCharError(s *source.Source, content []byte, line, col int) *llx.Error {
	r, _ := utf8.DecodeRune(content)
	msg := fmt.Sprintf("wrong char \"%c\" (u+%x)", r, r)
	return llx.NewError(WrongCharError, msg, s.Name(), line, col)
}

func wrongTokenError(t *Token) *llx.Error {
	return llx.FormatErrorPos(t, BadTokenError, "bad token %q", t.Text())
}

func (l *Lexer) matchToken(src *source.Source, content []byte, pos int) (*Token, int, error) {
	content = content[pos:]
	match := l.re.FindSubmatchIndex(content)
	if len(match) == 0 || match[0] != 0 || match[1] <= match[0] {
		line, col := src.LineCol(pos)
		return nil, 0, wrongCharError(src, content, line, col)
	}

	for i := 2; i < len(match); i += 2 {
		if match[i] >= 0 && match[i+1] >= 0 {
			sp := source.NewPos(src, pos+match[i])
			tokenType := ErrorTokenType
			typeName := ErrorTokenName
			if len(l.types) >= (i >> 1) {
				tokenType = l.types[(i>>1)-1].Type
				typeName = l.types[(i>>1)-1].TypeName
			}
			token := &Token{
				tokenType,
				typeName,
				string(content[match[i]:match[i+1]]),
				sp,
			}
			if tokenType == ErrorTokenType {
				return nil, 0, wrongTokenError(token)
			}

			return token, match[1], nil
		}
	}

	return nil, match[1], nil
}

func (l *Lexer) fetch(q *source.Queue) (*Token, error) {
	content, pos := q.ContentPos()
	src := q.Source()
	if len(content)-pos <= 0 {
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

// Next fetches token starting at current source position and advances current position.
// Returns nil token and llx.Error and does not make any changes if there is a lexical error.
// Returns EoI token if queue is empty.
// Returns EoF token and discards current source if current position is beyond the end of current source.
func (l *Lexer) Next(q *source.Queue) (*Token, error) {
	for {
		t, e := l.fetch(q)
		if t != nil || e != nil {
			return t, e
		}
	}
}

// Shrink tries to fetch a token which starts at the same position as given and is at least one byte shorter.
// Adjusts current position and returns shrunk token on success.
// Makes no changes and returns nil if given token has no captured source and position information,
// was fetched from source other than current, or a lexical error occurs.
func (l *Lexer) Shrink(q *source.Queue, tok *Token) *Token {
	if tok == nil || len(tok.text) <= 1 {
		return nil
	}

	src := q.Source()
	if src == nil || src != tok.pos.Source() {
		return nil
	}

	currentPos := q.Pos()
	q.Seek(tok.pos.Pos())
	content, pos := q.ContentPos()
	content = content[:pos+len(tok.Text())-1]
	result, advance, _ := l.matchToken(q.Source(), content, pos)
	if result == nil {
		q.Seek(currentPos)
	} else {
		q.Skip(advance)
	}
	return result
}
