package parser

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ava12/llx"
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
	AnyToken: func (token *lexer.Token, pc *ParseContext) (emit bool, e error) {
		return true, nil
	},
}

func testGrammarSamplesWithHooks (t *testing.T, name, grammar string, samples []srcExprSample, ths, lhs TokenHooks) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	for i, sample := range samples {
		n, e := parseAsTestNode(g, sample.src, ths, lhs)
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
	testGrammarSamplesWithHooks(t, name, grammar, samples, hs, nil)
}

func testErrorSamplesWithHooks (t *testing.T, name, grammar string, samples []srcErrSample, hs *Hooks) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	parser := New(g)

	for i, sample := range samples {
		q := source.NewQueue().Append(source.New("sample", []byte(sample.src)))
		_, e := parser.Parse(q, hs)
		if e == nil {
			t.Errorf("grammar %q, sample #%d: expecting error code %d, got success", name, i, sample.err)
			continue
		}

		le, f := e.(*llx.Error)
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

func testErrorSamples (t *testing.T, name, grammar string, samples []srcErrSample) {
	testErrorSamplesWithHooks(t, name, grammar, samples, nil)
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

func TestHandlerKeyErrors (t *testing.T) {
	name := "handler key errors"
	grammar := "$any = /./; g = $any;"
	queue := source.NewQueue().Append(source.New("x", []byte("x")))
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	parser := New(g)

	samples := []struct{
		hooks Hooks
		err int
	}{
		{Hooks{TokenHooks{"space": nil}, nil, nil}, UnknownTokenTypeError},
		{Hooks{nil, TokenHooks{"y": nil}, nil}, UnknownTokenLiteralError},
		{Hooks{nil, nil, NonTermHooks{"foo": nil}}, UnknownNonTermError},
	}

	for i, sample := range samples {
		_, e := parser.Parse(queue, &sample.hooks)
		var (
			ee *llx.Error
			code int
			f bool
		)
		if e != nil {
			ee, f = e.(*llx.Error)
			if f {
				code = ee.Code
			}
		}
		if e == nil || !f || code != sample.err {
			t.Errorf("sample #%d: expecting error code %d, got: %s (code %d)", i, sample.err, e, code)
		}
	}
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

	pc.tokens.Append(lexer.NewToken(2, "op", "+", nil))
	_, e = pc.nextToken(1)
	if e == nil {
		t.Fatal("expecting UnexpectedGroupError, got success")
	}

	ee, f := e.(*llx.Error)
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
	grammar := "!aside $space; !group $eol $name; !group $str $any;" +
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
	grammar := spaceDef + "!group $name; !group $str $any; $name = /[a-z]+/; $str = /<.*?>/; $any = /[^\\n]+/;" +
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
	grammar := "$space = /\\s+/; $char = /[bcdf]|aa?|ee?/; !literal 'a' 'b' 'c' 'd' 'e' 'f' 'aa' 'ee';" +
		"g = {('b', 'aa') | ('b', 'ee', 'f') | ('f', 'a', 'c', 'e') | $space};"
	samples := []srcExprSample{
		{"fce baabbef baa", "f a c e _ b aa b ee f _ b aa"},
	}

	prevTokenText := ""
	ths := TokenHooks{
		"char": func (t *lexer.Token, pc *ParseContext) (bool, error) {
			f := t.Text() != prevTokenText // x x -> x
			prevTokenText = t.Text()
			return f, nil
		},
		AnyToken: func (t *lexer.Token, pc *ParseContext) (bool, error) {
			return false, pc.EmitToken(lexer.NewToken(0, "space", "_", nil)) // " " -> _
		},
	}
	lhs := TokenHooks{
		"e": func (t *lexer.Token, pc *ParseContext) (bool, error) {
			if prevTokenText != "b" {
				return true, nil
			}

			return false, pc.EmitToken(lexer.NewToken(9, "char", "ee", nil)) // e -> ee
		},
		"c": func (t *lexer.Token, pc *ParseContext) (bool, error) {
			return true, pc.EmitToken(lexer.NewToken(2, "char", "a", nil)) // c -> a c
		},
	}

	testGrammarSamplesWithHooks(t, name, grammar, samples, ths, lhs)
}

func TestEofHooks (t *testing.T) {
	name := "EoF hooks"
	grammar := "!aside $space $indent; !extern $begin $end; " +
		"$indent = /(?:\\n|^)\\t+/; $space = /[ \\t]+/; $name = /\\w+/; " +
		"g = {$name | block}; block = $begin, {$name | block}, $end;"
	samples := []srcExprSample{
		{
			"foo\n\tbar baz\n\t\tqux",
			"foo (block { bar baz (block { qux } ) } )",
		},
		{
			"\tfoo\n\t\t\tbar\n\t\tbaz",
			"(block { foo (block { (block { bar } ) baz } ) } )",
		},
	}

	prevIndent := 0
	hooks := TokenHooks{
		"indent": func (t *lexer.Token, pc *ParseContext) (bool, error) {
			text := t.Text()
			indent := len(text)
			if text[0] == '\n' {
				indent--
			}
			var e error
			for indent > prevIndent {
				e = pc.EmitToken(lexer.NewToken(3, "begin", "{", nil))
				prevIndent++
			}
			for indent < prevIndent {
				e = pc.EmitToken(lexer.NewToken(4, "end", "}", nil))
				prevIndent--
			}
			return false, e
		},
		EofToken: func (t *lexer.Token, pc *ParseContext) (bool, error) {
			var e error
			for prevIndent > 0 {
				e = pc.EmitToken(lexer.NewToken(4, "end", "}", nil))
				prevIndent--
			}
			return false, e
		},
	}

	testGrammarSamplesWithHooks(t, name, grammar, samples, hooks, nil)
}

func TestResolveAnyTokenEof (t *testing.T) {
	name := "resolve * AnyToken * EoF"
	grammar := "$name = /\\w+/; $op = /[()+]/; g = sum | call; sum = $name, ['+', $name]; call = $name, '(', $name, ')';"
	samples := []srcErrSample{
		{"foo(", UnexpectedEofError},
	}
	testErrorSamples(t, name, grammar, samples)
}

func TestCaselessTokens (t *testing.T) {
	name := "caseless tokens"
	grammar := spaceDef + "$name = /\\w+/; !caseless $name; " +
		"g = {[key], $name}; key = 'FOO' | 'BAR';"
	samples := []srcExprSample{
		{"foo BAR BAZ Bar foo FOO qux", "(key foo) BAR BAZ (key Bar) foo (key FOO) qux"},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestTrailingAsides (t *testing.T) {
	name := "(non)trailing aside tokens"
	grammar := "!aside $space; $space = /-/; $char = /[a-z]/; $digit = /\\d/; $op = /\\[|\\]/; " +
		"g = {ch | di | bl}; ch = $char, [$digit]; di = $digit; bl = '[', {ch | di | bl}, ']';"
	samples := []srcExprSample{
		{"--a--1--", "- - (ch a - - 1) - -"},
		{"--a--b--", "- - (ch a) - - (ch b) - -"},
		{"-[-a-1-[-b-]-]-", "- (bl [ - (ch a - 1) - (bl [ - (ch b) - ] ) - ] ) -"},
	}
	testGrammarSamples(t, name, grammar, samples, true)
}

func TestIncludeSource (t *testing.T) {
	name := "inserting sources"
	grammar := "$char = /[a-z]/; $digit = /\\d/; $inc = /#/; " +
		"g = {foo | bar | $digit}; inc = '#', $digit; foo = $char, [inc], $char; bar = $char, [inc], $digit;"
	sources := []string{
		"a1",
		"34",
	}
	samples := []srcExprSample{
		{"foo#9bar", "(foo f o) (foo o (inc # 9) b) (foo a r)"},
		{"a#0", "(foo a (inc # 0) a) 1"},
		{"a#13", "(bar a (inc # 1) 3) 4 3"},
	}

	expectIncludeIndex := false
	tokenHooks := TokenHooks{
		"inc": func (token *lexer.Token, pc *ParseContext) (emit bool, e error) {
			expectIncludeIndex = true
			return true, nil
		},
		"digit": func (token *lexer.Token, pc *ParseContext) (emit bool, e error) {
			if !expectIncludeIndex {
				return true, nil
			}

			expectIncludeIndex = false
			t := token.Text()
			index, _ := strconv.Atoi(t)
			if index < len(sources) {
				return true, pc.IncludeSource(source.New(t, []byte(sources[index])))
			} else {
				return true, pc.IncludeSource(nil)
			}
		},
		AnyToken: func (token *lexer.Token, pc *ParseContext) (emit bool, e error) {
			expectIncludeIndex = false
			return true, nil
		},
	}
	testGrammarSamplesWithHooks(t, name, grammar, samples, tokenHooks, nil)
}

type includeHook struct {
	pc *ParseContext
	sources []string
	index int
}

func (ih *includeHook) NewNonTerm (nonTerm string, token *Token) error {
	return nil
}

func (ih *includeHook) HandleNonTerm (nonTerm string, result interface{}) error {
	return nil
}

func (ih *includeHook) HandleToken (token *Token) error {
	if token.TypeName() == "digit" {
		ih.index, _ = strconv.Atoi(token.Text())
	}
	return nil
}

func (ih *includeHook) EndNonTerm () (result interface{}, e error) {
	if ih.index < len(ih.sources) {
		e = ih.pc.IncludeSource(source.New("inc", []byte(ih.sources[ih.index])))
	}
	return
}

func TestIncludeSourceError (t *testing.T) {
	name := "inserting source error"
	grammar := "$char = /[a-z]/; $digit = /\\d/; $inc = /#/; " +
		"g = {foo | bar}; inc = '#', $digit; foo = $char, [inc], $char; bar = $char, [inc], $digit;"
	sources := []string{
		"cd",
	}
	samples := []srcErrSample{
		{"a#0b", IncludeUnresolvedError},
	}
	hs := &Hooks{
		NonTerms: NonTermHooks{
			"inc": func (nonTerm string, t *lexer.Token, pc *ParseContext) (NonTermHookInstance, error) {
				return &includeHook{pc, sources, 0}, nil
			},
		},
	}
	testErrorSamplesWithHooks(t, name, grammar, samples, hs)
}

func TestReservedLiterals (t *testing.T) {
	g0 := "!aside $space; $space = /\\s+/; $name = /\\w+/; g = 'var', $name;"
	g1 := "!reserved 'var'; " + g0
	src := "var var"
	expected := "var var"
	testGrammarSamples(t, "correct", g0, []srcExprSample{{src, expected}}, false)
	testErrorSamples(t, "reserved", g1, []srcErrSample{{src, UnexpectedTokenError}})
}

func TestBypass (t *testing.T) {
	toks := "$w =/\\w/; "
	samples := []struct{
		grammar string
		correct, wrong []string
	}{
		{toks + "g = {['a'], 'b'};", []string{"", "abb", "bab"}, []string{"a", "aba", "aab"}},
		{toks + "g = {{'a'}, 'b'};", []string{"", "bb", "aabb"}, []string{"aba"}},
		{toks + "g = {'a', ['b']};", []string{"", "aa", "aba"}, []string{"abb"}},
		{toks + "g = {'a', {'b'}};", []string{"", "aa", "abba"}, []string{"b"}},
		{toks + "g = [['a'], 'b'];", []string{"", "ab", "b"}, []string{"a", "bb"}},

		{toks + "g = [{'a'}, 'b'];", []string{"", "b", "aab"}, []string{"abab", "abb"}},
		{toks + "g = ['a', ['b']];", []string{"", "a", "ab"}, []string{"aa", "aba"}},
		{toks + "g = ['a', {'b'}];", []string{"", "a", "ab", "abb"}, []string{"aba"}},
		{toks + "g = {['a'], ['b'], 'c'}, ['d'];", []string{"abc", "acc", "bcd", "cc", "d"}, []string{"ca", "ad", "bd"}},
		{toks + "g = {'a', 'b'}, 'a', 'c';", []string{"ac", "abac", "ababac"}, []string{"abc"}},

		{toks + "g = {['a'], 'b'}, 'a', 'c';", []string{"ac", "bbac", "abbac"}, []string{"aac", "abc"}},
	}

	for i, s := range samples {
		g, e := langdef.ParseString("", s.grammar)
		if e != nil {
			t.Fatalf("sample #%d: unexpected grammar error: %s", i, e)
		}

		p := New(g)
		for j, src := range s.correct {
			q := source.NewQueue().Append(source.New("", []byte(src)))
			_, e = p.Parse(q, nil)
			if e != nil {
				t.Errorf("sample #%d, correct example #%d: unexpected error: %s", i, j, e)
			} else if !q.Eof() {
				t.Errorf("sample #%d, correct example #%d: input left: %q", i, j, q.Source().Content())
			}
		}
		for j, src := range s.wrong {
			q := source.NewQueue().Append(source.New("", []byte(src)))
			_, e = p.Parse(q, nil)
			if e == nil && q.IsEmpty() {
				t.Errorf("sample #%d, wrong example #%d: expecting error or input left, got success", i, j)
			}
		}
	}
}

type nthi struct {
	nt string
	result *[]string
}

func (hi nthi) NewNonTerm (nonTerm string, token *Token) error {
	*hi.result = append(*hi.result, hi.nt + "^" + nonTerm + token.Text())
	return nil
}

func (hi nthi) HandleNonTerm (nonTerm string, result interface{}) error {
	*hi.result = append(*hi.result, hi.nt + "$" + nonTerm + result.(string))
	return nil
}

func (hi nthi) HandleToken (token *Token) error {
	*hi.result = append(*hi.result, hi.nt + ":" + token.Text())
	return nil
}

func (hi nthi) EndNonTerm () (result interface{}, e error) {
	*hi.result = append(*hi.result, hi.nt + ".")
	return hi.nt, nil
}

func TestNonTermHooks (t *testing.T) {
	result := make([]string, 0)
	hs := Hooks{
		NonTerms: NonTermHooks{
			AnyNonTerm: func (nonTerm string, token *Token, pc *ParseContext) (NonTermHookInstance, error) {
				return &nthi{nonTerm, &result}, nil
			},
		},
	}

	grammar := "!aside $sp; $sp = /\\s+/; $name = /\\w+/; $op = /[+*]/; " +
		"s = p, {'+', p}; p = $name, {'*', $name};"

	g, e := langdef.ParseString("", grammar)
	if e != nil {
		t.Fatal("unexpected grammar error: " + e.Error())
	}

	src := "a + b * c"
	expected := "s^pa p:a p. s$pp s:+ s^pb p:b p:* p:c p. s$pp s."

	p := New(g)
	_, e = p.ParseString("", src, &hs)
	if e != nil {
		t.Fatal("unexpected error: " + e.Error())
	}

	got := strings.Join(result, " ")
	if got != expected {
		t.Errorf("expecting %q, got %q", expected, got)
	}
}
