package parser

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ava12/llx"
	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/internal/test"
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
	AnyToken: func(_ context.Context, token *lexer.Token, pc *ParseContext) (bool, []*Token, error) {
		return true, nil, nil
	},
}

func testGrammarSamplesWithHooks(t *testing.T, name, grammar string, samples []srcExprSample, ths, lhs TokenHooks) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	p, _ := New(g)
	for i, sample := range samples {
		n, e := parseAsTestNode(context.Background(), p, sample.src, ths, lhs)
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

func testGrammarSamples(t *testing.T, name, grammar string, samples []srcExprSample, captureAside bool) {
	var hs TokenHooks

	if captureAside {
		hs = testTokenHooks
	}
	testGrammarSamplesWithHooks(t, name, grammar, samples, hs, nil)
}

func testErrorSamplesWithHooks(t *testing.T, name, grammar string, samples []srcErrSample, hs Hooks) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	parser, _ := New(g)

	for i, sample := range samples {
		q := source.NewQueue().Append(source.New("sample", []byte(sample.src)))
		_, e := parser.Parse(context.Background(), q, hs)
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

func testErrorSamples(t *testing.T, name, grammar string, samples []srcErrSample) {
	testErrorSamplesWithHooks(t, name, grammar, samples, Hooks{})
}

const spaceDef = "!aside $space; $space = /\\s+/; "

func TestErrors(t *testing.T) {
	name := "errors"
	grammar := spaceDef + "$name = /\\w+/; $op = /[()]/; s = 'foo' | 'bar', '(', 'bar' | 'baz', ')';"
	samples := []srcErrSample{
		{"foo(bar", UnexpectedEoiError},
		{"foo(bar baz", UnexpectedTokenError},
	}
	testErrorSamples(t, name, grammar, samples)
}

func TestHandlerKeyErrors(t *testing.T) {
	name := "handler key errors"
	grammar := "$any = /./; g = $any;"
	queue := source.NewQueue().Append(source.New("x", []byte("x")))
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	parser, _ := New(g)

	samples := []struct {
		hooks Hooks
		err   int
	}{
		{Hooks{TokenHooks{"space": nil}, nil, nil}, UnknownTokenTypeError},
		{Hooks{nil, nil, NodeHooks{"foo": nil}}, UnknownNodeError},
	}

	for i, sample := range samples {
		_, e := parser.Parse(context.Background(), queue, sample.hooks)
		var (
			ee   *llx.Error
			code int
			f    bool
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

func TestSimple(t *testing.T) {
	name := "simple"
	grammar := "$char = /\\w/; s = {a | b | c}; a = 'a',{'a'}; b = 'b', ['b']; c = 'c', {a | b | c};"
	samples := []srcExprSample{
		{"aaa", "(a a a a)"},
		{"bbb", "(b b b)(b b)"},
		{"cabcaa", "(c c (a a) (b b) (c c (a a a)))"},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestAside(t *testing.T) {
	name := "aside"
	grammar := "!aside $sep; $sep = /-/; $char = /\\w/; s = {'a' | 'b' | 'c'};"
	samples := []srcExprSample{
		{"abc", "a b c"},
		{"a-a-a", "a - a - a"},
		{"--b--c--", "- - b - - c - -"},
	}
	testGrammarSamples(t, name, grammar, samples, true)
}

func TestAri(t *testing.T) {
	name := "ari"
	grammar := spaceDef + "$num=/-?\\d+/; $op=/[()^*\\/+-]/; $minus=/-/; !group $minus; ari=sum; sum=pro,{('+'|'-'),pro}; pro=pow,{('*'|'/'),pow}; pow=val,{'^',val}; val=$num;"
	samples := []srcExprSample{
		{
			"2 + 2",
			"(sum (pro (pow (val 2))) + (pro (pow (val 2))))",
		},
		{
			"2 + 3^4*5",
			"(sum (pro (pow (val 2))) + (pro (pow (val 3) ^ (val 4)) * (pow (val 5))))",
		},
		{
			"-2-3*-1",
			"(sum (pro (pow (val -2))) - (pro (pow (val 3)) * (pow (val -1))))",
		},
	}
	g, _ := langdef.ParseString("ari", grammar)
	fmt.Printf("%v\r\n%v\r\n%v\r\n", g.States, g.Rules, g.Nodes)
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestMultiRuleAri(t *testing.T) {
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

func TestTokenHooks(t *testing.T) {
	name := "token hooks"
	grammar := "$space = /\\s+/; $char = /[bcdf]|aa?|ee?/; !literal 'a' 'b' 'c' 'd' 'e' 'f' 'aa' 'ee';" +
		"g = {('b', 'aa') | ('b', 'ee', 'f') | ('f', 'a', 'c', 'e') | $space};"
	samples := []srcExprSample{
		{"fce baabbef baa", "f a c e _ b aa b ee f _ b aa"},
	}

	prevTokenText := ""
	ths := TokenHooks{
		"char": func(_ context.Context, t *lexer.Token, pc *ParseContext) (bool, []*Token, error) {
			f := t.Text() != prevTokenText // x x -> x
			prevTokenText = t.Text()
			return f, nil, nil
		},
		AnyToken: func(_ context.Context, t *lexer.Token, pc *ParseContext) (bool, []*Token, error) {
			return false, []*Token{lexer.NewToken(0, "space", []byte{'_'}, source.Pos{})}, nil // " " -> _
		},
	}
	lhs := TokenHooks{
		"e": func(_ context.Context, t *lexer.Token, pc *ParseContext) (bool, []*Token, error) {
			if prevTokenText != "b" {
				return true, nil, nil
			}

			return false, []*Token{lexer.NewToken(1, "char", []byte("ee"), source.Pos{})}, nil // e -> ee
		},
		"c": func(_ context.Context, t *lexer.Token, pc *ParseContext) (bool, []*Token, error) {
			extra := []*Token{
				lexer.NewToken(1, "char", []byte{'a'}, source.Pos{}),
				t,
			}
			return false, extra, nil // c -> a c
		},
	}

	testGrammarSamplesWithHooks(t, name, grammar, samples, ths, lhs)
}

func TestEofHooks(t *testing.T) {
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
		"indent": func(_ context.Context, t *lexer.Token, pc *ParseContext) (bool, []*Token, error) {
			text := t.Text()
			indent := len(text)
			if text[0] == '\n' {
				indent--
			}
			var e error
			var extra []*Token
			for indent > prevIndent {
				extra = append(extra, lexer.NewToken(3, "begin", []byte{'{'}, source.Pos{}))
				prevIndent++
			}
			for indent < prevIndent {
				extra = append(extra, lexer.NewToken(4, "end", []byte{'}'}, source.Pos{}))
				prevIndent--
			}
			return false, extra, e
		},
		EofToken: func(_ context.Context, t *lexer.Token, pc *ParseContext) (bool, []*Token, error) {
			var e error
			var extra []*Token
			for prevIndent > 0 {
				extra = append(extra, lexer.NewToken(4, "end", []byte{'}'}, source.Pos{}))
				prevIndent--
			}
			return false, extra, e
		},
	}

	testGrammarSamplesWithHooks(t, name, grammar, samples, hooks, nil)
}

func TestResolveAnyTokenEof(t *testing.T) {
	name := "resolve * AnyToken * EoF"
	grammar := "$name = /\\w+/; $op = /[()+]/; g = sum | call; sum = $name, ['+', $name]; call = $name, '(', $name, ')';"
	samples := []srcErrSample{
		{"foo(", UnexpectedEoiError},
	}
	testErrorSamples(t, name, grammar, samples)
}

func TestCaselessTokens(t *testing.T) {
	name := "caseless tokens"
	grammar := spaceDef + "$name = /\\w+/; !caseless $name; " +
		"g = {[key], $name}; key = 'FOO' | 'BAR';"
	samples := []srcExprSample{
		{"foo BAR BAZ Bar foo FOO qux", "(key foo) BAR BAZ (key Bar) foo (key FOO) qux"},
	}
	testGrammarSamples(t, name, grammar, samples, false)
}

func TestTrailingAsides(t *testing.T) {
	name := "(non)trailing aside tokens"
	grammar := "!aside $space; $space = /-/; $char = /[a-z]/; $digit = /\\d/; $op = /\\[|\\]/; " +
		"g = {ch | di | bl}; ch = $char, [$digit]; di = $digit; bl = '[', {ch | di | bl}, ']';"
	samples := []srcExprSample{
		//{"--a--1--", "- - (ch a - - 1) - -"},
		{"--a--b--", "- - (ch a) - - (ch b) - -"},
		//{"-[-a-1-[-b-]-]-", "- (bl [ - (ch a - 1) - (bl [ - (ch b) - ] ) - ] ) -"},
	}
	testGrammarSamples(t, name, grammar, samples, true)
}

func TestReservedLiterals(t *testing.T) {
	g0 := "!aside $space; $space = /\\s+/; $name = /\\w+/; g = 'var', $name;"
	g1 := "!reserved 'var'; " + g0
	src := "var var"
	expected := "var var"
	testGrammarSamples(t, "correct", g0, []srcExprSample{{src, expected}}, false)
	testErrorSamples(t, "reserved", g1, []srcErrSample{{src, UnexpectedTokenError}})
}

func TestBypass(t *testing.T) {
	toks := "$w =/\\w/; "
	samples := []struct {
		grammar        string
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

		p, _ := New(g)
		for j, src := range s.correct {
			q := source.NewQueue().Append(source.New("", []byte(src)))
			_, e = p.Parse(context.Background(), q, Hooks{})
			if e != nil {
				t.Errorf("sample #%d, correct example #%d: unexpected error: %s", i, j, e)
			} else if !q.Eof() {
				t.Errorf("sample #%d, correct example #%d: input left: %q", i, j, q.Source().Content())
			}
		}
		for j, src := range s.wrong {
			q := source.NewQueue().Append(source.New("", []byte(src)))
			_, e = p.Parse(context.Background(), q, Hooks{})
			if e == nil && q.IsEmpty() {
				t.Errorf("sample #%d, wrong example #%d: expecting error or input left, got success", i, j)
			}
		}
	}
}

type nthi struct {
	nt     string
	result *[]string
}

func (hi nthi) NewNode(node string, token *Token) error {
	*hi.result = append(*hi.result, hi.nt+"^"+node+token.Text())
	return nil
}

func (hi nthi) HandleNode(node string, result any) error {
	*hi.result = append(*hi.result, hi.nt+"$"+node+result.(string))
	return nil
}

func (hi nthi) HandleToken(token *Token) error {
	*hi.result = append(*hi.result, hi.nt+":"+token.Text())
	return nil
}

func (hi nthi) EndNode() (result any, e error) {
	*hi.result = append(*hi.result, hi.nt+".")
	return hi.nt, nil
}

func TestNodeHooks(t *testing.T) {
	result := make([]string, 0)
	hs := Hooks{
		Nodes: NodeHooks{
			AnyNode: func(_ context.Context, node string, token *Token, pc *ParseContext) (NodeHookInstance, error) {
				return &nthi{node, &result}, nil
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

	p, _ := New(g)
	_, e = p.ParseString(context.Background(), "", src, hs)
	if e != nil {
		t.Fatal("unexpected error: " + e.Error())
	}

	got := strings.Join(result, " ")
	if got != expected {
		t.Errorf("expecting %q, got %q", expected, got)
	}
}

func TestContext(t *testing.T) {
	grammar := "!aside $s; $s = /\\s+/; $n = /\\S+/; g = {$n};"
	g, e := langdef.ParseString("", grammar)
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	p, e := New(g)
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	src := "foo bar baz"
	tokenHook := func(_ context.Context, tok *lexer.Token, _ *ParseContext) (bool, []*Token, error) {
		time.Sleep(time.Second)
		return true, nil, nil
	}
	hooks := Hooks{
		Tokens: TokenHooks{
			AnyToken: tokenHook,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, e = p.ParseString(ctx, "", src, hooks)
	if e == nil || !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("expecting DeadlineExceeded, got %v", e)
	}
}

type testLayer Hooks

func (l testLayer) Init(context.Context, *ParseContext) (Hooks, error) {
	return Hooks(l), nil
}

type testReplaceLayerEntry struct {
	extra []string
	emit  bool
}

type testReplaceLayerTemplate struct {
	override map[string]testReplaceLayerEntry
}

func (t testReplaceLayerTemplate) Setup(commands []grammar.LayerCommand, p *Parser) (HookLayer, error) {
	replaces := make(map[string]testReplaceLayerEntry, len(commands)+len(t.override))
	var tokenTypes []string
	var emitTypeName string

	for _, command := range commands {
		switch command.Command {
		case "watch":
			tokenTypes = append(tokenTypes, command.Arguments...)
		case "emit":
			emitTypeName = command.Arguments[0]
		default:
			emit := len(command.Arguments) > 1 && command.Arguments[0] == command.Arguments[1]
			var offset int
			if emit {
				offset = 2
			} else {
				offset = 1
			}
			replaces[command.Arguments[0]] = testReplaceLayerEntry{command.Arguments[offset:], emit}
		}
	}
	for key, entry := range t.override {
		replaces[key] = entry
	}

	handler := func(_ context.Context, tok *Token, _ *ParseContext) (emit bool, extra []*Token, e error) {
		entry, found := replaces[tok.Text()]
		if !found {
			entry, found = replaces[""]
		}
		if !found {
			return true, nil, nil
		}

		emit = entry.emit
		extra = make([]*Token, 0, len(entry.extra))
		var newTok *Token
		typeName := emitTypeName
		if typeName == "" {
			typeName = tok.TypeName()
		}
		for _, text := range entry.extra {
			newTok, e = p.MakeToken(typeName, []byte(text))
			if e != nil {
				return
			}

			extra = append(extra, newTok)
		}
		return
	}

	result := testLayer{
		Tokens: TokenHooks{},
	}

	if len(tokenTypes) == 0 {
		tokenTypes = append(tokenTypes, AnyToken)
	}
	for _, tokenType := range tokenTypes {
		result.Tokens[tokenType] = handler
	}

	return result, nil
}

func TestLayerRegister(t *testing.T) {
	grammar := "$char = /\\w/; g = {$char}; @test-setup _('', '', x);"
	templates := map[string]HookLayerTemplate{
		"test-setup": testReplaceLayerTemplate{
			override: map[string]testReplaceLayerEntry{
				"a": {[]string{"z"}, true},
			},
		},
	}
	src := "abc"
	expected := "a z b x c x"

	g, e := langdef.ParseString("", grammar)
	if e != nil {
		t.Fatalf("got unexpected error: %s", e.Error())
	}

	_, e = New(g)
	test.ExpectErrorCode(t, UnknownLayerError, e)

	_, e = New(g, WithLayerTemplates(templates))
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	e = RegisterHookLayer("test-setup", testReplaceLayerTemplate{})
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}
	e = RegisterHookLayer("test-setup", testReplaceLayerTemplate{})
	test.ExpectErrorCode(t, LayerRegisteredError, e)

	_, e = New(g)
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	p, e := New(g, WithLayerTemplates(templates))
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	node, e := parseAsTestNode(context.Background(), p, src, nil, nil)
	if e == nil {
		e = newTreeValidator(node, expected).validate()
	}
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}
}

func TestLayerTokenHooks(t *testing.T) {
	grammar := `$char = /\w/; g = {$char}; 
		@replace _(a, a) _(b) _(c, c, c);
		@replace _(a, x) _(c, c, z);`
	src := "abc"
	expected := "x c z c z"
	templates := map[string]HookLayerTemplate{
		"replace": testReplaceLayerTemplate{},
	}

	var (
		p *Parser
		n *treeNode
	)
	g, e := langdef.ParseString("", grammar)
	if e == nil {
		p, e = New(g, WithLayerTemplates(templates))
	}
	if e == nil {
		n, e = parseAsTestNode(context.Background(), p, src, nil, nil)
	}
	if e == nil {
		e = newTreeValidator(n, expected).validate()
	}
	if e != nil {
		t.Fatalf("got unexpected error: %s", e.Error())
	}
}

type testLayerRecordTemplate struct {
	result *treeNode
}

func (lt *testLayerRecordTemplate) Setup(commands []grammar.LayerCommand, p *Parser) (HookLayer, error) {
	return testLayer{
		Nodes: NodeHooks{
			AnyNode: func(ctx context.Context, node string, t *lexer.Token, pc *ParseContext) (NodeHookInstance, error) {
				result := nodeNode(node)
				if lt.result == nil {
					lt.result = result
				}
				return result, nil
			},
		},
	}, nil
}

func TestLayerNodeHooks(t *testing.T) {
	grammar := `$char = /\w/; @foo; @bar;
		g = {syl}; syl = $char, {vowel}; vowel = 'a' | 'e' | 'i' | 'o' | 'u';`
	src := "winnie"
	expected := "(syl w (vowel i)) (syl n) (syl n (vowel i) (vowel e))"
	fooLt := testLayerRecordTemplate{}
	barLt := testLayerRecordTemplate{}
	templates := map[string]HookLayerTemplate{
		"foo": &fooLt,
		"bar": &barLt,
	}

	var p *Parser
	var n *treeNode
	g, e := langdef.ParseString("", grammar)
	if e == nil {
		p, e = New(g, WithLayerTemplates(templates))
	}
	if e == nil {
		n, e = parseAsTestNode(context.Background(), p, src, nil, nil)
	}
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	results := []*treeNode{n, fooLt.result, barLt.result}
	for i, result := range results {
		e = newTreeValidator(result, expected).validate()
		if e != nil {
			t.Fatalf("result #%d: error: %s", i, e)
		}
	}
}

func TestUserLiterals(t *testing.T) {
	grammar := `$char = /\w/; g = {$char};`
	hooks := TokenHooks{
		"f": func(_ context.Context, _ *lexer.Token, pc *ParseContext) (bool, []*Token, error) {
			newToken, _ := pc.Parser().MakeToken("char", []byte("b"))
			return false, []*Token{newToken}, nil
		},
	}
	samples := []srcExprSample{
		{"foo", "b o o"},
	}
	testGrammarSamplesWithHooks(t, "", grammar, samples, nil, hooks)
}

func TestEofEoiHandling(t *testing.T) {
	grammar := `$char = /\w/; g= {$char};
		@replace watch('-eof-') emit(char) replace('', eof);
		@replace watch('-eoi-') emit(char) replace('', eoi);
		@replace watch('-eof-', '-eoi-') emit(char) replace('', end);`
	src := "abc"
	expected := "a b c eof end eoi end"
	templates := map[string]HookLayerTemplate{
		"replace": testReplaceLayerTemplate{},
	}

	var (
		p *Parser
		n *treeNode
	)
	g, e := langdef.ParseString("", grammar)
	if e == nil {
		p, e = New(g, WithLayerTemplates(templates))
	}
	if e == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		n, e = parseAsTestNode(ctx, p, src, nil, nil)
	}
	if e == nil {
		e = newTreeValidator(n, expected).validate()
	}
	if e != nil {
		t.Fatalf("got unexpected error: %s", e.Error())
	}
}
