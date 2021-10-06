package tree

import (
	"testing"
)

func TestAncestor (t *testing.T) {
	assert(t, Ancestor(nil, 0) == nil)
	assert(t, Ancestor(nil, -1) == nil)
	assert(t, Ancestor(nil, 1) == nil)

	root, i := buildTree(t, "(child t (nt))")
	child := i["child"]
	grandT := i["t"]
	grandNT := i["nt"]

	assert(t, Ancestor(root, 0) == nil)

	assert(t, Ancestor(grandT, -10) == grandT)
	assert(t, Ancestor(grandT, -1) == grandT)
	assert(t, Ancestor(grandT, 0) == child)
	assert(t, Ancestor(grandT, 1) == root)
	assert(t, Ancestor(grandT, 2) == nil)
	assert(t, Ancestor(grandT, 20) == nil)

	assert(t, Ancestor(grandNT, -10) == grandNT)
	assert(t, Ancestor(grandNT, -1) == grandNT)
	assert(t, Ancestor(grandNT, 0) == child)
	assert(t, Ancestor(grandNT, 1) == root)
	assert(t, Ancestor(grandNT, 2) == nil)
	assert(t, Ancestor(grandNT, 20) == nil)
}

func TestNodeLevel (t *testing.T) {
	assert(t, NodeLevel(nil) == 0)

	root, i := buildTree(t, "(child (grand))")
	child := i["child"]
	grand := i["grand"]

	assert(t, NodeLevel(root) == 0)
	assert(t, NodeLevel(child) == 1)
	assert(t, NodeLevel(grand) == 2)
}

func TestSiblingIndex (t *testing.T) {
	assert(t, SiblingIndex(nil) == 0)

	_, i := buildTree(t, "(1st) (2nd) (3rd)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]

	assert(t, SiblingIndex(first) == 0)
	assert(t, SiblingIndex(second) == 1)
	assert(t, SiblingIndex(third) == 2)
}

func TestNthChild (t *testing.T) {
	assert(t, NthChild(nil, 0) == nil)

	parent, i := buildTree(t, "(1st) (2nd grand) (3rd)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]

	assert(t, NthChild(first, 0) == nil)

	assert(t, NthChild(parent, -40) == nil)
	assert(t, NthChild(parent, -4) == nil)
	assert(t, NthChild(parent, -3) == first)
	assert(t, NthChild(parent, -2) == second)
	assert(t, NthChild(parent, -1) == third)
	assert(t, NthChild(parent, 0) == first)
	assert(t, NthChild(parent, 1) == second)
	assert(t, NthChild(parent, 2) == third)
	assert(t, NthChild(parent, 3) == nil)
	assert(t, NthChild(parent, 30) == nil)
}

func TestNthSibling (t *testing.T) {
	assert(t, NthSibling(nil, 0) == nil)

	_, i := buildTree(t, "(1st) (2nd) (3rd) (4th)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]
	fourth := i["4th"]

	assert(t, NthSibling(second, -20) == nil)
	assert(t, NthSibling(second, -2) == nil)
	assert(t, NthSibling(second, -1) == first)
	assert(t, NthSibling(second, 0) == second)
	assert(t, NthSibling(second, 1) == third)
	assert(t, NthSibling(second, 2) == fourth)
	assert(t, NthSibling(second, 3) == nil)
	assert(t, NthSibling(second, 30) == nil)
}

func TestNumOfChildren (t *testing.T) {
	assert(t, NumOfChildren(nil, 0) == 0)

	root, i := buildTree(t, "(1st) (2nd (leaf)) (3rd)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]
	leaf := i["leaf"]

	assert(t, NumOfChildren(root, AllLevels) == 4)
	assert(t, NumOfChildren(root, 0) == 3)
	assert(t, NumOfChildren(root, 1) == 4)
	assert(t, NumOfChildren(root, 2) == 4)
	assert(t, NumOfChildren(root, 20) == 4)
	assert(t, NumOfChildren(first, 0) == 0)
	assert(t, NumOfChildren(second, 0) == 1)
	assert(t, NumOfChildren(third, 0) == 0)
	assert(t, NumOfChildren(leaf, 0) == 0)
}

func TestFirstTokenNode (t *testing.T) {
	assert (t, FirstTokenNode(nil) == nil)

	root, i := buildTree(t, "(1st) (2nd leaf other)")
	first := i["1st"]
	second := i["2nd"]
	leaf := i["leaf"]

	assert(t, FirstTokenNode(root) == leaf)
	assert(t, FirstTokenNode(first) == nil)
	assert(t, FirstTokenNode(second) == leaf)
	assert(t, FirstTokenNode(leaf) == leaf)
}

func TestLastTokenNode (t *testing.T) {
	assert (t, LastTokenNode(nil) == nil)

	root, i := buildTree(t, "(1st) (2nd leaf other)")
	first := i["1st"]
	second := i["2nd"]
	other := i["other"]

	assert(t, LastTokenNode(root) == other)
	assert(t, LastTokenNode(first) == nil)
	assert(t, LastTokenNode(second) == other)
	assert(t, LastTokenNode(other) == other)
}

func TestNextTokenNode(t *testing.T) {
	assert(t, NextTokenNode(nil) == nil)

	root, i := buildTree(t, "(1st) (2nd (nested foo bar)) (3rd baz)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]
	foo := i["foo"]
	bar := i["bar"]
	baz := i["baz"]

	assert(t, NextTokenNode(root) == nil)
	assert(t, NextTokenNode(first) == foo)
	assert(t, NextTokenNode(second) == baz)
	assert(t, NextTokenNode(third) == nil)
	assert(t, NextTokenNode(foo) == bar)
	assert(t, NextTokenNode(bar) == baz)
	assert(t, NextTokenNode(baz) == nil)
}

func TestPrevTokenNode(t *testing.T) {
	assert(t, NextTokenNode(nil) == nil)

	root, i := buildTree(t, "(1st foo) (2nd (nested bar baz)) (3rd)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]
	foo := i["foo"]
	bar := i["bar"]
	baz := i["baz"]

	assert(t, PrevTokenNode(root) == nil)
	assert(t, PrevTokenNode(first) == nil)
	assert(t, PrevTokenNode(second) == foo)
	assert(t, PrevTokenNode(third) == baz)
	assert(t, PrevTokenNode(foo) == nil)
	assert(t, PrevTokenNode(bar) == foo)
	assert(t, PrevTokenNode(baz) == bar)
}

func TestChildren (t *testing.T) {
	assert(t, len(Children(nil)) == 0)

	src := "(foo) (bar baz (qux (x)))"
	children := "(foo) (bar)"
	root, i := buildTree(t, src)

	matchNodes(t, children, Children(root) ...)
	matchNodes(t, "", Children(i["foo"]) ...)
	matchNodes(t, "baz (qux)", Children(i["bar"]) ...)
	matchNodes(t, "", Children(i["baz"]) ...)
	matchNodes(t, "(x)", Children(i["qux"]) ...)
}

func TestDetach (t *testing.T) {
	Detach(nil)

	root, i := buildTree(t, "(1st) (2nd) (3rd) (4th)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]
	fourth := i["4th"]

	Detach(root)

	Detach(third)
	assert(t, third.Parent() == nil)
	assert(t, third.Prev() == nil)
	assert(t, third.Next() == nil)
	assert(t, second.Next() == fourth)
	assert(t, fourth.Prev() == second)

	Detach(first)
	assert(t, first.Parent() == nil)
	assert(t, first.Prev() == nil)
	assert(t, first.Next() == nil)
	assert(t, second.Prev() == nil)
	assert(t, root.FirstChild() == second)

	Detach(fourth)
	assert(t, fourth.Parent() == nil)
	assert(t, fourth.Prev() == nil)
	assert(t, fourth.Next() == nil)
	assert(t, second.Next() == nil)

	Detach(second)
	assert(t, second.Parent() == nil)
	assert(t, second.Prev() == nil)
	assert(t, second.Next() == nil)
	assert(t, root.FirstChild() == nil)
	assert(t, root.LastChild() == nil)
}

func TestReplace (t *testing.T) {
	Replace(nil, nil)

	root, i := buildTree(t, "(1st) (2nd) (3rd) (re) (re2)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]
	re := i["re"]
	re2 := i["re2"]

	Replace(nil, re2)
	assert(t, re2.Parent() == root)

	Replace(re2, nil)
	assert(t, re2.Parent() == nil)
	assert(t, re2.Prev() == nil)

	Replace(first, re)
	assert(t, first.Parent() == nil)
	assert(t, first.Next() == nil)
	assert(t, root.FirstChild() == re)
	assert(t, re.Parent() == root)
	assert(t, re.Next() == second)
	assert(t, third.Next() == nil)

	Replace(re, first)
	Replace(second, re)
	assert(t, second.Parent() == nil)
	assert(t, second.Prev() == nil)
	assert(t, second.Next() == nil)
	assert(t, re.Parent() == root)
	assert(t, re.Prev() == first)
	assert(t, re.Next() == third)
	assert(t, first.Next() == re)
	assert(t, third.Prev() == re)

	Replace(re, second)
	Replace(third, re)
	assert(t, third.Parent() == nil)
	assert(t, third.Prev() == nil)
	assert(t, third.Next() == nil)
	assert(t, re.Parent() == root)
	assert(t, re.Prev() == second)
	assert(t, re.Next() == nil)
	assert(t, second.Next() == re)
}

func TestAppendSibling (t *testing.T) {
	AppendSibling(nil, nil)

	root, i := buildTree(t, "(1st) (2nd)")
	first := i["1st"]
	second := i["2nd"]
	re := &nonTermNode{}

	AppendSibling(nil, first)
	assert(t, first.Parent() == root)

	AppendSibling(first, nil)
	assert(t, first.Next() == second)

	AppendSibling(first, re)
	assert(t, re.Parent() == root)
	assert(t, re.Prev() == first)
	assert(t, re.Next() == second)
	assert(t, first.Next() == re)
	assert(t, second.Prev() == re)

	AppendSibling(second, re)
	assert(t, first.Next() == second)
	assert(t, second.Prev() == first)
	assert(t, second.Next() == re)
	assert(t, re.Prev() == second)
	assert(t, re.Next() == nil)
}

func TestPrependSibling (t *testing.T) {
	PrependSibling(nil, nil)

	root, i := buildTree(t, "(1st) (2nd)")
	first := i["1st"]
	second := i["2nd"]
	re := &nonTermNode{}

	PrependSibling(nil, first)
	assert(t, first.Parent() == root)

	PrependSibling(second, nil)
	assert(t, second.Prev() == first)

	PrependSibling(first, re)
	assert(t, re.Parent() == root)
	assert(t, re.Prev() == nil)
	assert(t, re.Next() == first)
	assert(t, first.Prev() == re)

	PrependSibling(second, re)
	assert(t, first.Next() == re)
	assert(t, second.Prev() == re)
	assert(t, re.Prev() == first)
	assert(t, re.Next() == second)
}

func TestAppendChild (t *testing.T) {
	AppendChild(nil, nil)

	root := &nonTermNode{}
	re := &nonTermNode{}
	re2 := &nonTermNode{}

	AppendChild(root, re)
	assert(t, root.FirstChild() == re)
	assert(t, re.Parent() == root)

	AppendChild(root, nil)
	AppendChild(nil, re)
	assert(t, root.FirstChild() == re)
	assert(t, re.Parent() == root)

	AppendChild(root, re2)
	assert(t, root.FirstChild() == re)
	assert(t, re2.Parent() == root)
	assert(t, re.Next() == re2)
	assert(t, re2.Prev() == re)
}
