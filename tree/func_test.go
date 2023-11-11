package tree

import (
	"testing"
)

func TestFirstTokenElement(t *testing.T) {
	assert(t, FirstTokenElement(nil) == nil)

	root, i := buildTree(t, "(1st) (2nd leaf other)")
	first := i["1st"]
	second := i["2nd"]
	leaf := i["leaf"]

	assert(t, FirstTokenElement(root) == leaf)
	assert(t, FirstTokenElement(first) == nil)
	assert(t, FirstTokenElement(second) == leaf)
	assert(t, FirstTokenElement(leaf) == leaf)
}

func TestLastTokenElement(t *testing.T) {
	assert(t, LastTokenElement(nil) == nil)

	root, i := buildTree(t, "(1st) (2nd leaf other)")
	first := i["1st"]
	second := i["2nd"]
	other := i["other"]

	assert(t, LastTokenElement(root) == other)
	assert(t, LastTokenElement(first) == nil)
	assert(t, LastTokenElement(second) == other)
	assert(t, LastTokenElement(other) == other)
}

func TestNextTokenElement(t *testing.T) {
	assert(t, NextTokenElement(nil) == nil)

	root, i := buildTree(t, "(1st) (2nd (nested foo bar)) (3rd baz)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]
	foo := i["foo"]
	bar := i["bar"]
	baz := i["baz"]

	assert(t, NextTokenElement(root) == nil)
	assert(t, NextTokenElement(first) == foo)
	assert(t, NextTokenElement(second) == baz)
	assert(t, NextTokenElement(third) == nil)
	assert(t, NextTokenElement(foo) == bar)
	assert(t, NextTokenElement(bar) == baz)
	assert(t, NextTokenElement(baz) == nil)
}

func TestPrevTokenElement(t *testing.T) {
	assert(t, NextTokenElement(nil) == nil)

	root, i := buildTree(t, "(1st foo) (2nd (nested bar baz)) (3rd)")
	first := i["1st"]
	second := i["2nd"]
	third := i["3rd"]
	foo := i["foo"]
	bar := i["bar"]
	baz := i["baz"]

	assert(t, PrevTokenElement(root) == nil)
	assert(t, PrevTokenElement(first) == nil)
	assert(t, PrevTokenElement(second) == foo)
	assert(t, PrevTokenElement(third) == baz)
	assert(t, PrevTokenElement(foo) == nil)
	assert(t, PrevTokenElement(bar) == foo)
	assert(t, PrevTokenElement(baz) == bar)
}

func TestChildren(t *testing.T) {
	assert(t, len(Children(nil)) == 0)

	src := "(foo) (bar baz (qux (x)))"
	children := "(foo) (bar)"
	root, i := buildTree(t, src)

	matchNodes(t, children, Children(root)...)
	matchNodes(t, "", Children(i["foo"])...)
	matchNodes(t, "baz (qux)", Children(i["bar"])...)
	matchNodes(t, "", Children(i["baz"])...)
	matchNodes(t, "(x)", Children(i["qux"])...)
}

func TestDetach(t *testing.T) {
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

func TestReplace(t *testing.T) {
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

func TestAppendSibling(t *testing.T) {
	AppendSibling(nil, nil)

	root, i := buildTree(t, "(1st) (2nd)")
	first := i["1st"]
	second := i["2nd"]
	re := &nodeElement{}

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

func TestPrependSibling(t *testing.T) {
	PrependSibling(nil, nil)

	root, i := buildTree(t, "(1st) (2nd)")
	first := i["1st"]
	second := i["2nd"]
	re := &nodeElement{}

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

func TestAppendChild(t *testing.T) {
	AppendChild(nil, nil)

	root := &nodeElement{}
	re := &nodeElement{}
	re2 := &nodeElement{}

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
