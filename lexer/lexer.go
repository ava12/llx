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

// TokenTypeSet represents a set of expected token types, each one is coded as 1 << type.
type TokenTypeSet = uint64

const AllTokenTypes = TokenTypeSet(1<<64 - 1)

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
// A group that has no description or that has token type < 0 or > 63 is treated as ErrorTokenType.
func New(re *regexp.Regexp, types []TokenType) *Lexer {
	ts := make([]TokenType, len(types))
	for i, t := range types {
		ts[i].TypeName = t.TypeName
		if t.Type >= 0 && t.Type < 64 {
			ts[i].Type = t.Type
		} else {
			ts[i].Type = ErrorTokenType
		}
	}
	return &Lexer{types: ts, re: re}
}

func wrongCharError(s *source.Source, content []byte, line, col int) *llx.Error {
	r, _ := utf8.DecodeRune(content)
	msg := fmt.Sprintf("wrong char \"%c\" (u+%x)", r, r)
	return llx.NewError(WrongCharError, msg, s.Name(), line, col)
}

func wrongTokenError(t *Token) *llx.Error {
	return llx.FormatErrorPos(t, BadTokenError, "bad token %q", t.Text())
}

func (l *Lexer) matchToken(src *source.Source, content []byte, pos int, tts TokenTypeSet) (*Token, int, error) {
	content = content[pos:]
	match := l.re.FindSubmatchIndex(content)
	if len(match) == 0 || match[0] != 0 || match[1] <= match[0] {
		line, col := src.LineCol(pos)
		return nil, 0, wrongCharError(src, content, line, col)
	}

	subMaskMatched := false
	for i := 2; i < len(match); i += 2 {
		if match[i] >= 0 && match[i+1] >= 0 {
			subMaskMatched = true
			sp := source.NewPos(src, pos+match[i])
			tokenType := ErrorTokenType
			typeName := ErrorTokenName
			if len(l.types) >= (i >> 1) {
				tokenType = l.types[(i>>1)-1].Type
				typeName = l.types[(i>>1)-1].TypeName
				if tokenType >= 0 && tts&(1<<tokenType) == 0 {
					continue
				}
			}
			token := NewToken(tokenType, typeName, content[match[i]:match[i+1]], sp)
			if tokenType == ErrorTokenType {
				return nil, 0, wrongTokenError(token)
			}

			return token, match[1], nil
		}
	}

	advance := 0
	if !subMaskMatched {
		advance = match[1]
	}
	return nil, advance, nil
}

func (l *Lexer) fetch(q *source.Queue, tSet TokenTypeSet) (*Token, bool, error) {
	content, pos := q.ContentPos()
	src := q.Source()
	if len(content)-pos <= 0 {
		if src == nil {
			return EoiToken(), false, nil
		} else {
			q.NextSource()
			return EofToken(src), false, nil
		}
	}

	tok, advance, e := l.matchToken(src, content, pos, tSet)
	q.Skip(advance)
	return tok, advance > 0, e
}

// Next fetches token starting at current source position and advances current position.
// Returns nil token and llx.Error and does not make any changes if there is a lexical error.
// Returns EoI token if queue is empty.
// Returns EoF token and discards current source if current position is beyond the end of current source.
func (l *Lexer) Next(q *source.Queue) (*Token, error) {
	for {
		t, _, e := l.fetch(q, AllTokenTypes)
		if t != nil || e != nil {
			return t, e
		}
	}
}

// NextOf fetches token of specified type starting at current source position and advances current position.
// Returns nil, nil and makes no changes if cannot fetch token of one of specified types.
// Returns nil token and llx.Error and does not make any changes if there is a lexical error.
// Returns EoI token if queue is empty.
// Returns EoF token and discards current source if current position is beyond the end of current source.
func (l *Lexer) NextOf(q *source.Queue, tts TokenTypeSet) (*Token, error) {
	for {
		t, advanced, e := l.fetch(q, tts)
		if t != nil || e != nil || !advanced {
			return t, e
		}
	}
}
