package parser

import (
	"testing"

	"github.com/ava12/llx/errors"
	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

type srcExprSample struct {
	src, expr string
}

type srcErrSample struct {
	src string
	err int
}

var testTokenHooks = TokenHooks{
	AnyTokenType: func (token *lexer.Token, pc *ParseContext) (emit bool, e error) {
		return true, nil
	},
}

func testGrammarSamplesWithHooks (t *testing.T, name, grammar string, samples []srcExprSample, hs TokenHooks) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	for i, sample := range samples {
		n, e := parseAsTestNode(g, sample.src, hs)
		if e != nil {
			t.Errorf("grammar %q, sample #%d: got error: %s", name, i, e.Error())
			continue
		}

		e = newTreeValidator(n, sample.expr).validate()
		if e != nil {
			t.Errorf("grammar %q, sample #%d: validation error: %s", name, i, e.Error())
		}
	}
}

func testGrammarSamples (t *testing.T, name, grammar string, samples []srcExprSample, captureAside bool) {
	var hs TokenHooks

	if captureAside {
		hs = testTokenHooks
	}
	testGrammarSamplesWithHooks(t, name, grammar, samples, hs)
}

func testErrorSamples (t *testing.T, name, grammar string, samples []srcErrSample) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	for i, sample := range samples {
		_, e := parseAsTestNode(g, sample.src, nil)
		if e == nil {
			t.Errorf("grammar %q, sample #%d: expecting error code %d, got success", name, i, sample.err)
			continue
		}

		le, f := e.(*errors.Error)
		if !f {
			t.Errorf("grammar %q, sample #%d: expecting llx error code %d, got: %s", name, i, sample.err, e.Error())
			continue
		}

		if le.Code != sample.err {
			t.Errorf("grammar %q, sample #%d: expecting error code %d, got code %d (%s)", name, i, sample.err, le.Code, le.Error())
			continue
		}
	}
}

const spaceDef = "!aside $space; $space = /\\s+/; "

func TestErrors (t *testing.T) {
	name := "errors"
	grammar := spaceDef + "$name = /\\w+/; $op = /[()]/; s = 'foo' | 'bar', '(', 'bar' | 'baz', ')';"
	samples := []srcErrSample{
		{"foo(bar", UnexpectedEofError},
		{"foo(bar baz", UnexpectedTokenError},
	}
	testErrorSamples(t, name, grammar, samples)
}

func TestUnexpectedGroupError (t *testing.T) {
	grammar := spaceDef + "!group $op; $name = /\\w+/; $op = /\\+/; g = $name, $op, $name;"
	sample := "foo + bar"
	g, e := langdef.ParseString("sample", grammar)
	var (
		pc *ParseContext
		q *source.Queue
	)
	if e == nil {
		q = source.NewQueue().Append(source.New("sample", []byte(sample)))
		p := New(g)
		pc, e = newParseContext(p, q, &Hooks{})
	}
	if e != nil {
		t.Fatal("unexpected error:" + e.Error())
	}

	pc.tokens = append(pc.tokens, lexer.NewToken(2, "op", "+", q.SourcePos()))
	_, e = pc.nextToken(1)
	if e == nil {
		t.Fatal("expecting UnexpectedGroupError, got success")
	}

	ee, f := e.(*errors.Error)
	if !f || ee.Code != UnexpectedGroupError {
		t.Fatal("expecting UnexpectedGroupError, got: " + e.Error())
	}
}

func TestSimple (t *testing.T) {
	name := "simple"
	grammar := "$char = /\\w/; s = {a | b | c}; a = 'a',{'a'}; b = 'b', ['b']; c = 'c', {a | b | c};"
	samples := []srcExprSample{
		{"aaa", "(a a a a)"},
		{"bbb", "(b b b)(b b)"},
		{"cabcaa", "(c c (a a) (b b) (c c (a a a)))"},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestAside (t *testing.T) {
	name := "aside"
	grammar := "!aside $sep; $sep = /-/; $char = /\\w/; s = {'a' | 'b' | 'c'};"
	samples := []srcExprSample{
		{"abc", "a b c"},
		{"a-a-a", "a - a - a"},
		{"--b--c--", "- - b - - c - -"},
	}
	testGrammarSamples(t, name, grammar, samples, true)
}

func TestAri (t *testing.T) {
	name := "ari"
	grammar := spaceDef + "$num=/\\d+/; $op=/[()^*\\/+-]/; ari=sum; sum=pro,{('+'|'-'),pro}; pro=pow,{('*'|'/'),pow}; pow=val,{'^',val}; val=$num|('(',sum,')');"
	samples := []srcExprSample{
		{
			"2 + 2",
			"(sum (pro (pow (val 2))) + (pro (pow (val 2))))",
		},
		{
			"2 + 3^4*5",
			"(sum (pro (pow (val 2))) + (pro (pow (val 3) ^ (val 4)) * (pow (val 5))))",
		},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestMultiRuleAri (t *testing.T) {
	name := "ari"
	grammar := spaceDef + "$num=/\\d+/; $op=/[()^*\\/+-]/;" +
		"ari=val|pow|pro|sum;" +
		"val=$num|('(',sum,')');" +
		"pow=val,'^',val,{'^',val};" +
		"pro=val|pow,('*'|'/'),val|pow,{('*'|'/'),val|pow};" +
		"sum=val|pow|pro,('+'|'-'),val|pow|pro,{('+'|'-'),val|pow|pro};"
	samples := []srcExprSample{
		{
			"2",
			"(val 2)",
		},
		{
			"2 + 2",
			"(sum (val 2) + (val 2))",
		},
		{
			"2 + 3^4*5",
			"(sum (val 2) + (pro (pow (val 3) ^ (val 4)) * (val 5)))",
		},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestGroups (t *testing.T) {
	name := "groups"
	grammar := "!aside $space; !group $space $eol $name; !group $space $str $any;" +
		"$space = /[ \\t]+/; $eol = /\\n/; $name = /\\w+/; $str = /<.*?>/; $any = /[^\\n]+/;" +
		"g = {$name, val, $eol}; val = $str | $any;"
	samples := []srcExprSample{
		{
			"foo bar\nbar <bar baz>\nbaz baz qux\n",
			"foo (val bar) eol bar (val '<bar baz>') eol baz (val 'baz qux') eol",
		},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestMultiGroups (t *testing.T) {
	name := "multi-groups"
	grammar := spaceDef + "!group $space $name; !group $space $str $any; $name = /[a-z]+/; $str = /<.*?>/; $any = /[^\\n]+/;" +
		"g = {s | a}; s = $name, $str; a = $name, $any;"
	samples := []srcExprSample{
		{
			"foo <foo> bar bar baz",
			"(s foo '<foo>') (a bar 'bar baz')",
		},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestTokenShrinking (t *testing.T) {
	name := "shrinking"
	grammar := spaceDef + "!shrink $op; $name = /\\w+/; $op = /[()]|<<?|>>?/;" +
		"g = val, {val};" +
		"val = $name | pair | group; pair = '(', $name, '<<' | '>>', val, ')'; group = '<', val, {val}, '>';"
	samples := []srcExprSample{
		{
			"<<foo> bar>",
			"(val (group < (val (group < (val foo) >)) (val bar) >))",
		},
		{
			"<foo <bar>>",
			"(val (group < (val foo) (val (group < (val bar) > )) > ))",
		},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestTokenHooks (t *testing.T) {
	name := "token hooks"
	grammar := "$space = /\\s+/; $char = /[bcdf]|aa?|ee?/; !literal 'a' 'b' 'c' 'd' 'e' 'f';" +
		"g = {('b', 'aa') | ('b', 'ee', 'f') | ('f', 'a', 'c', 'e') | $space};"
	samples := []srcExprSample{
		{"fce baabbef baa", "f a c e _ b aa b ee f _ b aa"},
	}

	prevTokenText := ""
	hooks := TokenHooks{
		6: func (t *lexer.Token, pc *ParseContext) (bool, error) {
			if prevTokenText != "b" {
				return true, nil
			}

			return false, pc.EmitToken(lexer.NewToken(6, "char", "ee", t)) // e -> ee
		},
		4: func (t *lexer.Token, pc *ParseContext) (bool, error) {
			return true, pc.EmitToken(lexer.NewToken(4, "char", "a", t)) // c -> a c
		},
		1: func (t *lexer.Token, pc *ParseContext) (bool, error) {
			f := (t.Text() != prevTokenText) // x x -> x
			prevTokenText = t.Text()
			return f, nil
		},
		AnyTokenType: func (t *lexer.Token, pc *ParseContext) (bool, error) {
			return false, pc.EmitToken(lexer.NewToken(0, "space", "_", t)) // " " -> _
		},
	}

	testGrammarSamplesWithHooks(t, name, grammar, samples, hooks)
}
