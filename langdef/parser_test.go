package langdef

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	err "github.com/ava12/llx/errors"
	"github.com/ava12/llx/source"
)

func checkErrorCode (t *testing.T, samples []string, code int) {
	eCode := strconv.Itoa(code)
	for index, src := range samples {
		errPrefix := "input #" + strconv.Itoa(index)
		_, e := Parse(source.New("string", []byte(src)))

		if code == 0 {
			if e != nil {
				t.Error(errPrefix + ": unexpected error: " + e.Error())
			}
			return
		}

		if e == nil {
			t.Error(errPrefix + ": error expected, got success")
			return
		}

		pe, is := e.(*err.Error)
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

func TestUnknownTerminal (t *testing.T) {
	samples := []string{
		"foo = $foo;",
	}
	checkErrorCode(t, samples, UnknownTerminalError)
}

func TestWrongTerminal (t *testing.T) {
	samples := []string{
		"!aside $foo; $foo = /foo/; bar = $foo;",
		"!error $foo; $foo = /foo/; bar = $foo;",
	}
	checkErrorCode(t, samples, WrongTerminalError)
}

func TestTerminalDefined (t *testing.T) {
	samples := []string{
		"$foo = /a/; $bar = /b/; $foo = /c/;",
	}
	checkErrorCode(t, samples, TerminalDefinedError)
}

func TestNonerminalDefined (t *testing.T) {
	samples := []string{
		"foo = 'foo'; bar = 'bar'; foo = 'baz';",
	}
	checkErrorCode(t, samples, NonterminalDefinedError)
}

func TestWrongRe (t *testing.T) {
	res := []string{"\x80", "(foo", "foo)", "[foo", "\\C"}
	for _, re := range res {
		s := source.New("", []byte("$foo = /" + re + "/;"))
		_, e := Parse(s)
		if e == nil {
			t.Fatalf("expected error on re /%s/", re)
		}
		ee, f := e.(*err.Error)

		if !f {
			t.Fatalf("expected ParseError on re /%s/, got: %v", re, e)
		}
		if ee.Code != WrongRegexpError {
			t.Fatalf("expected code %d on re /%s/, got %d (%s)", WrongRegexpError, re, ee.Code, ee.Error())
		}
	}
}

func TestUknownNonerminal (t *testing.T) {
	samples := []string{
		"foo = 'foo' | bar;",
	}
	checkErrorCode(t, samples, UnknownNonterminalError)
}

func TestUnusedNonterminals (t *testing.T) {
	samples := []string{
		"foo = 'foo' | 'bar'; bar = baz | 'bar'; baz = 'baz';",
	}
	checkErrorCode(t, samples, UnusedNonterminalError)
}

func TestUnresolved (t *testing.T) {
	samples := []string{
		"foo = bar | baz; bar = baz | foo; baz = foo | bar;",
	}
	checkErrorCode(t, samples, UnresolvedError)
}

func TestRecursions (t *testing.T) {
	samples := []string{
		"foo = bar; bar = bar | 'baz';",
		"foo = bar; bar = 'bar' | baz; baz = bar, 'baz';",
	}
	checkErrorCode(t, samples, RecursionError)
}

func TestGroupNumberError (t *testing.T) {
	var sample strings.Builder
	r := 'A'
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&sample, "!group %c; !group %[1]c%[1]c;\n", r)
		r++
	}
	checkErrorCode(t, []string{sample.String()}, GroupNumberError)
}

func TestRedefineGroupError (t *testing.T) {
	samples := []string{
		"!group foo; !group foo;",
	}
	checkErrorCode(t, samples, RedefineGroupError)
}

func TestWrongGroupError (t *testing.T) {
	samples := []string{
		"!group foo; !extern $foo; foo = $foo;",
	}
	checkErrorCode(t, samples, WrongGroupError)
}

func TestUnresolvedGroupError (t *testing.T) {
	samples := []string{
		"!group $foo; !extern $foo; foo = 'foo';",
	}
	checkErrorCode(t, samples, UnresolvedGroupError)
}


func TestNoError (t *testing.T) {
	samples := []string{
		"foo = 'foo' | bar; bar = 'bar' | 'baz';",
		"!aside; !extern; !error; !shrink; !group; foo = 'foo';",
	}
	checkErrorCode(t, samples, 0)
}


func TestNoDuplicateLiterals (t *testing.T) {
	sample := "grammar = 'a', 'foo', 'is', foo, 'or', 'a', 'bar'; foo = 'a', ('foo' | 'bar');"
	expectedTermCnt := 5
	g, e := ParseString("", sample)
	if e != nil {
		t.Fatal("unexpected error: " + e.Error())
	}

	termCnt := len(g.Terms)
	if termCnt != expectedTermCnt {
		names := make([]string, termCnt)
		for i, term := range g.Terms {
			names[i] = term.Name
		}
		t.Fatalf("expecting %d terms, got %d terms: %q", expectedTermCnt, termCnt, names)
	}
}

