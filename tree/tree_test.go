package tree

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/source"
)

const treeGrammarDef = "!aside $space; $space = /\\s+/; " +
	"$name = /[a-z0-9]+/; $string = /\".*?\"|'.*?'/; $op = /[()]/; " +
	"tree-def = {$name | $string | nt}; nt = '(', $name, {$name | $string | nt}, ')';"

type treeDefHook struct {
	nt      *nodeElement
	gotName bool
}

func (h *treeDefHook) NewNode(node string, token *lexer.Token) error {
	return nil
}

func (h *treeDefHook) HandleNode(node string, result interface{}) error {
	h.nt.AddChild(result.(Element), nil)
	return nil
}

func (h *treeDefHook) HandleToken(token *lexer.Token) error {
	t := token.Text()
	if t != "(" && t != ")" {
		if h.gotName {
			h.nt.AddChild(NewTokenElement(token), nil)
		} else {
			h.nt.typeName = t
			h.gotName = true
		}
	}
	return nil
}

func (h *treeDefHook) EndNode() (result interface{}, e error) {
	return h.nt, nil
}

func newTreeDefHook(_ context.Context, node string, tok *lexer.Token, pc *parser.ParseContext) (parser.NodeHookInstance, error) {
	return &treeDefHook{nt: &nodeElement{token: tok}, gotName: (node == "tree-def")}, nil
}

var (
	treeHooks = parser.Hooks{
		Nodes: parser.NodeHooks{
			parser.AnyNode: NodeHook,
		},
	}
	treeDefHooks = parser.Hooks{
		Nodes: parser.NodeHooks{
			parser.AnyNode: newTreeDefHook,
		},
	}
	treeParser *parser.Parser
)

func init() {
	g, e := langdef.ParseString("src", treeGrammarDef)
	if e != nil {
		fmt.Println("error in tree description grammar:", e.Error())
	}
	treeParser, _ = parser.New(g)
}

func serialize(root NodeElement) string {
	if root == nil {
		return ""
	}

	b := &strings.Builder{}
	serializeSiblings(root.FirstChild(), b)
	if b.Len() == 0 {
		return ""
	} else {
		return b.String()[1:]
	}
}

func serializeSiblings(n Element, b *strings.Builder) {
	for n != nil {
		if !n.IsNode() {
			b.WriteString(" " + n.Token().Text())
		} else {
			b.WriteString(" (" + n.TypeName())
			serializeSiblings(n.(NodeElement).FirstChild(), b)
			b.WriteString(")")
		}
		n = n.Next()
	}
}

type parsingSample struct {
	src, expected string
}

func checkParsingSample(t *testing.T, p *parser.Parser, sampleNo int, s parsingSample) {
	root, e := p.ParseString(context.Background(), "parsingSample", s.src, treeHooks)
	if e != nil {
		t.Errorf("parsingSample #%d: unexpected error: %s", sampleNo, e)
		return
	}

	got := serialize(root.(NodeElement))
	if got != s.expected {
		t.Errorf("parsingSample #%d: %q expected, got %q", sampleNo, s.expected, got)
	}
}

func checkParsing(t *testing.T, gsrc string, samples []parsingSample) {
	g, e := langdef.ParseString("grammar", gsrc)
	if e != nil {
		t.Error("unexpected error: " + e.Error())
		return
	}

	p, _ := parser.New(g)

	for i, s := range samples {
		checkParsingSample(t, p, i, s)
	}
}

func assert(t *testing.T, flag bool) {
	if !flag {
		_, fname, line, _ := runtime.Caller(1)
		t.Fatalf("assertion failed at %s:%d", fname, line)
	}
}

func TestParsing(t *testing.T) {
	grammar := "$char = /[a-z]/; $digit = /[0-9]/; $op = /-/; " +
		"g = {$char | cd}; cd = $char, $digit, {'-', $char | cd};"
	samples := []parsingSample{
		{"", ""},
		{"a", "a"},
		{"a1", "(cd a 1)"},
		{"e2-e4a1-b", "(cd e 2 - (cd e 4)) (cd a 1 - b)"},
	}
	checkParsing(t, grammar, samples)
}

func parseTreeDescription(t *testing.T, src string) NodeElement {
	if treeParser == nil {
		t.Fatal("cannot parse tree")
	}

	res, e := treeParser.ParseString(context.Background(), "src", src, treeDefHooks)
	if e != nil {
		t.Fatal("error: " + e.Error())
	}

	return res.(NodeElement)
}

func buildTree(t *testing.T, src string) (NodeElement, map[string]Element) {
	root := parseTreeDescription(t, src)
	index := make(map[string]Element)
	Walk(root, WalkLtr, func(s WalkStat) WalkerFlags {
		n := s.Element
		if n.IsNode() {
			index[n.TypeName()] = n
		} else {
			index[n.Token().Text()] = n
		}
		return 0
	})

	return root, index
}

func TestWalker(t *testing.T) {
	it := NewWalker(nil, WalkLtr)
	assert(t, it.Next().Element == nil)

	src := "(foo (f1 (f11 f111 f112)) f2)(bar b1)(baz)"
	root, i := buildTree(t, src)

	it = NewWalker(root, WalkLtr)
	assert(t, it.Step(WalkerStop).Element == nil)
	assert(t, it.Next().Element == nil)

	it = NewWalker(root, WalkLtr)
	assert(t, it.Next().Element == root)
	assert(t, it.Next().Element == i["foo"])
	assert(t, it.Next().Element == i["f1"])
	assert(t, it.Next().Element == i["f11"])
	assert(t, it.Next().Element == i["f111"])
	assert(t, it.Step(WalkerSkipSiblings).Element == i["f2"])
	assert(t, it.Next().Element == i["bar"])
	assert(t, it.Step(WalkerSkipChildren).Element == i["baz"])
	assert(t, it.Next().Element == nil)

	it = NewWalker(root, WalkRtl)
	assert(t, it.Next().Element == root)
	assert(t, it.Next().Element == i["baz"])
	assert(t, it.Next().Element == i["bar"])
	assert(t, it.Step(WalkerSkipChildren).Element == i["foo"])
	assert(t, it.Next().Element == i["f2"])
	assert(t, it.Next().Element == i["f1"])
	assert(t, it.Next().Element == i["f11"])
	assert(t, it.Step(WalkerSkipSiblings).Element == i["f112"])
	assert(t, it.Step(WalkerSkipSiblings).Element == nil)
}

func matchNodes(t *testing.T, expected string, ns ...Element) {
	root := NewNodeElement("", nil)
	for _, n := range ns {
		if n.IsNode() {
			ntn := n.(NodeElement)
			root.AddChild(&nodeElement{typeName: ntn.TypeName()}, nil)
		} else {
			root.AddChild(&tokenElement{token: n.Token()}, nil)
		}
	}
	got := serialize(root)
	if got != expected {
		_, fname, line, _ := runtime.Caller(1)
		t.Errorf("%q expected, got %q at %s:%d", expected, got, fname, line)
	}
}

func TestWalkSkipChildren(t *testing.T) {
	src := "(foo (f1 (f11 f111)) f2)(bar b1)(baz)"
	expectedLtr := "() (foo) (f1) f2 (bar) b1 (baz)"
	expectedRtl := "() (baz) (bar) b1 (foo) f2 (f1)"
	root := parseTreeDescription(t, src)
	nodes := make([]Element, 0)
	f := func(s WalkStat) (flags WalkerFlags) {
		n := s.Element
		nodes = append(nodes, n)
		if n.Parent() != nil && n.Parent().Parent() != nil {
			flags = WalkerSkipChildren
		}
		return
	}

	Walk(root, WalkLtr, f)
	matchNodes(t, expectedLtr, nodes...)

	nodes = nodes[:0]
	Walk(root, WalkRtl, f)
	matchNodes(t, expectedRtl, nodes...)
}

func TestWalkSkipSiblings(t *testing.T) {
	src := "(foo f0 (f1 (f11 f111)) f2)(bar b1)(baz)"
	expectedLtr := "() (foo) f0 (f1) (f11) f111 (bar) b1 (baz)"
	expectedRtl := "() (baz) (bar) b1 (foo) f2 (f1) (f11) f111"
	root := parseTreeDescription(t, src)
	nodes := make([]Element, 0)
	f := func(s WalkStat) (flags WalkerFlags) {
		n := s.Element
		nodes = append(nodes, n)
		if n.TypeName() == "f1" {
			flags = WalkerSkipSiblings
		}
		return
	}

	Walk(root, WalkLtr, f)
	matchNodes(t, expectedLtr, nodes...)

	nodes = nodes[:0]
	Walk(root, WalkRtl, f)
	matchNodes(t, expectedRtl, nodes...)
}

func TestWalkStop(t *testing.T) {
	src := "(foo f0 (f1 (f11 f111)) f2)(bar b1)(baz)"
	expectedLtr := "() (foo) f0 (f1)"
	expectedRtl := "() (baz) (bar) b1 (foo) f2 (f1)"
	root := parseTreeDescription(t, src)
	nodes := make([]Element, 0)
	f := func(s WalkStat) (flags WalkerFlags) {
		n := s.Element
		nodes = append(nodes, n)
		if n.TypeName() == "f1" {
			flags = WalkerStop
		}
		return
	}

	Walk(root, WalkLtr, f)
	matchNodes(t, expectedLtr, nodes...)

	nodes = nodes[:0]
	Walk(root, WalkRtl, f)
	matchNodes(t, expectedRtl, nodes...)
}

func TestUnique(t *testing.T) {
	ntn := &nodeElement{typeName: "foo"}
	tn := &tokenElement{token: lexer.NewToken(0, "bar", []byte("BAR"), source.Pos{})}
	nodes := []Element{nil, ntn, nil, tn, nil, ntn, nil}

	got := NewSelector().Apply(nodes...)
	matchNodes(t, "(foo) BAR (foo)", got...)

	got = NewSelector().Unique().Apply(nodes...)
	matchNodes(t, "(foo) BAR", got...)
}

func TestTransform(t *testing.T) {
	f := func(n Element) []Element {
		if n.IsNode() {
			return Children(n)
		} else {
			return nil
		}
	}

	src := "(foo (x)) (bar baz (qux (y)))"
	children := "(foo) (bar)"
	expect1 := "(x) baz (qux)"
	expect2 := "(y)"
	nodes := Children(parseTreeDescription(t, src))

	xf := NewSelector().Extract(f)
	got1 := xf.Apply(nodes...)
	got2 := xf.Apply(got1...)
	matchNodes(t, expect1, got1...)
	matchNodes(t, expect2, got2...)

	got := NewSelector().Extract(f).Extract(f).Apply(nodes...)
	matchNodes(t, expect2, got...)

	matchNodes(t, children, nodes...)
}

func TestFilter(t *testing.T) {
	f := func(n Element) bool {
		nn, v := n.(NodeElement)
		return v && nn.FirstChild() != nil && nn.FirstChild().Next() == nil
	}

	src := "(foo) (bar baz) (qux (x) (y z)) (a b)"
	expect := "(bar) (a)"
	nodes := Children(parseTreeDescription(t, src))
	xf := NewSelector().Filter(f)
	got := xf.Apply(nodes...)
	matchNodes(t, expect, got...)
}

func TestSelect(t *testing.T) {
	f := func(n Element) []Element {
		return Children(n)
	}

	src := "(foo) (bar baz) (qux (x) (y z)) (a b)"
	expect := "baz (x) (y) b"
	nodes := Children(parseTreeDescription(t, src))
	xf := NewSelector().Extract(f)
	got := xf.Apply(nodes...)
	matchNodes(t, expect, got...)
}

func TestSearch(t *testing.T) {
	f := func(n Element) bool {
		nn, v := n.(NodeElement)
		return v && nn.FirstChild() != nil && nn.FirstChild().Next() == nil
	}

	src := "(foo) (bar baz) (qux (x y)) (a b (c d))"
	expect0 := "(bar) (qux) (c)"
	expect1 := "(bar) (qux) (x) (c)"
	nodes := Children(parseTreeDescription(t, src))
	got0 := NewSelector().Search(f).Apply(nodes...)
	got1 := NewSelector().DeepSearch(f).Apply(nodes...)
	matchNodes(t, expect0, got0...)
	matchNodes(t, expect1, got1...)
}

func TestIsNot(t *testing.T) {
	f := func(n Element) bool {
		return n.IsNode()
	}
	ff := IsNot(f)
	tn := tokenElement{}
	ntn := nodeElement{}
	assert(t, ff(&tn))
	assert(t, !ff(&ntn))
}

func TestIsAny(t *testing.T) {
	f1 := func(n Element) bool {
		return !n.IsNode()
	}
	f2 := func(n Element) bool {
		nn, v := n.(NodeElement)
		return v && nn.FirstChild() == nil
	}
	ff := IsAny(f1, f2)
	tn := tokenElement{}
	nt := nodeElement{}
	ntn := nodeElement{firstChild: &nt}
	assert(t, ff(&tn))
	assert(t, ff(&nt))
	assert(t, !ff(&ntn))
}

func TestIsAll(t *testing.T) {
	f1 := func(n Element) bool {
		return n.IsNode()
	}
	f2 := func(n Element) bool {
		nn, v := n.(NodeElement)
		return v && nn.FirstChild() != nil
	}
	ff := IsAll(f1, f2)
	tn := tokenElement{}
	nt := nodeElement{}
	ntn := nodeElement{firstChild: &nt}
	assert(t, !ff(&tn))
	assert(t, !ff(&nt))
	assert(t, ff(&ntn))
}

func TestIsA(t *testing.T) {
	ff := IsA("foo", "qux")
	tn0 := tokenElement{token: lexer.NewToken(1, "bar", []byte("foo"), source.Pos{})}
	tn1 := tokenElement{token: lexer.NewToken(2, "foo", nil, source.Pos{})}
	nt0 := nodeElement{typeName: "baz"}
	nt1 := nodeElement{typeName: "qux"}
	assert(t, ff(&tn1))
	assert(t, !ff(&tn0))
	assert(t, ff(&nt1))
	assert(t, !ff(&nt0))
}

func TestIsALiteral(t *testing.T) {
	ff := IsALiteral("foo", "qux")
	tn0 := tokenElement{token: lexer.NewToken(1, "foo", []byte("bar"), source.Pos{})}
	tn1 := tokenElement{token: lexer.NewToken(2, "bar", []byte("foo"), source.Pos{})}
	tn2 := tokenElement{token: lexer.NewToken(3, "baz", []byte("qux"), source.Pos{})}
	nt := nodeElement{typeName: "foo", token: lexer.NewToken(4, "foo", []byte("foo"), source.Pos{})}
	assert(t, !ff(&tn0))
	assert(t, ff(&tn1))
	assert(t, ff(&tn2))
	assert(t, !ff(&nt))
}

func TestAny(t *testing.T) {
	empty := func(n Element) []Element {
		return nil
	}
	ident := func(n Element) []Element {
		return []Element{n}
	}
	ans := func(n Element) []Element {
		res := n.Parent()
		if res == nil {
			return nil
		} else {
			return []Element{res}
		}
	}
	parent := nodeElement{typeName: "foo"}
	child := nodeElement{typeName: "bar", parent: &parent}

	matchNodes(t, "(foo)", Any(ans, ident)(&child)...)
	matchNodes(t, "(foo)", Any(ans, ident)(&parent)...)
	matchNodes(t, "", Any(ans, empty)(&parent)...)
}

func TestAll(t *testing.T) {
	empty := func(n Element) []Element {
		return nil
	}
	ident := func(n Element) []Element {
		return []Element{n}
	}
	ans := func(n Element) []Element {
		res := n.Parent()
		if res == nil {
			return nil
		} else {
			return []Element{res}
		}
	}
	parent := nodeElement{typeName: "foo"}
	child := nodeElement{typeName: "bar", parent: &parent}

	matchNodes(t, "(foo) (bar)", All(ans, ident)(&child)...)
	matchNodes(t, "(foo)", Any(ans, ident)(&parent)...)
	matchNodes(t, "(foo)", Any(ans, empty)(&child)...)
	matchNodes(t, "", Any(ans, empty)(&parent)...)
}

func TestNth(t *testing.T) {
	els := []Element{
		&nodeElement{typeName: "foo"},
		&nodeElement{typeName: "bar"},
		&nodeElement{typeName: "baz"},
		&nodeElement{typeName: "qux"},
	}
	ex := func(Element) []Element {
		return els
	}
	samples := []struct {
		in, expected []int
	}{
		{nil, nil},
		{[]int{1}, []int{1}},
		{[]int{2, 4, 0}, []int{2, 0}},
		{[]int{-1, 0, 1}, []int{0, 1}},
	}
	for i, s := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func(t *testing.T) {
			got := Nth(ex, s.in...)(els[0])
			if len(got) != len(s.expected) {
				t.Errorf("expecting %d elements, got %d", len(s.expected), len(got))
				return
			}

			for i, el := range got {
				if el != els[s.expected[i]] {
					t.Errorf("element #%d: expecting %q, got %q", i, els[s.expected[i]].TypeName(), el.TypeName())
					return
				}
			}
		})
	}
}

func TestAncestors(t *testing.T) {
	foo := nodeElement{typeName: "foo"}
	bar := nodeElement{typeName: "bar", parent: &foo}
	baz := nodeElement{typeName: "baz", parent: &bar}
	qux := nodeElement{typeName: "qux", parent: &baz}
	matchNodes(t, "", Ancestors(&foo)...)
	matchNodes(t, "(foo)", Ancestors(&bar)...)
	matchNodes(t, "(bar) (foo)", Ancestors(&baz)...)
	matchNodes(t, "(baz) (bar) (foo)", Ancestors(&qux)...)
}

func TestPrevSiblings(t *testing.T) {
	els := []Element{
		&nodeElement{typeName: "foo"},
		&nodeElement{typeName: "bar"},
		&nodeElement{typeName: "baz"},
	}
	parent := &nodeElement{}
	for _, el := range els {
		parent.AddChild(el, nil)
	}

	matchNodes(t, "", PrevSiblings(els[0])...)
	matchNodes(t, "(foo)", PrevSiblings(els[1])...)
	matchNodes(t, "(bar) (foo)", PrevSiblings(els[2])...)
}

func TestNextSiblings(t *testing.T) {
	els := []Element{
		&nodeElement{typeName: "foo"},
		&nodeElement{typeName: "bar"},
		&nodeElement{typeName: "baz"},
	}
	parent := &nodeElement{}
	for _, el := range els {
		parent.AddChild(el, nil)
	}

	matchNodes(t, "(bar) (baz)", NextSiblings(els[0])...)
	matchNodes(t, "(baz)", NextSiblings(els[1])...)
	matchNodes(t, "", NextSiblings(els[2])...)
}
