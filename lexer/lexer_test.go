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
	tokenRe *regexp.Regexp
	tokenTypes []TokenType
	tokenSamples []byte
)

func init () {
	tokenRe = regexp.MustCompile("(?s:[\\s]+|(\\d+)|([a-z_][a-z0-9_]*)|('.*?')|('.{0,10}))")
	tokenTypes = []TokenType {{1, "number"}, {2, "name"}, {3, "string"}}
	tokenSamples = []byte("123 foo 'bar'")
}

func lexer () (*Lexer, *source.Queue) {
	queue := source.NewQueue()
	return New(tokenRe, tokenTypes, queue), queue
}

func TestEmpty (t *testing.T) {
	sources := []string{"", " ", "  ", " \t\r\n "}
	for _, src := range sources {
		l, q := lexer()
		q.Append(source.New("", []byte(src)))
		tok, e := l.Next()
		if e != nil {
			t.Fatalf("source %q: unexpected error %s", src, e)
		}
		if tok.Type() != EofTokenType || tok.TypeName() != EofTokenName {
			t.Fatalf("source %q: unexpected token %s", src, tok.TypeName())
		}
	}
}

func TestTokenSamples (t *testing.T) {
	l, q := lexer()
	q.Append(source.New("", tokenSamples))
	for _, tokType := range tokenTypes {
		tok, e := l.Next()
		if tok == nil || e != nil {
			t.Fatalf("expecting %q token, got error %v", tokType.TypeName, e)
		}
		if tok.TypeName() != tokType.TypeName || tok.Type() != tokType.Type {
			t.Fatalf("expecting %q (%d) token, got %q (%d)", tokType.TypeName, tokType.Type, tok.TypeName(), tok.Type())
		}
	}
	tok, e := l.Next()
	if tok == nil || e != nil {
		t.Fatalf("expecting EoF, got %v, %v", tok, e)
	}
	if tok.TypeName() != EofTokenName {
		t.Fatalf("expecting EoF, got %q", tok.TypeName())
	}
}

func TestBrokenToken (t *testing.T) {
	l, q := lexer()
	q.Append(source.New("", []byte("\n  '*  *")))
	tok, e := l.Next()
	if tok != nil {
		t.Fatalf("expected error, got %q token", tok.TypeName())
	}
	ee, f := e.(*llx.Error)
	if !f || ee.Code != ErrBadToken {
		t.Fatalf("expected WrongTokenError, got %v", e)
	}
	if ee.Line != 2 || ee.Col != 3 {
		t.Fatalf("expected error at line 2, col 3, got %d, %d", ee.Line, ee.Col)
	}
	if !strings.Contains(ee.Message, "\"'*  *\"") {
		t.Fatalf("expected broken token in error message, got %q", ee.Message)
	}
}

func TestSourceBoundary (t *testing.T) {
	l, q := lexer()
	q.Append(source.New("", []byte("foo")))
	q.Append(source.New("", []byte("bar")))
	expectedTokens := []string{"foo", EofTokenName, "bar", EofTokenName, EoiTokenName}
	for i, expected := range expectedTokens {
		tok, e := l.Next()
		if e != nil {
			t.Fatalf("step %d: unexpected error: %s", i, e.Error())
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

func TestTokenTypes (t *testing.T) {
	re := regexp.MustCompile("(\\d+)|\\s+|(\\w+)|#.*\\n|([+-])")
	types := []TokenType{{0, "num"}, {2, "name"}, {4, "op"}}
	src := "1 + foo"
	expected := []int{0, 2, 1}

	queue := source.NewQueue().Append(source.New("", []byte(src)))
	lexer := New(re, types, queue)
	for i, n := range expected {
		tok, e := lexer.Next()
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

func TestShrinkToken (t *testing.T) {
	re := regexp.MustCompile("(\\s+)|(#[a-z]+=*)")
	types := []TokenType{{0, "space"}, {1, "name"}}
	queue := source.NewQueue()
	lexer := New(re, types, queue)
	queue.Append(source.New("", []byte("  #foo="))).Append(source.New("", []byte("#bar=")))

	tok, e := lexer.Shrink(nil)
	if tok != nil || e != nil {
		t.Fatalf("expecting nil token, got: %v, %v", tok, e)
	}

	tok, e = lexer.Next()
	if e == nil {
		tok, e = lexer.Next()
	}
	if e != nil {
		t.Fatalf("unexpected error: %s", e.Error())
	}

	for i := 4; i > 1; i-- {
		tok, e = lexer.Shrink(tok)
		if e != nil {
			t.Fatalf("step %d: unexpected error: %s", i, e.Error())
		}
		if tok == nil {
			t.Fatalf("step %d: nil token", i)
		}
		if tok.Type() != 1 || tok.TypeName() != "name" || tok.Text() != "#foo="[: i] {
			t.Fatalf("step %d: wrong token: %v", i, tok)
		}
	}

	tok, e = lexer.Shrink(tok)
	if tok != nil || e == nil {
		t.Fatalf("error expected after token shrinked, got: %v, %v", tok, e)
	}

	tok = &Token{1, "name", "#", queue.Source(), 1, 1}
	tok, e = lexer.Shrink(tok)
	if tok != nil || e != nil {
		t.Fatalf("expecting nil, nil for single char token, got: %v, %v", tok, e)
	}
}

func TestErrorPos (t *testing.T) {
	re := regexp.MustCompile("(\\s+)|(\\w+)|(<\\w+>)|(<.+)")
	types := []TokenType{
		{0, "space"},
		{1, "word"},
		{2, "tag"},
		{ErrorTokenType, ""},
	}
	samples := []struct {
		src string
		err, line, col int
	}{
		{"foo\n<bar> &baz", ErrWrongChar, 2, 7},
		{"foo\n <bar\nbaz", ErrBadToken, 2, 2},
	}
	q := source.NewQueue()
	l := New(re, types, q)
	for i, s := range samples {
		q.NextSource()
		q.Append(source.New("src", []byte(s.src)))
		tok, e := l.Next()
		for e == nil && tok != nil {
			tok, e = l.Next()
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
