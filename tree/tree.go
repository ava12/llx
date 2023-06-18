package tree

import (
	"errors"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/source"
)

type Element interface {
	IsNode () bool
	TypeName () string
	Token () *lexer.Token
	Parent () NodeElement
	Prev () Element
	Next () Element
	SetParent (NodeElement)
	SetPrev (Element)
	SetNext (Element)
	Pos () source.Pos
}

type NodeElement interface {
	Element
	FirstChild () Element
	LastChild () Element
	AddChild (n, before Element)
	RemoveChild (Element)
}


func Ancestor (n Element, level int) Element {
	for n != nil && level >= 0 {
		n = n.Parent()
		level--
	}
	return n
}

func NodeLevel (n Element) (l int) {
	if n == nil {
		return
	}

	p := n.Parent()
	for p != nil {
		l++
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

func (tn *tokenElement) IsNode() bool {
	return false
}

func (tn *tokenElement) TypeName () string {
	return tn.token.TypeName()
}

func (tn *tokenElement) Parent () NodeElement {
	return tn.parent
}

func (tn *tokenElement) Prev () Element {
	return tn.prev
}

func (tn *tokenElement) Next () Element {
	return tn.next
}

func (tn *tokenElement) Pos () source.Pos {
	return tn.token.Pos()
}

func (tn *tokenElement) Token () *lexer.Token {
	return tn.token
}

func (tn *tokenElement) SetParent (p NodeElement) {
	tn.parent = p
}

func (tn *tokenElement) SetPrev (p Element) {
	tn.prev = p
}

func (tn *tokenElement) SetNext (n Element) {
	tn.next = n
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

func (ntn *nodeElement) IsNode() bool {
	return true
}

func (ntn *nodeElement) TypeName () string {
	return ntn.typeName
}

func (ntn *nodeElement) Token () *lexer.Token {
	return ntn.token
}

func (ntn *nodeElement) Parent () NodeElement {
	return ntn.parent
}

func (ntn *nodeElement) FirstChild () Element {
	return ntn.firstChild
}

func (ntn *nodeElement) LastChild () Element {
	return ntn.lastChild
}

func (ntn *nodeElement) Prev () Element {
	return ntn.prev
}

func (ntn *nodeElement) Next () Element {
	return ntn.next
}

func (ntn *nodeElement) SetParent (p NodeElement) {
	ntn.parent = p
}

func (ntn *nodeElement) AddChild (c, before Element) {
	if c == nil || (before != nil && before.Parent() != ntn) {
		return
	}

	c.SetParent(ntn)
	if before == nil {
		if ntn.lastChild == nil {
			ntn.firstChild = c
		} else {
			c.SetPrev(ntn.lastChild)
			ntn.lastChild.SetNext(c)
		}
		ntn.lastChild = c
		return
	}

	prev := before.Prev()
	before.SetPrev(c)
	c.SetNext(before)
	c.SetPrev(prev)
	if prev == nil {
		ntn.firstChild = c
	} else {
		prev.SetNext(c)
	}
}

func (ntn *nodeElement) RemoveChild (c Element) {
	if c == nil || c.Parent() != ntn {
		return
	}

	prev := c.Prev()
	next := c.Next()
	c.SetParent(nil)
	c.SetPrev(nil)
	c.SetNext(nil)
	if prev == nil {
		ntn.firstChild = next
	} else {
		prev.SetNext(next)
	}
	if next == nil {
		ntn.lastChild = prev
	} else {
		next.SetPrev(prev)
	}
}

func (ntn *nodeElement) SetPrev (p Element) {
	ntn.prev = p
}

func (ntn *nodeElement) SetNext (n Element) {
	ntn.next = n
}

func (ntn *nodeElement) Pos () source.Pos {
	if ntn.firstChild == nil {
		return source.Pos{}
	} else {
		return ntn.firstChild.Pos()
	}
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
