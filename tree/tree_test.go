package tree

import (
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
	nt *nodeNode
	gotName bool
}

func (h *treeDefHook) NewNode (node string, token *lexer.Token) error {
	return nil
}

func (h *treeDefHook) HandleNode (node string, result interface{}) error {
	h.nt.AddChild(result.(Node), nil)
	return nil
}

func (h *treeDefHook) HandleToken (token *lexer.Token) error {
	t := token.Text()
	if t != "(" && t != ")" {
		if h.gotName {
			h.nt.AddChild(NewTokenNode(token), nil)
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

func newTreeDefHook (node string, tok *lexer.Token, pc *parser.ParseContext) (parser.NodeHookInstance, error) {
	return &treeDefHook{nt: &nodeNode{token: tok}, gotName: (node == "tree-def")}, nil
}

var (
	treeHooks = &parser.Hooks{
		Nodes: parser.NodeHooks{
			parser.AnyNode: NodeHook,
		},
	}
	treeDefHooks = &parser.Hooks{
		Nodes: parser.NodeHooks{
			parser.AnyNode: newTreeDefHook,
		},
	}
	treeParser *parser.Parser
)

func init () {
	g, e := langdef.ParseString("src", treeGrammarDef)
	if e != nil {
		fmt.Println("error in tree description grammar:", e.Error())
	}
	treeParser = parser.New(g)
}

func serialize (root NodeNode) string {
	if root == nil {
		return ""
	}

	b := &strings.Builder{}
	serializeSiblings(root.FirstChild(), b)
	if b.Len() == 0 {
		return ""
	} else {
		return b.String()[1 :]
	}
}

func serializeSiblings (n Node, b *strings.Builder) {
	for n != nil {
		if !n.IsNode() {
			b.WriteString(" " + n.Token().Text())
		} else {
			b.WriteString(" (" + n.TypeName())
			serializeSiblings(n.(NodeNode).FirstChild(), b)
			b.WriteString(")")
		}
		n = n.Next()
	}
}

type parsingSample struct {
	src, expected string
}

func checkParsingSample (t *testing.T, p *parser.Parser, sampleNo int, s parsingSample) {
	root, e := p.ParseString("parsingSample", s.src, treeHooks)
	if e != nil {
		t.Errorf("parsingSample #%d: unexpected error: %s", sampleNo, e)
		return
	}

	got := serialize(root.(NodeNode))
	if got != s.expected {
		t.Errorf("parsingSample #%d: %q expected, got %q", sampleNo, s.expected, got)
	}
}

func checkParsing (t *testing.T, gsrc string, samples []parsingSample) {
	g, e := langdef.ParseString("grammar", gsrc)
	if e != nil {
		t.Error("unexpected error: " + e.Error())
		return
	}

	p := parser.New(g)

	for i, s := range samples {
		checkParsingSample(t, p, i, s)
	}
}

func assert (t *testing.T, flag bool) {
	if !flag {
		_, fname, line, _ := runtime.Caller(1)
		t.Fatalf("assertion failed at %s:%d", fname, line)
	}
}

func TestParsing (t *testing.T) {
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

func parseTreeDescription (t *testing.T, src string) NodeNode {
	if treeParser == nil {
		t.Fatal("cannot parse tree")
	}

	res, e := treeParser.ParseString("src", src, treeDefHooks)
	if e != nil {
		t.Fatal("error: " + e.Error())
	}

	return res.(NodeNode)
}

func buildTree (t *testing.T, src string) (NodeNode, map[string]Node) {
	root := parseTreeDescription(t, src)
	index := make(map[string]Node)
	Walk(root, WalkLtr, func (n Node) WalkerFlags {
		if n.IsNode() {
			index[n.TypeName()] = n
		} else {
			index[n.Token().Text()] = n
		}
		return 0
	})

	return root, index
}

func TestIterator (t *testing.T) {
	it := NewIterator(nil, WalkLtr)
	assert(t, it.Next() == nil)

	src := "(foo (f1 (f11 f111 f112)) f2)(bar b1)(baz)"
	root, i := buildTree(t, src)

	it = NewIterator(root, WalkLtr)
	assert(t, it.Step(WalkerStop) == nil)
	assert(t, it.Next() == nil)

	it = NewIterator(root, WalkLtr)
	assert(t, it.Next() == root)
	assert(t, it.Next() == i["foo"])
	assert(t, it.Next() == i["f1"])
	assert(t, it.Next() == i["f11"])
	assert(t, it.Next() == i["f111"])
	assert(t, it.Step(WalkerSkipSiblings) == i["f2"])
	assert(t, it.Next() == i["bar"])
	assert(t, it.Step(WalkerSkipChildren) == i["baz"])
	assert(t, it.Next() == nil)

	it = NewIterator(root, WalkRtl)
	assert(t, it.Next() == root)
	assert(t, it.Next() == i["baz"])
	assert(t, it.Next() == i["bar"])
	assert(t, it.Step(WalkerSkipChildren) == i["foo"])
	assert(t, it.Next() == i["f2"])
	assert(t, it.Next() == i["f1"])
	assert(t, it.Next() == i["f11"])
	assert(t, it.Step(WalkerSkipSiblings) == i["f112"])
	assert(t, it.Step(WalkerSkipSiblings) == nil)
}

func matchNodes (t *testing.T, expected string, ns ... Node) {
	root := NewNodeNode("", nil)
	for _, n := range ns {
		if n.IsNode() {
			ntn := n.(NodeNode)
			root.AddChild(&nodeNode{typeName: ntn.TypeName()}, nil)
		} else {
			root.AddChild(&tokenNode{token: n.Token()}, nil)
		}
	}
	got := serialize(root)
	if got != expected {
		_, fname, line, _ := runtime.Caller(1)
		t.Errorf("%q expected, got %q at %s:%d", expected, got, fname, line)
	}
}

func TestWalkSkipChildren (t *testing.T) {
	src := "(foo (f1 (f11 f111)) f2)(bar b1)(baz)"
	expectedLtr := "() (foo) (f1) f2 (bar) b1 (baz)"
	expectedRtl := "() (baz) (bar) b1 (foo) f2 (f1)"
	root := parseTreeDescription(t, src)
	nodes := make([]Node, 0)
	f := func (n Node) (flags WalkerFlags) {
		nodes = append(nodes, n)
		if NodeLevel(n) >= 2 {
			flags = WalkerSkipChildren
		}
		return
	}

	Walk(root, WalkLtr, f)
	matchNodes(t, expectedLtr, nodes ...)

	nodes = nodes[: 0]
	Walk(root, WalkRtl, f)
	matchNodes(t, expectedRtl, nodes ...)
}

func TestWalkSkipSiblings (t *testing.T) {
	src := "(foo f0 (f1 (f11 f111)) f2)(bar b1)(baz)"
	expectedLtr := "() (foo) f0 (f1) (f11) f111 (bar) b1 (baz)"
	expectedRtl := "() (baz) (bar) b1 (foo) f2 (f1) (f11) f111"
	root := parseTreeDescription(t, src)
	nodes := make([]Node, 0)
	f := func (n Node) (flags WalkerFlags) {
		nodes = append(nodes, n)
		if n.TypeName() == "f1" {
			flags = WalkerSkipSiblings
		}
		return
	}

	Walk(root, WalkLtr, f)
	matchNodes(t, expectedLtr, nodes ...)

	nodes = nodes[: 0]
	Walk(root, WalkRtl, f)
	matchNodes(t, expectedRtl, nodes ...)
}

func TestWalkStop (t *testing.T) {
	src := "(foo f0 (f1 (f11 f111)) f2)(bar b1)(baz)"
	expectedLtr := "() (foo) f0 (f1)"
	expectedRtl := "() (baz) (bar) b1 (foo) f2 (f1)"
	root := parseTreeDescription(t, src)
	nodes := make([]Node, 0)
	f := func (n Node) (flags WalkerFlags) {
		nodes = append(nodes, n)
		if n.TypeName() == "f1" {
			flags = WalkerStop
		}
		return
	}

	Walk(root, WalkLtr, f)
	matchNodes(t, expectedLtr, nodes ...)

	nodes = nodes[: 0]
	Walk(root, WalkRtl, f)
	matchNodes(t, expectedRtl, nodes ...)
}

func TestTransformer (t *testing.T) {
	ntn := &nodeNode{typeName: "foo"}
	tn := &tokenNode{token: lexer.NewToken(0, "bar", "BAR", source.Pos{})}
	nodes := []Node{nil, ntn, nil, tn, nil, ntn, nil}
	got := NewSelector().Apply(nodes ...)
	assert(t, len(got) == 2)
	matchNodes(t, "(foo) BAR", got ...)
}

func TestTransform (t *testing.T) {
	f := func (n Node) []Node {
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

	xf := NewSelector().Use(f)
	got1 := xf.Apply(nodes ...)
	got2 := xf.Apply(got1 ...)
	matchNodes(t, expect1, got1 ...)
	matchNodes(t, expect2, got2 ...)

	got := NewSelector().Use(f).Use(f).Apply(nodes ...)
	matchNodes(t, expect2, got ...)

	matchNodes(t, children, nodes ...)
}

func TestFilter (t *testing.T) {
	f := func (n Node) bool {
		return (NumOfChildren(n, 0) == 1)
	}

	src := "(foo) (bar baz) (qux (x) (y z)) (a b)"
	expect := "(bar) (a)"
	nodes := Children(parseTreeDescription(t, src))
	xf := NewSelector().Filter(f)
	got := xf.Apply(nodes ...)
	matchNodes(t, expect, got ...)
}

func TestSelect (t *testing.T) {
	f := func (n Node) []Node {
		return Children(n)
	}

	src := "(foo) (bar baz) (qux (x) (y z)) (a b)"
	expect := "baz (x) (y) b"
	nodes := Children(parseTreeDescription(t, src))
	xf := NewSelector().Extract(f)
	got := xf.Apply(nodes ...)
	matchNodes(t, expect, got ...)
}

func TestSearch (t *testing.T) {
	f := func (n Node) bool {
		return (NumOfChildren(n, 0) == 1)
	}

	src := "(foo) (bar baz) (qux (x y)) (a b (c d))"
	expect0 := "(bar) (qux) (c)"
	expect1 := "(bar) (qux) (x) (c)"
	nodes := Children(parseTreeDescription(t, src))
	got0 := NewSelector().Search(f).Apply(nodes ...)
	got1 := NewSelector().DeepSearch(f).Apply(nodes ...)
	matchNodes(t, expect0, got0 ...)
	matchNodes(t, expect1, got1 ...)
}

func TestIsNot (t *testing.T) {
	f := func (n Node) bool {
		return n.IsNode()
	}
	ff := IsNot(f)
	tn := tokenNode{}
	ntn := nodeNode{}
	assert(t, ff(&tn))
	assert(t, !ff(&ntn))
}

func TestIsAny (t *testing.T) {
	f1 := func (n Node) bool {
		return !n.IsNode()
	}
	f2 := func (n Node) bool {
		return (NumOfChildren(n, 0) == 0)
	}
	ff := IsAny(f1, f2)
	tn := tokenNode{}
	nt := nodeNode{}
	ntn := nodeNode{firstChild: &nt}
	assert(t, ff(&tn))
	assert(t, ff(&nt))
	assert(t, !ff(&ntn))
}

func TestIsAll (t *testing.T) {
	f1 := func (n Node) bool {
		return n.IsNode()
	}
	f2 := func (n Node) bool {
		return (NumOfChildren(n, 0) > 0)
	}
	ff := IsAll(f1, f2)
	tn := tokenNode{}
	nt := nodeNode{}
	ntn := nodeNode{firstChild: &nt}
	assert(t, !ff(&tn))
	assert(t, !ff(&nt))
	assert(t, ff(&ntn))
}

func TestIsA (t *testing.T) {
	ff := IsA("foo", "qux")
	tn0 := tokenNode{token: lexer.NewToken(1, "bar", "foo", source.Pos{})}
	tn1 := tokenNode{token: lexer.NewToken(2, "foo", "", source.Pos{})}
	nt0 := nodeNode{typeName: "baz"}
	nt1 := nodeNode{typeName: "qux"}
	assert(t, ff(&tn1))
	assert(t, !ff(&tn0))
	assert(t, ff(&nt1))
	assert(t, !ff(&nt0))
}

func TestIsALiteral (t *testing.T) {
	ff := IsALiteral("foo", "qux")
	tn0 := tokenNode{token: lexer.NewToken(1, "foo", "bar", source.Pos{})}
	tn1 := tokenNode{token: lexer.NewToken(2, "bar", "foo", source.Pos{})}
	tn2 := tokenNode{token: lexer.NewToken(3, "baz", "qux", source.Pos{})}
	nt := nodeNode{typeName: "foo", token: lexer.NewToken(4, "foo", "foo", source.Pos{})}
	assert(t, !ff(&tn0))
	assert(t, ff(&tn1))
	assert(t, ff(&tn2))
	assert(t, !ff(&nt))
}

func TestHas (t *testing.T) {
	_, i := buildTree(t, "(foo (y x)) (bar z) (baz)")

	ff := Has(Children, IsALiteral("x"))
	assert(t, !ff(i["foo"]))
	assert(t, ff(i["y"]))
	assert(t, !ff(i["x"]))
	assert(t, !ff(i["bar"]))
	assert(t, !ff(i["baz"]))

	ff = Has(Children, nil)
	assert(t, ff(i["foo"]))
	assert(t, ff(i["y"]))
	assert(t, !ff(i["x"]))
	assert(t, ff(i["bar"]))
	assert(t, !ff(i["baz"]))

	ff = Has(nil, IsALiteral("x"))
	assert(t, ff(i["foo"]))
	assert(t, ff(i["y"]))
	assert(t, ff(i["x"]))
	assert(t, !ff(i["bar"]))
	assert(t, !ff(i["baz"]))
}

func TestAny (t *testing.T) {
	empty := func (n Node) []Node {
		return nil
	}
	ident := func (n Node) []Node {
		return []Node{n}
	}
	ans := func (n Node) []Node {
		res := Ancestor(n, 0)
		if res == nil {
			return nil
		} else {
			return []Node{res}
		}
	}
	parent := nodeNode{typeName: "foo"}
	child := nodeNode{typeName: "bar", parent: &parent}

	matchNodes(t, "(foo)", Any(ans, ident)(&child) ...)
	matchNodes(t, "(foo)", Any(ans, ident)(&parent) ...)
	matchNodes(t, "", Any(ans, empty)(&parent) ...)
}

func TestAll (t *testing.T) {
	empty := func (n Node) []Node {
		return nil
	}
	ident := func (n Node) []Node {
		return []Node{n}
	}
	ans := func (n Node) []Node {
		res := Ancestor(n, 0)
		if res == nil {
			return nil
		} else {
			return []Node{res}
		}
	}
	parent := nodeNode{typeName: "foo"}
	child := nodeNode{typeName: "bar", parent: &parent}

	matchNodes(t, "(foo) (bar)", All(ans, ident)(&child) ...)
	matchNodes(t, "(foo)", Any(ans, ident)(&parent) ...)
	matchNodes(t, "(foo)", Any(ans, empty)(&child) ...)
	matchNodes(t, "", Any(ans, empty)(&parent) ...)
}

func TestAncestors (t *testing.T) {
	foo := nodeNode{typeName: "foo"}
	bar := nodeNode{typeName: "bar", parent: &foo}
	baz := nodeNode{typeName: "baz", parent: &bar}
	qux := nodeNode{typeName: "qux", parent: &baz}
	f := Ancestors(1, 2, 0)

	matchNodes(t, "", f(&foo) ...)
	matchNodes(t, "(foo)", f(&bar) ...)
	matchNodes(t, "(foo) (bar)", f(&baz) ...)
	matchNodes(t, "(bar) (foo) (baz)", f(&qux) ...)
}

func TestNthChildren (t *testing.T) {
	src := "(foo bar baz) (a b) (x)"
	root, i := buildTree(t, src)
	f := NthChildren(1, 2, 0, -1, -2)

	matchNodes(t, "(a) (x) (foo) (x) (a)", f(root) ...)
	matchNodes(t, "baz bar baz bar", f(i["foo"]) ...)
	matchNodes(t, "b b", f(i["a"]) ...)
	matchNodes(t, "", f(i["x"]) ...)
}

func TestNthSiblings (t *testing.T) {
	src := "(foo bar baz qux) (a b c) (x y) (z)"
	_, i := buildTree(t, src)
	f := NthSiblings(1, 2, 0, -2, -1)

	matchNodes(t, "(a) (x) (foo)", f(i["foo"]) ...)
	matchNodes(t, "baz qux bar", f(i["bar"]) ...)
	matchNodes(t, "qux baz bar", f(i["baz"]) ...)
	matchNodes(t, "qux bar baz", f(i["qux"]) ...)
	matchNodes(t, "(x) (z) (a) (foo)", f(i["a"]) ...)
	matchNodes(t, "c b", f(i["b"]) ...)
	matchNodes(t, "c b", f(i["c"]) ...)
	matchNodes(t, "(z) (x) (foo) (a)", f(i["x"]) ...)
	matchNodes(t, "y", f(i["y"]) ...)
	matchNodes(t, "(z) (a) (x)", f(i["z"]) ...)
}
