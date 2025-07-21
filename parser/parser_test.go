package parser_test

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
	"github.com/ava12/llx/parser"
	pt "github.com/ava12/llx/parser/test"
	"github.com/ava12/llx/source"
)

type srcExprSample struct {
	src, expr string
}

type srcErrSample struct {
	src string
	err int
}

func testGrammarSamplesWithHooks(t *testing.T, name, grammar string, samples []srcExprSample,
	ths, lhs parser.TokenHooks, opts ...parser.ParseOption) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	p, _ := parser.New(g)
	for i, sample := range samples {
		n, e := pt.ParseAsTestNode(context.Background(), p, sample.src, ths, lhs, opts...)
		if e != nil {
			t.Errorf("grammar %q, sample #%d: got error: %s", name, i, e.Error())
			continue
		}

		e = pt.NewTreeValidator(n, sample.expr).Validate()
		if e != nil {
			t.Errorf("grammar %q, sample #%d: validation error: %s", name, i, e.Error())
		}
	}
}

func testGrammarSamples(t *testing.T, name, grammar string, samples []srcExprSample, opts ...parser.ParseOption) {
	var hs parser.TokenHooks

	testGrammarSamplesWithHooks(t, name, grammar, samples, hs, nil, opts...)
}

func testErrorSamplesWithHooks(t *testing.T, name, grammar string, samples []srcErrSample, hs parser.Hooks) {
	g, e := langdef.ParseString(name, grammar)
	if e != nil {
		t.Errorf("grammar %q: got error: %s", name, e.Error())
		return
	}

	parser, _ := parser.New(g)

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
	testErrorSamplesWithHooks(t, name, grammar, samples, parser.Hooks{})
}

const spaceDef = "!aside $space; $space = /\\s+/; "

func TestErrors(t *testing.T) {
	name := "errors"
	grammar := spaceDef + "$name = /\\w+/; $op = /[()]/; s = 'foo' | 'bar', '(', 'bar' | 'baz', ')';"
	samples := []srcErrSample{
		{"foo(bar", parser.UnexpectedEoiError},
		{"foo(bar baz", parser.UnexpectedTokenError},
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

	p, _ := parser.New(g)

	samples := []struct {
		hooks parser.Hooks
		err   int
	}{
		{parser.Hooks{parser.TokenHooks{"space": nil}, nil, nil}, parser.UnknownTokenTypeError},
		{parser.Hooks{nil, nil, parser.NodeHooks{"foo": nil}}, parser.UnknownNodeError},
	}

	for i, sample := range samples {
		_, e := p.Parse(context.Background(), queue, sample.hooks)
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

func TestRemainingSourceError(t *testing.T) {
	grammar := `!aside $space; $space = /\s+/; $name = /\w+/; g = {b}; b = 'do', {s}, 'done'; s = $name | b;`
	samples := []struct {
		src   string
		isErr bool
	}{
		{"do foo done", false},
		{"do foo done\n  ", false},
		{"do foo done bar", true},
	}
	g, e := langdef.ParseString("", grammar)
	test.ExpectNoError(t, e)

	p, e := parser.New(g)
	test.ExpectNoError(t, e)

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func(t *testing.T) {
			_, e = p.ParseString(context.Background(), "", sample.src, parser.Hooks{}, parser.WithFullSource())
			if sample.isErr {
				test.ExpectErrorCode(t, parser.RemainingSourceError, e)
			} else {
				test.ExpectNoError(t, e)
			}
		})
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
	testGrammarSamples(t, name, grammar, samples)
}

func TestAside(t *testing.T) {
	name := "aside"
	grammar := "!aside $sep; $sep = /-/; $char = /\\w/; s = {'a' | 'b' | 'c'};"
	samples := []srcExprSample{
		{"abc", "a b c"},
		{"a-a-a", "a - a - a"},
		{"--b--c--", "- - b - - c - -"},
	}
	testGrammarSamples(t, name, grammar, samples, parser.WithAsides())
}

func TestAri(t *testing.T) {
	name := "ari"
	grammar := spaceDef + "$num=/-?\\d+/; $op=/[()^*\\/+-]/; $minus=/-/; !group $minus; " +
		"ari=sum; sum=pro,{('+'|'-'),pro}; pro=pow,{('*'|'/'),pow}; pow=val,{'^',val}; val=$num;"
	samples := []srcExprSample{
		/*{
			"2 + 2",
			"(sum (pro (pow (val 2))) + (pro (pow (val 2))))",
		},
		{
			"2 + 3^4*5",
			"(sum (pro (pow (val 2))) + (pro (pow (val 3) ^ (val 4)) * (pow (val 5))))",
		},*/
		{
			"-2-3*-1",
			"(sum (pro (pow (val -2))) - (pro (pow (val 3)) * (pow (val -1))))",
		},
	}
	g, _ := langdef.ParseString("ari", grammar)
	fmt.Printf("%v\r\n%v\r\n%v\r\n", g.States, g.Rules, g.Nodes)
	testGrammarSamples(t, name, grammar, samples)
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
	testGrammarSamples(t, name, grammar, samples)
}

func TestTokenHooks(t *testing.T) {
	name := "token hooks"
	grammar := "$space = /\\s+/; $char = /[bcdf]|aa?|ee?/; !literal 'a' 'b' 'c' 'd' 'e' 'f' 'aa' 'ee';" +
		"g = {('b', 'aa') | ('b', 'ee', 'f') | ('f', 'a', 'c', 'e') | $space};"
	samples := []srcExprSample{
		{"fce baabbef baa", "f a c e _ b aa b ee f _ b aa"},
	}

	prevTokenText := ""
	ths := parser.TokenHooks{
		"char": func(_ context.Context, t *lexer.Token, _ *parser.TokenContext) (bool, []*parser.Token, error) {
			f := t.Text() != prevTokenText // x x -> x
			prevTokenText = t.Text()
			return f, nil, nil
		},
		parser.AnyToken: func(_ context.Context, t *lexer.Token, _ *parser.TokenContext) (bool, []*parser.Token, error) {
			return false, []*parser.Token{lexer.NewToken(0, "space", []byte{'_'}, source.Pos{})}, nil // " " -> _
		},
	}
	lhs := parser.TokenHooks{
		"e": func(_ context.Context, t *lexer.Token, _ *parser.TokenContext) (bool, []*parser.Token, error) {
			if prevTokenText != "b" {
				return true, nil, nil
			}

			return false, []*parser.Token{lexer.NewToken(1, "char", []byte("ee"), source.Pos{})}, nil // e -> ee
		},
		"c": func(_ context.Context, t *lexer.Token, _ *parser.TokenContext) (bool, []*parser.Token, error) {
			extra := []*parser.Token{
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
	hooks := parser.TokenHooks{
		"indent": func(_ context.Context, t *lexer.Token, _ *parser.TokenContext) (bool, []*parser.Token, error) {
			text := t.Text()
			indent := len(text)
			if text[0] == '\n' {
				indent--
			}
			var e error
			var extra []*parser.Token
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
		parser.EofToken: func(_ context.Context, t *lexer.Token, _ *parser.TokenContext) (bool, []*parser.Token, error) {
			var e error
			var extra []*parser.Token
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
		{"foo(", parser.UnexpectedEoiError},
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
	testGrammarSamples(t, name, grammar, samples)
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
	testGrammarSamples(t, name, grammar, samples, parser.WithAsides())
}

func TestReservedLiterals(t *testing.T) {
	g0 := "!aside $space; $space = /\\s+/; $name = /\\w+/; g = 'var', $name;"
	g1 := "!reserved 'var'; " + g0
	src := "var var"
	expected := "var var"
	testGrammarSamples(t, "correct", g0, []srcExprSample{{src, expected}})
	testErrorSamples(t, "reserved", g1, []srcErrSample{{src, parser.UnexpectedTokenError}})
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

		p, _ := parser.New(g)
		for j, src := range s.correct {
			q := source.NewQueue().Append(source.New("", []byte(src)))
			_, e = p.Parse(context.Background(), q, parser.Hooks{})
			if e != nil {
				t.Errorf("sample #%d, correct example #%d: unexpected error: %s", i, j, e)
			} else if !q.Eof() {
				t.Errorf("sample #%d, correct example #%d: input left: %q", i, j, q.Source().Content())
			}
		}
		for j, src := range s.wrong {
			q := source.NewQueue().Append(source.New("", []byte(src)))
			_, e = p.Parse(context.Background(), q, parser.Hooks{})
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

func (hi nthi) NewNode(node string, token *parser.Token) error {
	*hi.result = append(*hi.result, hi.nt+"^"+node+token.Text())
	return nil
}

func (hi nthi) HandleNode(node string, result any) error {
	*hi.result = append(*hi.result, hi.nt+"$"+node+result.(string))
	return nil
}

func (hi nthi) HandleToken(token *parser.Token) error {
	*hi.result = append(*hi.result, hi.nt+":"+token.Text())
	return nil
}

func (hi nthi) EndNode() (result any, e error) {
	*hi.result = append(*hi.result, hi.nt+".")
	return hi.nt, nil
}

func TestNodeHooks(t *testing.T) {
	result := make([]string, 0)
	hs := parser.Hooks{
		Nodes: parser.NodeHooks{
			parser.AnyNode: func(_ context.Context, node string, token *parser.Token, _ *parser.NodeContext) (parser.NodeHookInstance, error) {
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

	p, _ := parser.New(g)
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

	p, e := parser.New(g)
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	src := "foo bar baz"
	tokenHook := func(_ context.Context, tok *lexer.Token, _ *parser.TokenContext) (bool, []*parser.Token, error) {
		time.Sleep(time.Second)
		return true, nil, nil
	}
	hooks := parser.Hooks{
		Tokens: parser.TokenHooks{
			parser.AnyToken: tokenHook,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, e = p.ParseString(ctx, "", src, hooks)
	if e == nil || !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("expecting DeadlineExceeded, got %v", e)
	}
}

type testLayer parser.Hooks

func (l testLayer) Init(context.Context, *parser.ParseContext) parser.Hooks {
	return parser.Hooks(l)
}

func (l testLayer) Setup(commands []grammar.LayerCommand, p *parser.Parser) (parser.HookLayer, error) {
	return l, nil
}

type testReplaceLayerEntry struct {
	extra []string
	emit  bool
}

type testReplaceLayerTemplate struct {
	override map[string]testReplaceLayerEntry
}

func (t testReplaceLayerTemplate) Setup(commands []grammar.LayerCommand, p *parser.Parser) (parser.HookLayer, error) {
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

	handler := func(_ context.Context, tok *parser.Token, _ *parser.TokenContext) (emit bool, extra []*parser.Token, e error) {
		entry, found := replaces[tok.Text()]
		if !found {
			entry, found = replaces[""]
		}
		if !found {
			return true, nil, nil
		}

		emit = entry.emit
		extra = make([]*parser.Token, 0, len(entry.extra))
		var newTok *parser.Token
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
		Tokens: parser.TokenHooks{},
	}

	if len(tokenTypes) == 0 {
		tokenTypes = append(tokenTypes, parser.AnyToken)
	}
	for _, tokenType := range tokenTypes {
		result.Tokens[tokenType] = handler
	}

	return result, nil
}

func TestLayerRegister(t *testing.T) {
	grammar := "$char = /\\w/; g = {$char}; @test-setup _('', '', x);"
	templates := map[string]parser.HookLayerTemplate{
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

	_, e = parser.New(g)
	test.ExpectErrorCode(t, parser.UnknownLayerError, e)

	_, e = parser.New(g, parser.WithLayerTemplates(templates))
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	e = parser.RegisterHookLayer("test-setup", testReplaceLayerTemplate{})
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}
	e = parser.RegisterHookLayer("test-setup", testReplaceLayerTemplate{})
	test.ExpectErrorCode(t, parser.LayerRegisteredError, e)

	_, e = parser.New(g)
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	p, e := parser.New(g, parser.WithLayerTemplates(templates))
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	node, e := pt.ParseAsTestNode(context.Background(), p, src, nil, nil)
	if e == nil {
		e = pt.NewTreeValidator(node, expected).Validate()
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
	templates := map[string]parser.HookLayerTemplate{
		"replace": testReplaceLayerTemplate{},
	}

	var (
		p *parser.Parser
		n *pt.TreeNode
	)
	g, e := langdef.ParseString("", grammar)
	if e == nil {
		p, e = parser.New(g, parser.WithLayerTemplates(templates))
	}
	if e == nil {
		n, e = pt.ParseAsTestNode(context.Background(), p, src, nil, nil)
	}
	if e == nil {
		e = pt.NewTreeValidator(n, expected).Validate()
	}
	if e != nil {
		t.Fatalf("got unexpected error: %s", e.Error())
	}
}

type testLayerRecordTemplate struct {
	result *pt.TreeNode
}

func (lt *testLayerRecordTemplate) Setup(commands []grammar.LayerCommand, p *parser.Parser) (parser.HookLayer, error) {
	return testLayer{
		Nodes: parser.NodeHooks{
			parser.AnyNode: func(ctx context.Context, node string, t *lexer.Token, _ *parser.NodeContext) (parser.NodeHookInstance, error) {
				result := pt.NodeNode(node)
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
	templates := map[string]parser.HookLayerTemplate{
		"foo": &fooLt,
		"bar": &barLt,
	}

	var p *parser.Parser
	var n *pt.TreeNode
	g, e := langdef.ParseString("", grammar)
	if e == nil {
		p, e = parser.New(g, parser.WithLayerTemplates(templates))
	}
	if e == nil {
		n, e = pt.ParseAsTestNode(context.Background(), p, src, nil, nil)
	}
	if e != nil {
		t.Fatalf("unexpected error: %s", e)
	}

	results := []*pt.TreeNode{n, fooLt.result, barLt.result}
	for i, result := range results {
		e = pt.NewTreeValidator(result, expected).Validate()
		if e != nil {
			t.Fatalf("result #%d: error: %s", i, e)
		}
	}
}

func TestUserLiterals(t *testing.T) {
	grammar := `$char = /\w/; g = {$char};`
	hooks := parser.TokenHooks{
		"f": func(_ context.Context, _ *lexer.Token, tc *parser.TokenContext) (bool, []*parser.Token, error) {
			newToken, _ := tc.ParseContext().Parser().MakeToken("char", []byte("b"))
			return false, []*parser.Token{newToken}, nil
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
	templates := map[string]parser.HookLayerTemplate{
		"replace": testReplaceLayerTemplate{},
	}

	var (
		p *parser.Parser
		n *pt.TreeNode
	)
	g, e := langdef.ParseString("", grammar)
	if e == nil {
		p, e = parser.New(g, parser.WithLayerTemplates(templates))
	}
	if e == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		n, e = pt.ParseAsTestNode(ctx, p, src, nil, nil)
	}
	if e == nil {
		e = pt.NewTreeValidator(n, expected).Validate()
	}
	if e != nil {
		t.Fatalf("got unexpected error: %s", e.Error())
	}
}

func TestPeekToken(t *testing.T) {
	grammar := `$char = /./; g = {$char};
	@replace replace('c', 'c', 'c', 'c');
	@replace replace('e', 'i');
	@tmp;
	`
	samples := []struct {
		src, expected string
	}{
		{"con", "cccon"},
		{"ciao", "chchchiao"},
		{"dagger", "daghghir"},
	}

	tmpLayer := testLayer{
		Tokens: parser.TokenHooks{
			"char": func(_ context.Context, token *parser.Token, tc *parser.TokenContext) (bool, []*parser.Token, error) {
				if token.Text() != "c" && token.Text() != "g" {
					return true, nil, nil
				}

				triggered := false
			loop:
				for {
					tok, e := tc.PeekToken()
					if e != nil || tok == nil || tok.Type() < 0 {
						return true, nil, nil
					}

					switch tok.Text()[0] {
					case 'a', 'o', 'u':
						break loop
					case 'e', 'i':
						triggered = true
						break loop
					}
				}

				var extra []*parser.Token
				if triggered {
					tok, _ := tc.ParseContext().Parser().MakeToken("char", []byte("h"))
					extra = append(extra, tok)
				}
				return true, extra, nil
			},
		},
	}

	g, e := langdef.ParseString("", grammar)
	test.ExpectNoError(t, e)
	p, e := parser.New(g, parser.WithLayerTemplates(map[string]parser.HookLayerTemplate{
		"replace": testReplaceLayerTemplate{},
		"tmp":     tmpLayer,
	}))
	test.ExpectNoError(t, e)

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func(t *testing.T) {
			var got []string
			hs := parser.Hooks{
				Tokens: parser.TokenHooks{
					parser.AnyToken: func(_ context.Context, token *parser.Token, tc *parser.TokenContext) (bool, []*parser.Token, error) {
						got = append(got, token.Text())
						return true, nil, nil
					},
				},
			}
			_, e := p.ParseString(context.Background(), "", sample.src, hs)
			test.ExpectNoError(t, e)
			test.ExpectString(t, sample.expected, strings.Join(got, ""))
		})
	}
}

func TestEofToken(t *testing.T) {
	grammar := `$nl = /\n/; $name = /\S+/; g = {$nl | ($name, $nl | $)};`
	samples := []srcExprSample{
		{"foo\nbar", "foo nl bar"},
		{"foo\nbar\n", "foo nl bar nl"},
	}
	testGrammarSamples(t, "eof token", grammar, samples)
}
