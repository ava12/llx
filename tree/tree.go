// Package tree provides functions for building, manipulating, and traversing parse trees.
// A tree consists of linked node and token elements, the root is the initial node element.
package tree

import (
	"errors"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
)

// Element represents parse tree element, either a node or a token.
type Element interface {
	// IsNode returns true for node element, false for token element.
	IsNode () bool
	// TypeName returns token or node type name.
	TypeName () string
	// Token returns token for this element (for node it is initial token).
	Token () *lexer.Token
	// Parent returns parent node element, nil for tree root.
	Parent () NodeElement
	// Prev returns previous sibling for element, nil for first child.
	Prev () Element
	// Next returns the next sibling for element, nil for last child.
	Next () Element
	// SetParent sets parent node for element, nil to remove element from tree.
	// Used by SetChild and RemoveChild.
	SetParent (NodeElement)
	// SetPrev sets previous sibling for element, nil to make the first sibling or to remove element from tree.
	// Used by functions manipulating child and sibling elements.
	SetPrev (Element)
	// SetNext sets the next sibling for element, nil to make the last sibling or to remove element from tree.
	// Used by functions manipulating child and sibling elements.
	SetNext (Element)
}

// NodeElement represents parse tree node.
type NodeElement interface {
	Element
	// FirstChild returns first child element or nil if there are no children.
	FirstChild () Element
	// LastChild returns last child element or nil if there are no children.
	LastChild () Element
	// AddChild inserts child element. New child is placed before given one or after the last child.
	// Does nothing if n is nil or given element does not belong to node.
	AddChild (n, before Element)
	// RemoveChild detaches given child element from node.
	// Does nothing if given element is not this node's child.
	RemoveChild (Element)
}


func Ancestor (n Element, level int) Element {
	for n != nil && level >= 0 {
		n = n.Parent()
		level--
	}
	return n
}

func NodeLevel (n Element) (level int) {
	if n == nil {
		return
	}

	p := n.Parent()
	for p != nil {
		level++
		p = p.Parent()
	}
	return
}

func SiblingIndex (n Element) (i int) {
	if n == nil {
		return
	}

	p := n.Prev()
	for p != nil {
		i++
		p = p.Prev()
	}
	return
}

func NthChild (n Element, i int) Element {
	if n == nil || !n.IsNode() {
		return nil
	}

	nn := n.(NodeElement)
	var c Element
	if i >= 0 {
		c = nn.FirstChild()
		for c != nil && i > 0 {
			c = c.Next()
			i--
		}
	} else {
		i++
		c = nn.LastChild()
		for c != nil && i < 0 {
			c = c.Prev()
			i++
		}
	}

	return c
}

func NthSibling (n Element, i int) Element {
	if i < 0 {
		for n != nil && i < 0 {
			n = n.Prev()
			i++
		}
	} else {
		for n != nil && i > 0 {
			n = n.Next()
			i--
		}
	}
	return n
}

const AllLevels = -1

func NumOfChildren (parent Element, levels int) int {
	if parent == nil || !parent.IsNode() {
		return 0
	}

	c := parent.(NodeElement).FirstChild()
	i := 0
	for c != nil {
		i++
		if levels != 0 {
			i += NumOfChildren(c, levels - 1)
		}
		c = c.Next()
	}
	return i
}

func FirstTokenElement (n Element) Element {
	if n == nil || !n.IsNode() {
		return n
	}

	n = n.(NodeElement).FirstChild()
	for n != nil && n.IsNode() {
		nn := FirstTokenElement(n)
		if nn != nil {
			return nn
		}

		n = n.Next()
	}

	return n
}

func LastTokenElement (n Element) Element {
	if n == nil || !n.IsNode() {
		return n
	}

	n = n.(NodeElement).LastChild()
	for n != nil && n.IsNode() {
		nn := LastTokenElement(n)
		if nn != nil {
			return nn
		}

		n = n.Prev()
	}

	return n
}

func NextTokenElement (n Element) Element {
	if n == nil {
		return nil
	}

	nn := n.Next()
	for nn == nil {
		n = n.Parent()
		if n == nil {
			return nil
		}

		nn = n.Next()
	}

	return FirstTokenElement(nn)
}

func PrevTokenElement (n Element) Element {
	if n == nil {
		return nil
	}

	nn := n.Prev()
	for nn == nil {
		n = n.Parent()
		if n == nil {
			return nil
		}

		nn = n.Prev()
	}

	return LastTokenElement(nn)
}

func Children (n Element) []Element {
	if n == nil || !n.IsNode() {
		return nil
	}

	res := make([]Element, 0)
	c := n.(NodeElement).FirstChild()
	for c != nil {
		res = append(res, c)
		c = c.Next()
	}
	return res
}


func Detach (n Element) {
	if n == nil || n.Parent() == nil {
		return
	}

	n.Parent().RemoveChild(n)
}

func Replace (old, n Element) {
	if n == nil || old == nil {
		Detach(old)
		return
	}

	pa := old.Parent()
	ne := old.Next()
	Detach(old)
	Detach(n)
	pa.AddChild(n, ne)
}

func AppendSibling (prev, node Element) {
	if node == nil || prev == nil {
		return
	}

	Detach(node)
	next := prev.Next()
	parent := prev.Parent()
	if parent == nil {
		node.SetPrev(prev)
		node.SetNext(next)
		prev.SetNext(node)
		if next != nil {
			next.SetPrev(node)
		}
	} else {
		parent.AddChild(node, next)
	}
}

func PrependSibling (next, node Element) {
	if node == nil || next == nil {
		return
	}

	Detach(node)
	parent := next.Parent()
	if parent == nil {
		prev := next.Prev()
		node.SetPrev(prev)
		node.SetNext(next)
		next.SetPrev(node)
		if prev != nil {
			prev.SetNext(node)
		}
	} else {
		parent.AddChild(node, next)
	}
}

func AppendChild (parent NodeElement, node Element) {
	if parent == nil || node == nil {
		return
	}

	Detach(node)
	parent.AddChild(node, nil)
}


type WalkerFlags = int
const (
	WalkerStop = 1 << iota
	WalkerSkipChildren
	WalkerSkipSiblings
)

type WalkMode int
const (
	WalkLtr WalkMode = 0
	WalkRtl WalkMode = 1
)

type Iterator struct {
	root, current Element
	flagStack     []WalkerFlags
	mode          WalkMode
}

func NewIterator (n Element, m WalkMode) *Iterator {
	return &Iterator{root: n, mode: m}
}

func (it *Iterator) Step (f WalkerFlags) Element {
	if (f & WalkerStop) != 0 {
		it.root = nil
		it.flagStack = nil
	}

	if it.root == nil {
		return nil
	}

	if it.current == nil {
		it.current = it.root
		return it.current
	}

	n := it.current
	rtl := (it.mode & WalkRtl) != 0
	if n.IsNode() && (f & WalkerSkipChildren) == 0 {
		if rtl {
			n = n.(NodeElement).LastChild()
		} else {
			n = n.(NodeElement).FirstChild()
		}
		if n != nil {
			it.pushFlags(f)
			it.current = n
			return n
		}
	}

	for it.current != it.root {
		if (f & WalkerSkipSiblings) == 0 {
			if rtl {
				n = it.current.Prev()
			} else {
				n = it.current.Next()
			}
			if n != nil {
				it.current = n
				return n
			}
		}

		n = it.current.Parent()
		if n == nil || len(it.flagStack) < 2 {
			break
		}

		f = it.popFlags()
		it.current = n
	}

	it.root = nil
	it.flagStack = nil
	return nil
}

func (it *Iterator) Next () Element {
	return it.Step(0)
}

func (it *Iterator) pushFlags (f WalkerFlags) {
	it.flagStack = append(it.flagStack, f &^ WalkerSkipChildren)
}

func (it *Iterator) popFlags () (f WalkerFlags) {
	l := len(it.flagStack) - 1
	f = it.flagStack[l]
	it.flagStack = it.flagStack[: l]
	return
}


type NodeVisitor func (n Element) WalkerFlags

func Walk (n Element, mode WalkMode, visitor NodeVisitor) {
	flags := 0
	it := NewIterator(n, mode)
	n = it.Step(flags)
	for n != nil {
		flags = visitor(n)
		if (flags & WalkerStop) != 0 {
			return
		}

		n = it.Step(flags)
	}
}


type NodeFilter func (n Element) bool
type NodeExtractor func (n Element) []Element

type NodeSelector func (n Element) []Element

type Selector struct {
	selectors []NodeSelector
}

func NewSelector () *Selector {
	return &Selector{}
}

func (s *Selector) Apply (input ...Element) []Element {
	res := make([]Element, 0)
	index := make(map[Element]bool)
	hasTransformers := (len(s.selectors) > 0)

	for i, n := range input {
		if n == nil {
			continue
		}

		var ns []Element
		if hasTransformers {
			ns = selectNodes(input[i : i + 1], s.selectors)
		} else {
			ns = input[i : i + 1]
		}

		for _, tn := range ns {
			if !index[tn] {
				index[tn] = true
				res = append(res, tn)
			}
		}
	}

	return res
}

func selectNodes (ns []Element, nss []NodeSelector) []Element {
	res := make([]Element, 0)
	s := nss[0]
	nss = nss[1 :]
	goDeeper := (len(nss) > 0)
	for _, n := range ns {
		if goDeeper {
			res = append(res, selectNodes(s(n), nss)...)
		} else {
			res = append(res, s(n)...)
		}
	}
	return res
}

func (s *Selector) Use (ns NodeSelector) *Selector {
	if ns != nil {
		s.selectors = append(s.selectors, ns)
	}
	return s
}

func (s *Selector) Filter (nf NodeFilter) *Selector {
	return s.Use(func (n Element) []Element {
		if nf(n) {
			return []Element{n}
		} else {
			return nil
		}
	})
}

func (s *Selector) Extract (ne NodeExtractor) *Selector {
	return s.Use(func (n Element) []Element {
		return ne(n)
	})
}

func (s *Selector) search (nf NodeFilter, deepSearch bool) *Selector {
	flags := 0
	if !deepSearch {
		flags = WalkerSkipChildren
	}
	return s.Use(func (n Element) []Element {
		f := 0
		res := make([]Element, 0)
		it := NewIterator(n, WalkLtr)
		for {
			nn := it.Step(f)
			if nn == nil {
				break
			}

			if nf(nn) {
				res = append(res, nn)
				f = flags
			} else {
				f = 0
			}
		}
		return res
	})
}

func (s *Selector) Search (nf NodeFilter) *Selector {
	return s.search(nf, false)
}

func (s *Selector) DeepSearch (nf NodeFilter) *Selector {
	return s.search(nf, true)
}


func IsNot (f NodeFilter) NodeFilter {
	return func (n Element) bool {
		return !f(n)
	}
}

func IsAny (fs ... NodeFilter) NodeFilter {
	return func (n Element) bool {
		for _, f := range fs {
			if f(n) {
				return true
			}
		}
		return false
	}
}

func IsAll (fs ... NodeFilter) NodeFilter {
	return func (n Element) bool {
		for _, f := range fs {
			if !f(n) {
				return false
			}
		}
		return true
	}
}

func IsA (names ... string) NodeFilter {
	return func (n Element) bool {
		tn := n.TypeName()
		for _, name := range names {
			if tn == name {
				return true
			}
		}

		return false
	}
}

func IsALiteral (texts ... string) NodeFilter {
	return func (n Element) bool {
		if n.IsNode() {
			return false
		}

		t := n.Token().Text()
		for _, text := range texts {
			if text == t {
				return true
			}
		}

		return false
	}
}

func Has (ne NodeExtractor, nf NodeFilter) NodeFilter {
	if ne == nil {
		return func(n Element) bool {
			it := NewIterator(n, WalkLtr)
			for nn := it.Next(); nn != nil; nn = it.Next() {
				if nf == nil || nf(nn) {
					return true
				}
			}
			return false
		}
	} else {
		return func (n Element) bool {
			ns := ne(n)
			for _, nn := range ns {
				if nf == nil || nf(nn) {
					return true
				}
			}
			return false
		}
	}
}


func Any (nes ...NodeExtractor) NodeExtractor {
	return func (n Element) (res []Element) {
		for _, ne := range nes {
			res = ne(n)
			if len(res) > 0 {
				break
			}
		}
		return
	}
}

func All (nes ...NodeExtractor) NodeExtractor {
	return func (n Element) (res []Element) {
		for _, ne := range nes {
			res = append(res, ne(n) ...)
		}
		return
	}
}

func Ancestors (levels ... int) NodeExtractor {
	return func (n Element) []Element {
		res := make([]Element, 0)
		for _, i := range levels {
			nn := Ancestor(n, i)
			if nn != nil {
				res = append(res, nn)
			}
		}
		return res
	}
}

func NthChildren (indexes ... int) NodeExtractor {
	return func (n Element) []Element {
		res := make([]Element, 0)
		for _, i := range indexes {
			nn := NthChild(n, i)
			if nn != nil {
				res = append(res, nn)
			}
		}
		return res
	}
}

func NthSiblings (indexes ... int) NodeExtractor {
	return func (n Element) []Element {
		res := make([]Element, 0)
		for _, i := range indexes {
			nn := NthSibling(n, i)
			if nn != nil {
				res = append(res, nn)
			}
		}
		return res
	}
}


type tokenElement struct {
	parent     NodeElement
	prev, next Element
	token      *lexer.Token
}

func NewTokenElement (t *lexer.Token) Element {
	return &tokenElement{token: t}
}

func (t *tokenElement) IsNode() bool {
	return false
}

func (t *tokenElement) TypeName () string {
	return t.token.TypeName()
}

func (t *tokenElement) Parent () NodeElement {
	return t.parent
}

func (t *tokenElement) Prev () Element {
	return t.prev
}

func (t *tokenElement) Next () Element {
	return t.next
}

func (t *tokenElement) Token () *lexer.Token {
	return t.token
}

func (t *tokenElement) SetParent (p NodeElement) {
	t.parent = p
}

func (t *tokenElement) SetPrev (p Element) {
	t.prev = p
}

func (t *tokenElement) SetNext (n Element) {
	t.next = n
}

type nodeElement struct {
	typeName              string
	token                 *lexer.Token
	parent                NodeElement
	prev, next            Element
	firstChild, lastChild Element
}

func NewNodeElement (typeName string, tok *lexer.Token) NodeElement {
	return &nodeElement{typeName: typeName, token: tok}
}

func (n *nodeElement) IsNode() bool {
	return true
}

func (n *nodeElement) TypeName () string {
	return n.typeName
}

func (n *nodeElement) Token () *lexer.Token {
	return n.token
}

func (n *nodeElement) Parent () NodeElement {
	return n.parent
}

func (n *nodeElement) FirstChild () Element {
	return n.firstChild
}

func (n *nodeElement) LastChild () Element {
	return n.lastChild
}

func (n *nodeElement) Prev () Element {
	return n.prev
}

func (n *nodeElement) Next () Element {
	return n.next
}

func (n *nodeElement) SetParent (p NodeElement) {
	n.parent = p
}

func (n *nodeElement) AddChild (c, before Element) {
	if c == nil || (before != nil && before.Parent() != n) {
		return
	}

	c.SetParent(n)
	if before == nil {
		if n.lastChild == nil {
			n.firstChild = c
		} else {
			c.SetPrev(n.lastChild)
			n.lastChild.SetNext(c)
		}
		n.lastChild = c
		return
	}

	prev := before.Prev()
	before.SetPrev(c)
	c.SetNext(before)
	c.SetPrev(prev)
	if prev == nil {
		n.firstChild = c
	} else {
		prev.SetNext(c)
	}
}

func (n *nodeElement) RemoveChild (c Element) {
	if c == nil || c.Parent() != n {
		return
	}

	prev := c.Prev()
	next := c.Next()
	c.SetParent(nil)
	c.SetPrev(nil)
	c.SetNext(nil)
	if prev == nil {
		n.firstChild = next
	} else {
		prev.SetNext(next)
	}
	if next == nil {
		n.lastChild = prev
	} else {
		next.SetPrev(prev)
	}
}

func (n *nodeElement) SetPrev (p Element) {
	n.prev = p
}

func (n *nodeElement) SetNext (next Element) {
	n.next = next
}

type HookInstance struct {
	node NodeElement
}

func NewHookInstance (typeName string, tok *lexer.Token) *HookInstance {
	return &HookInstance{NewNodeElement(typeName, tok)}
}

func (hi *HookInstance) NewNode(node string, token *lexer.Token) error {
	return nil
}

func (hi *HookInstance) HandleNode(name string, result interface{}) error {
	node, is := result.(Element)
	if !is {
		return errors.New("node " + name + " is not a tree.Element")
	}

	hi.node.AddChild(node, nil)
	return nil
}

func (hi *HookInstance) HandleToken (token *lexer.Token) error {
	hi.node.AddChild(NewTokenElement(token), nil)
	return nil
}

func (hi *HookInstance) EndNode() (result interface{}, e error) {
	return hi.node, nil
}

func NodeHook (node string, tok *lexer.Token, pc *parser.ParseContext) (parser.NodeHookInstance, error) {
	return NewHookInstance(node, tok), nil
}
