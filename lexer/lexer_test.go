package lexer

import (
	"regexp"
	"strings"
	"testing"

	err "github.com/ava12/llx/errors"
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
	ee, f := e.(*err.Error)
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
	t1, e1 := l.Next()
	t2, e2 := l.Next()
	if t1 == nil || t1.Text() != "foo" || t2 == nil || t2.Text() != "bar" {
		t.Fatalf("expected 2 tokens, got: %v %v / %v %v", t1, e1, t2, e2)
	}
}

func TestEof (t *testing.T) {
	l, q := lexer()
	q.Append(source.New("", []byte("foo")))
	l.Next()
	t1, e1 := l.Next()
	t2, e2 := l.Next()
	if t1 == nil || t1.TypeName() != EofTokenName || e1 != nil || t2 != nil || e2 != nil {
		t.Fatalf("expected EoF token and nil, got: %v %v / %v %v", t1, e1, t2, e2)
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
	re := regexp.MustCompile("(?:#([a-z]+)=*)")
	types := []TokenType{{1, "name"}}
	queue := source.NewQueue()
	lexer := New(re, types, queue)
	queue.Append(source.New("", []byte("#foo="))).Append(source.New("", []byte("#bar=")))

	tok, e := lexer.ShrinkToken()
	if tok != nil || e == nil {
		t.Fatalf("error expected before token fetch, got: %v, %v", tok, e)
	}

	_, e = lexer.Next()
	if e != nil {
		t.Fatalf("unexpected error: %s", e.Error())
	}

	for i := 3; i > 0; i-- {
		tok, e = lexer.ShrinkToken()
		if e != nil {
			t.Fatalf("step %d: unexpected error: %s", i, e.Error())
		}
		if tok == nil {
			t.Fatalf("step %d: nil token", i)
		}
		if tok.Type() != 1 || tok.TypeName() != "name" || tok.Text() != "foo"[: i] {
			t.Fatalf("step %d: wrong token: %v", i, tok)
		}
	}

	tok, e = lexer.ShrinkToken()
	if tok != nil || e == nil {
		t.Fatalf("error expected after token shrinked, got: %v, %v", tok, e)
	}
}
