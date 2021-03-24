package parser

import (
	"testing"

	"github.com/ava12/llx/errors"
	"github.com/ava12/llx/langdef"
)

type srcExprSample struct {
	src, expr string
}

type srcErrSample struct {
	src string
	err int
}

func testGrammarSamples (t *testing.T, name, grammar string, samples []srcExprSample, captureAside bool) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	for i, sample := range samples {
		n, e := parseAsTestNode(g, sample.src, captureAside)
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

func testErrorSamples (t *testing.T, name, grammar string, samples []srcErrSample) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	for i, sample := range samples {
		_, e := parseAsTestNode(g, sample.src, false)
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
