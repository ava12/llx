package langdef

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/ava12/llx/grammar"
	gr "github.com/ava12/llx/grammar"

	"github.com/ava12/llx"
	"github.com/ava12/llx/source"
)

func checkErrorCode(t *testing.T, samples []string, code int) {
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
			t.Errorf("%s: ParseError expected, got %q", errPrefix, e.Error())
			return
		}

		if pe.Code != code {
			t.Errorf("%s: expected error code %d, got %d (%s)", errPrefix, code, pe.Code, pe.Error())
			return
		}
	}
}

func TestUnexpectedEof(t *testing.T) {
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

func TestUnexpectedToken(t *testing.T) {
	samples := []string{
		"!error =",
		"!error $foo,",
		"!extern $foo; s $foo;",
		"$s = /\\s+/; $n = /\\d+/; g = {$n}; !side",
		"@foo bar((",
		"@foo bar(baz,)",
		"@foo bar(,)",
		"@foo bar;",
		"@foo bar(),",
	}
	checkErrorCode(t, samples, UnexpectedTokenError)
}

func TestUnknownToken(t *testing.T) {
	samples := []string{
		"foo = $foo;",
	}
	checkErrorCode(t, samples, UnknownTokenError)
}

func TestWrongToken(t *testing.T) {
	samples := []string{
		"!side $foo; $foo = /foo/; bar = $foo;",
		"!error $foo; $foo = /foo/; bar = $foo;",
	}
	checkErrorCode(t, samples, WrongTokenError)
}

func TestTokenDefined(t *testing.T) {
	samples := []string{
		"$foo = /a/; $bar = /b/; $foo = /c/;",
		"$ = /foo/;",
	}
	checkErrorCode(t, samples, TokenDefinedError)
}

func TestNodeDefined(t *testing.T) {
	samples := []string{
		"foo = 'foo'; bar = 'bar'; foo = 'baz';",
	}
	checkErrorCode(t, samples, NodeDefinedError)
}

func TestWrongRe(t *testing.T) {
	res := []string{"\x80", "(foo", "foo)", "[foo", "\\C"}
	for _, re := range res {
		s := source.New("", []byte("$foo = /"+re+"/;"))
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

func TestUnknownNode(t *testing.T) {
	samples := []string{
		"$name = /\\w+/; foo = 'foo' | bar;",
	}
	checkErrorCode(t, samples, UnknownNodeError)
}

func TestUnusedNode(t *testing.T) {
	samples := []string{
		"$name = /\\w+/; foo = 'foo' | 'bar'; bar = baz | 'bar'; baz = 'baz';",
	}
	checkErrorCode(t, samples, UnusedNodeError)
}

func TestUnresolved(t *testing.T) {
	samples := []string{
		"foo = bar | baz; bar = baz | foo; baz = foo | bar;",
	}
	checkErrorCode(t, samples, UnresolvedError)
}

func TestRecursions(t *testing.T) {
	samples := []string{
		"$name = /\\w+/; foo = bar; bar = bar | 'baz';",
		"$name = /\\w+/; foo = bar; bar = 'bar' | baz; baz = bar, 'baz';",
	}
	checkErrorCode(t, samples, RecursionError)
}

func TestTokenTypeNumberError(t *testing.T) {
	var sample strings.Builder
	for i := 0; i <= gr.MaxTokenType+1; i++ {
		fmt.Fprintf(&sample, "$t%d = /./;\n", i)
	}
	sample.WriteString("foo = 'bar';")
	checkErrorCode(t, []string{sample.String()}, TokenTypeNumberError)
}

func TestUnresolvedGroupsError(t *testing.T) {
	samples := []string{
		"$num = /\\d+/; $op = /[*\\/+-]/; g = 'x' | $num, $op, $num;",
		"!caseless $key; $key = /\\w+/; g = 'x', $key;",
		"!literal $num; $num = /\\d+/; $name = /\\w+/; g = 'foo' | $name | $num;",
	}
	checkErrorCode(t, samples, UnresolvedTokenTypesError)
}

func TestUndefinedTokenError(t *testing.T) {
	samples := []string{
		"!caseless $foo; g = $foo;",
	}
	checkErrorCode(t, samples, UndefinedTokenError)
}

func TestUnknownLiteralError(t *testing.T) {
	samples := []string{
		"!literal 'foo'; $name = /\\w+/; g = $name | 'foo' | 'bar';",
	}
	checkErrorCode(t, samples, UnknownLiteralError)
}

func TestReassignedGroupError(t *testing.T) {
	samples := []string{
		"$num = /\\d+/; $name = /\\w+/; !group $name; !group $name $num;",
	}
	checkErrorCode(t, samples, ReassignedGroupError)
}

func TestTemplateDefinedError(t *testing.T) {
	samples := []string{
		"$$foo = /bar/; $$foo = /baz/;",
	}
	checkErrorCode(t, samples, TemplateDefinedError)
}

func TestUnknownTemplateError(t *testing.T) {
	samples := []string{
		"$foo = /bar/ baz;",
	}
	checkErrorCode(t, samples, UnknownTemplateError)
}

func TestUnknownDirectiveError(t *testing.T) {
	samples := []string{
		"!dir;",
	}
	checkErrorCode(t, samples, UnknownDirectiveError)
}
func TestNoError(t *testing.T) {
	samples := []string{
		"$tok = /\\S+/; foo = 'foo' | bar; bar = 'bar' | 'baz';",
		"$tok = /\\S+/; !side; !extern; !error; !literal; !caseless; !reserved; @foo; foo = 'foo';",
		"!side $space; !group $space; $space = /\\s/; $name = /\\w/; g = {$name};",
		"$name = /\\w+/; !literal 'a' 'b'; g = $name;",
		"!literal $name 'a' 'b'; $name = /\\w+/; g = $name | 'a' | 'b';",
		"!extern $ex; $name = /\\S+/; g = $name, $ex;",
		"$n = /\\S+/; g = $n; @foo;",
		"$n = /\\S+/; g = $n; @ foo bar() baz();",
		"$op = /[+-]/; g = \"+\", \"+\" | \"-\";",
		"$op = /[+-]/; g = '+', '+' | '-';",
		"$$foo = /foo/; $$bar = /a/foo/z/; $tok = foo/-/bar; g = $tok;",
		"$nl = /\\n/; $name = /\\S+/; g = {$nl | ($name, $nl | $)};",
	}
	checkErrorCode(t, samples, 0)
}

func TestNoDuplicateLiterals(t *testing.T) {
	sample := "$tok = /\\S+/; grammar = 'a', 'foo', 'is', foo, 'or', 'a', 'bar'; foo = 'a', ('foo' | 'bar');"
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

func TestTokenDefOrder(t *testing.T) {
	sample := "!reserved 'foo'; $x = /\\w+/; !extern $y; $z = /#/; s = $x | $z | 'bar' | 'foo';"
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

func TestTokenFlags(t *testing.T) {
	nd := "$name = /\\w+/; "
	gd := " g = 'foo';"
	samples := []struct {
		src   string
		name  string
		flags gr.TokenFlags
	}{
		{nd + gd, "foo", gr.LiteralToken},
		{nd + "!side $sp; $sp = /\\s+/;" + gd, "sp", gr.SideToken},
		{nd + "!caseless $name; g = 'FOO';", "name", gr.CaselessToken},
		{nd + "!error $e; $e = /\\W/;" + gd, "e", gr.ErrorToken},
		{nd + "!extern $foo;" + gd, "foo", gr.ExternalToken},
		{nd + "!literal 'foo';" + gd, "foo", gr.LiteralToken},
		{nd + "!reserved 'foo';" + gd, "foo", gr.LiteralToken | gr.ReservedToken},
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

func TestStringEscape(t *testing.T) {
	samples := []struct {
		text, expected string
		code           int
	}{
		{`'\'`, `\`, 0},
		{`"\\"`, `\`, 0},
		{`"\""`, `"`, 0},
		{`"\n"`, "\n", 0},
		{`"\r"`, "\r", 0},
		{`"\t"`, "\t", 0},
		{`"\x4A\x4b"`, `JK`, 0},
		{`"\u0401\u0436"`, `Ёж`, 0},
		{`"\U001012Cd"`, "\U001012cd", 0},
		{`"hello\r\nworld"`, "hello\r\nworld", 0},
		{`"it\'s"`, "", InvalidEscapeError},
		{`"\'"`, "", InvalidEscapeError},
		{`"\N"`, "", InvalidEscapeError},
		{`"\x1"`, "", InvalidEscapeError},
		{`"\u123"`, "", InvalidEscapeError},
		{`"\U0010123"`, "", InvalidEscapeError},
		{`"\ud800\udc01"`, "", InvalidRuneError},
		{`"\ud9ab"`, "", InvalidRuneError},
		{`"\udddd"`, "", InvalidRuneError},
		{`"\U00110001"`, "", InvalidRuneError},
	}

	grammarTpl := `$space = /\s+/; $word = /\S.*/; g = %s;`

	for _, sample := range samples {
		t.Run(sample.text, func(t *testing.T) {
			grammar := fmt.Sprintf(grammarTpl, sample.text)
			result, e := ParseString("test", grammar)

			if e == nil {
				if sample.code != 0 {
					t.Errorf("expecting error code %d, got no error", sample.code)
				} else if result.Tokens[len(result.Tokens)-1].Name != sample.expected {
					t.Errorf("expecting %q, got %q", sample.expected, result.Tokens[len(result.Tokens)-1].Name)
				}
			} else {
				le, valid := e.(*llx.Error)
				if !valid {
					t.Errorf("expexting llx.Error, got %v", e)
				} else if le.Code != sample.code {
					t.Errorf("expecting error code %d, got %d", sample.code, le.Code)
				}
			}
		})
	}
}

func TestLayer(t *testing.T) {
	sample := `$n = /\\S+/; g = $n;
	@foo;
	@bar void() zig(zag) flip("flap", 'flop');
	@foo some(thing);
	`
	expected := []grammar.Layer{
		{Type: "foo"},
		{Type: "bar", Commands: []grammar.LayerCommand{
			{Command: "void"},
			{Command: "zig", Arguments: []string{"zag"}},
			{Command: "flip", Arguments: []string{"flap", "flop"}},
		}},
		{Type: "foo", Commands: []grammar.LayerCommand{
			{Command: "some", Arguments: []string{"thing"}},
		}},
	}

	g, e := ParseString("", sample)
	if e != nil {
		t.Fatalf("got unexpected error %v", e)
	}

	if len(g.Layers) != len(expected) {
		t.Fatalf("expecting %d layers, got %d (%v)", len(expected), len(g.Layers), g.Layers)
	}

	check := func(exp, got grammar.Layer) bool {
		if got.Type != exp.Type || len(got.Commands) != len(exp.Commands) {
			return false
		}

		for i, gotCmd := range got.Commands {
			expCmd := exp.Commands[i]
			if gotCmd.Command != expCmd.Command || len(gotCmd.Arguments) != len(expCmd.Arguments) {
				return false
			}

			for j, gotArg := range gotCmd.Arguments {
				expArg := expCmd.Arguments[j]
				if gotArg != expArg {
					return false
				}
			}
		}

		return true
	}

	for i, layer := range g.Layers {
		if !check(expected[i], layer) {
			t.Errorf("layer %d: expecting %v, got %v", i, expected[i], layer)
		}
	}
}

func TestTemplates(t *testing.T) {
	sample := `$$dd = /\d\d/; $$date = /\d{4}-/dd/-/dd; $$time = dd/:/dd/:/dd;
	$date = date; $time = time; $datetime = date /T/ time;
	g = $date | $time | $datetime;`

	expected := []string{
		`\d{4}-\d\d-\d\d`,
		`\d\d:\d\d:\d\d`,
		`\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d`,
	}

	g, e := ParseString("", sample)
	if e != nil {
		t.Fatalf("unexpected error: %v", e)
	}

	for i, re := range expected {
		if g.Tokens[i].Re != re {
			t.Errorf("token #%d(%s): expecting regexp %q, got %q",
				i, g.Tokens[i].Name, re, g.Tokens[i].Re)
		}
	}
}
