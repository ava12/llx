package lexer

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/ava12/llx"
	"github.com/ava12/llx/source"
)

var (
	tokenRe      *regexp.Regexp
	tokenTypes   []TokenType
	tokenSamples []byte
)

func init() {
	tokenRe = regexp.MustCompile("(?s:[\\s]+|(-?\\d+)|([a-z_][a-z0-9_]*)|('.*?')|('.{0,10}))")
	tokenTypes = []TokenType{{1, "number"}, {2, "name"}, {3, "string"}}
	tokenSamples = []byte("123 foo 'bar'")
}

func lexer() (*Lexer, *source.Queue) {
	queue := source.NewQueue()
	return New(tokenRe, tokenTypes), queue
}

func TestEmpty(t *testing.T) {
	sources := []string{"", " ", "  ", " \t\r\n "}
	for _, src := range sources {
		l, q := lexer()
		q.Append(source.New("", []byte(src)))
		tok, e := l.Next(q)
		if e != nil {
			t.Fatalf("source %q: unexpected error %s", src, e)
		}
		if tok.Type() != EofTokenType || tok.TypeName() != EofTokenName {
			t.Fatalf("source %q: unexpected token %s", src, tok.TypeName())
		}
	}
}

func TestTokenSamples(t *testing.T) {
	l, q := lexer()
	q.Append(source.New("", tokenSamples))
	for _, tokType := range tokenTypes {
		tok, e := l.Next(q)
		if tok == nil || e != nil {
			t.Fatalf("expecting %q token, got error %v", tokType.TypeName, e)
		}
		if tok.TypeName() != tokType.TypeName || tok.Type() != tokType.Type {
			t.Fatalf("expecting %q (%d) token, got %q (%d)", tokType.TypeName, tokType.Type, tok.TypeName(), tok.Type())
		}
	}
	tok, e := l.Next(q)
	if tok == nil || e != nil {
		t.Fatalf("expecting EoF, got %v, %v", tok, e)
	}
	if tok.TypeName() != EofTokenName {
		t.Fatalf("expecting EoF, got %q", tok.TypeName())
	}
}

func TestBrokenToken(t *testing.T) {
	l, q := lexer()
	q.Append(source.New("", []byte("\n  '*  *")))
	tok, e := l.Next(q)
	if tok != nil {
		t.Fatalf("expected error, got %q token", tok.TypeName())
	}
	ee, f := e.(*llx.Error)
	if !f || ee.Code != BadTokenError {
		t.Fatalf("expected WrongTokenError, got %v", e)
	}
	if ee.Line != 2 || ee.Col != 3 {
		t.Fatalf("expected error at line 2, col 3, got %d, %d", ee.Line, ee.Col)
	}
	if !strings.Contains(ee.Message, "\"'*  *\"") {
		t.Fatalf("expected broken token in error message, got %q", ee.Message)
	}
}

func TestSourceBoundary(t *testing.T) {
	l, q := lexer()
	q.Append(source.New("", []byte("foo")))
	q.Append(source.New("", []byte("bar")))
	expectedTokens := []string{"foo", EofTokenName, "bar", EofTokenName, EoiTokenName, EoiTokenName}
	for i, expected := range expectedTokens {
		tok, e := l.Next(q)
		if e != nil {
			t.Fatalf("step %d: unexpected error: %s", i, e.Error())
		}

		if tok == nil {
			t.Fatalf("step %d: got nil token", i)
		}

		got := tok.Text()
		if got == "" {
			got = tok.TypeName()
		}
		if got != expected {
			t.Fatalf("step %d: expecting %q token, got %q", i, expected, got)
		}
	}
}

func TestTokenTypes(t *testing.T) {
	re := regexp.MustCompile("(-?\\d+)|\\s+|(\\w+)|#.*\\n|([+-])")
	types := []TokenType{{0, "num"}, {2, "name"}, {4, "op"}}
	src := "1 + foo -2"
	expected := []int{0, 2, 1, 0}

	q := source.NewQueue().Append(source.New("", []byte(src)))
	lexer := New(re, types)
	for i, n := range expected {
		tok, e := lexer.Next(q)
		if e != nil {
			t.Fatalf("sample #%d: unexpected error: %s", i, e.Error())
		}
		if tok.Type() != types[n].Type || tok.TypeName() != types[n].TypeName {
			t.Fatalf(
				"sample #%d: expecting token %q (%d), got %q (%d)",
				i,
				types[n].TypeName,
				types[n].Type,
				tok.TypeName(),
				tok.Type(),
			)
		}
	}
}

func TestErrorPos(t *testing.T) {
	re := regexp.MustCompile("(\\s+)|(\\w+)|(<\\w+>)|(<.+)")
	types := []TokenType{
		{0, "space"},
		{1, "word"},
		{2, "tag"},
		{ErrorTokenType, ""},
	}
	samples := []struct {
		src            string
		err, line, col int
	}{
		{"foo\n<bar> &baz", WrongCharError, 2, 7},
		{"foo\n <bar\nbaz", BadTokenError, 2, 2},
	}
	q := source.NewQueue()
	l := New(re, types)
	for i, s := range samples {
		q.NextSource()
		q.Append(source.New("src", []byte(s.src)))
		tok, e := l.Next(q)
		for e == nil && tok != nil {
			tok, e = l.Next(q)
		}

		if e == nil {
			t.Errorf("sample %d: expecting an error, got EoF", i)
			continue
		}

		ee, f := e.(*llx.Error)
		if !f {
			t.Errorf("sample %d: expecting *llx.Error, got: %s", i, e)
			continue
		}

		tail := fmt.Sprintf("line %d col %d", s.line, s.col)
		if ee.Code != s.err || !strings.HasSuffix(ee.Message, tail) {
			t.Errorf("sample %d: expecting err %d at line %d col %d, got: %s", i, s.err, s.line, s.col, ee.Message)
		}
	}
}

func TestNextOf(t *testing.T) {
	re := regexp.MustCompile(`([a-z]+)|(-?\d+)|(".*?")|([+-])|(".{0,10})`)
	types := []TokenType{
		{1, "name"},
		{2, "number"},
		{3, "string"},
		{4, "op"},
		{ErrorTokenType, "string"},
	}
	samples := []struct {
		src            string
		types          TokenTypeSet
		err, tokenType int
		rescan         bool
	}{
		{"foo", 0b110, 0, 1, false},
		{"bar", 0b1100, 0, 1, true},
		{"?", AllTokenTypes, WrongCharError, 0, false},
		{`"broken`, AllTokenTypes, BadTokenError, 0, false},
		{"-123", 0b11010, 0, 0, false},
		{"-123", AllTokenTypes, 0, 2, false},
	}

	l := New(re, types)
	for i, s := range samples {
		name := fmt.Sprintf("s #%d", i)
		t.Run(name, func(t *testing.T) {
			q := source.NewQueue().Append(source.New("test", []byte(s.src)))
			tok, e := l.NextOf(q, s.types)

			if tok == nil && s.err == 0 && s.rescan {
				tok, e = l.Next(q)
			}

			if (e == nil) != (s.err == 0) {
				if e == nil {
					t.Errorf("expecting error code %d, got success", s.err)
				} else {
					t.Errorf("unexpected error %s", e.Error())
				}
				return
			}

			if e != nil {
				ee, valid := e.(*llx.Error)
				if !valid || ee.Code != s.err {
					t.Errorf("expecting error code %d, got %s", s.err, e.Error())
				}
				return
			}

			if tok == nil {
				if s.tokenType != 0 {
					t.Errorf("expecting token type %d, got nothing", s.tokenType)
				}
			} else if tok.Type() != s.tokenType {
				t.Errorf("expecting token type %d, got type %d", s.tokenType, tok.Type())
			}
		})
	}
}
