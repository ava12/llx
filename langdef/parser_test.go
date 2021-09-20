package langdef

import (
	"fmt"
	gr "github.com/ava12/llx/grammar"
	"strconv"
	"strings"
	"testing"

	"github.com/ava12/llx"
	"github.com/ava12/llx/source"
)

const toks = "$tok = /\\S+/;"

func checkErrorCode (t *testing.T, samples []string, code int) {
	eCode := strconv.Itoa(code)
	for index, src := range samples {
		errPrefix := "input #" + strconv.Itoa(index)
		_, e := Parse(source.New("string", []byte(src)))

		if code == 0 {
			if e != nil {
				t.Error(errPrefix + ": unexpected error: " + e.Error())
				return
			}
			continue
		}

		if e == nil {
			t.Error(errPrefix + ": error expected, got success")
			return
		}

		pe, is := e.(*llx.Error)
		if !is {
			t.Error(errPrefix + ": ParseError expected, got \"" + e.Error() + "\"")
			return
		}

		if pe.Code != code {
			t.Error(errPrefix + ": expected error code " + eCode + ", got " + strconv.Itoa(pe.Code))
			return
		}
	}
}

func TestUnexpectedEof (t *testing.T) {
	samples := []string{
		"",
		" ",
		"\n",
		"foo",
		"foo = ",
		"foo = 'bar'",
	}
	checkErrorCode(t, samples, UnexpectedEofError)
}

func TestUnexpectedToken (t *testing.T) {
	samples := []string{
		"!error =",
		"!error $foo,",
		"!extern $foo; s $foo;",
	}
	checkErrorCode(t, samples, UnexpectedTokenError)
}

func TestUnknownToken (t *testing.T) {
	samples := []string{
		"foo = $foo;",
	}
	checkErrorCode(t, samples, UnknownTokenError)
}

func TestWrongToken (t *testing.T) {
	samples := []string{
		"!aside $foo; $foo = /foo/; bar = $foo;",
		"!error $foo; $foo = /foo/; bar = $foo;",
	}
	checkErrorCode(t, samples, WrongTokenError)
}

func TestTokenDefined (t *testing.T) {
	samples := []string{
		"$foo = /a/; $bar = /b/; $foo = /c/;",
	}
	checkErrorCode(t, samples, TokenDefinedError)
}

func TestNonTerminalDefined (t *testing.T) {
	samples := []string{
		"foo = 'foo'; bar = 'bar'; foo = 'baz';",
	}
	checkErrorCode(t, samples, NonTerminalDefinedError)
}

func TestWrongRe (t *testing.T) {
	res := []string{"\x80", "(foo", "foo)", "[foo", "\\C"}
	for _, re := range res {
		s := source.New("", []byte("$foo = /" + re + "/;"))
		_, e := Parse(s)
		if e == nil {
			t.Fatalf("expected error on re /%s/", re)
		}
		ee, f := e.(*llx.Error)

		if !f {
			t.Fatalf("expected ParseError on re /%s/, got: %v", re, e)
		}
		if ee.Code != WrongRegexpError {
			t.Fatalf("expected code %d on re /%s/, got %d (%s)", WrongRegexpError, re, ee.Code, ee.Error())
		}
	}
}

func TestUnknownNonTerminal (t *testing.T) {
	samples := []string{
		"$name = /\\w+/; foo = 'foo' | bar;",
	}
	checkErrorCode(t, samples, UnknownNonTerminalError)
}

func TestUnusedNonTerminals (t *testing.T) {
	samples := []string{
		"$name = /\\w+/; foo = 'foo' | 'bar'; bar = baz | 'bar'; baz = 'baz';",
	}
	checkErrorCode(t, samples, UnusedNonTerminalError)
}

func TestUnresolved (t *testing.T) {
	samples := []string{
		"foo = bar | baz; bar = baz | foo; baz = foo | bar;",
	}
	checkErrorCode(t, samples, UnresolvedError)
}

func TestRecursions (t *testing.T) {
	samples := []string{
		"$name = /\\w+/; foo = bar; bar = bar | 'baz';",
		"$name = /\\w+/; foo = bar; bar = 'bar' | baz; baz = bar, 'baz';",
	}
	checkErrorCode(t, samples, RecursionError)
}

func TestGroupNumberError (t *testing.T) {
	var sample strings.Builder
	r := 'A'
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&sample, "!group $%c; !group $%[1]c%[1]c;\n", r)
		r++
	}
	checkErrorCode(t, []string{sample.String()}, GroupNumberError)
}

func TestUnresolvedGroupsError (t *testing.T) {
	samples := []string{
		"$num = /\\d+/; $op = /[*\\/+-]/; g = 'x' | $num, $op, $num;",
		"!caseless $key; $key = /\\w+/; g = 'x', $key;",
	}
	checkErrorCode(t, samples, UnresolvedGroupsError)
}

func TestDisjointGroupsError (t *testing.T) {
	samples := []string{
		"!group $c; $c = /\\w+/; $d = /\\d+/; g = $c | $d;",
	}
	checkErrorCode(t, samples, DisjointGroupsError)
}

func TestUndefinedTokenError (t *testing.T) {
	samples := []string{
		"!group $foo; g = $foo;",
	}
	checkErrorCode(t, samples, UndefinedTokenError)
}

func TestEmptyRepeatableError (t *testing.T) {
	samples := []string{
		"$d = /\\d/; $s = /[a-z]/; g = {[$s], [$d]};",
	}
	checkErrorCode(t, samples, EmptyRepeatableError)
}

func TestNoError (t *testing.T) {
	samples := []string{
		toks + "foo = 'foo' | bar; bar = 'bar' | 'baz';",
		toks + "!aside; !extern; !error; !shrink; !group; !literal; !caseless; !reserved; foo = 'foo';",
		"!aside $space; !group $space $name; $space = /\\s/; $name = /\\w/; g = {$name};",
		"$name = /\\w+/; !literal 'a' 'b'; g = $name;",
	}
	checkErrorCode(t, samples, 0)
}


func TestNoDuplicateLiterals (t *testing.T) {
	sample := toks + "grammar = 'a', 'foo', 'is', foo, 'or', 'a', 'bar'; foo = 'a', ('foo' | 'bar');"
	expectedTokCnt := 6
	g, e := ParseString("", sample)
	if e != nil {
		t.Fatal("unexpected error: " + e.Error())
	}

	tokCnt := len(g.Tokens)
	if tokCnt != expectedTokCnt {
		names := make([]string, tokCnt)
		for i, tok := range g.Tokens {
			names[i] = tok.Name
		}
		t.Fatalf("expecting %d toks, got %d toks: %q", expectedTokCnt, tokCnt, names)
	}
}

func TestTokenDefOrder (t *testing.T) {
	sample := "!literal 'foo'; $x = /\\w+/; !extern $y; $z = /#/; s = $x | $z | 'bar' | 'foo';"
	names := []string{"x", "z", "y", "foo", "bar"}
	g, e := ParseString("", sample)
	if e != nil {
		t.Fatalf("unexpected error: %s", e.Error())
	}

	got := make([]string, len(g.Tokens))
	for i, tok := range g.Tokens {
		got[i] = tok.Name
	}

	if len(names) != len(g.Tokens) {
		t.Fatalf("expecting %d tokens, got %d (%v)", len(names), len(g.Tokens), got)
	}

	for i, name := range got {
		if name != names[i] {
			t.Fatalf("expecting %v, got %v", names, got)
		}
	}
}

func TestTokenFlags (t *testing.T) {
	nd := "$name = /\\w+/; "
	gd := " g = 'foo';"
	samples := []struct{
		src   string
		name  string
		flags gr.TokenFlags
	}{
		{nd + gd, "foo", gr.LiteralToken},
		{nd + "!aside $sp; $sp = /\\s+/;" + gd, "sp", gr.AsideToken},
		{nd + "!caseless $name; g = 'FOO';", "name", gr.CaselessToken},
		{nd + "!error $e; $e = /\\W/;" + gd, "e", gr.ErrorToken},
		{nd + "!extern $foo;" + gd, "foo", gr.ExternalToken},
		{nd + "!literal 'foo';" + gd, "foo", gr.LiteralToken},
		{nd + "!reserved 'foo';" + gd, "foo", gr.LiteralToken | gr.ReservedToken},
		{nd + "!shrink $name;" + gd, "name", gr.ShrinkableToken},
	}

	sampleLoop:
	for i, s := range samples {
		g, e := ParseString("", s.src)
		if e != nil {
			t.Errorf("sample #%d: unexpected error: %s", i, e.Error())
			continue
		}

		for _, tok := range g.Tokens {
			if tok.Name == s.name {
				if tok.Flags != s.flags {
					t.Errorf("sample #%d: %q token: expecting flags=%d, got %d", i, s.name, s.flags, tok.Flags)
				}
				continue sampleLoop
			}
		}

		t.Errorf("sample #%d: %q token not found", i, s.name)
	}
}
