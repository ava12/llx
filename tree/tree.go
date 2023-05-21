package tree

import (
	"errors"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/source"
)

type StringWriter interface {
	WriteString (string) (int, error)
}

type Node interface {
	IsNode () bool
	TypeName () string
	Token () *lexer.Token
	Parent () NodeNode
	Prev () Node
	Next () Node
	SetParent (NodeNode)
	SetPrev (Node)
	SetNext (Node)
	Pos () source.Pos
}

type NodeNode interface {
	Node
	FirstChild () Node
	LastChild () Node
	AddChild (n, before Node)
	RemoveChild (Node)
}


func Ancestor (n Node, level int) Node {
	for n != nil && level >= 0 {
		n = n.Parent()
		level--
	}
	return n
}

func NodeLevel (n Node) (l int) {
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

func SiblingIndex (n Node) (i int) {
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

func NthChild (n Node, i int) Node {
	if n == nil || !n.IsNode() {
		return nil
	}

	nn := n.(NodeNode)
	var c Node
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

func NthSibling (n Node, i int) Node {
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

func NumOfChildren (parent Node, levels int) int {
	if parent == nil || !parent.IsNode() {
		return 0
	}

	c := parent.(NodeNode).FirstChild()
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

func FirstTokenNode (n Node) Node {
	if n == nil || !n.IsNode() {
		return n
	}

	n = n.(NodeNode).FirstChild()
	for n != nil && n.IsNode() {
		nn := FirstTokenNode(n)
		if nn != nil {
			return nn
		}

		n = n.Next()
	}

	return n
}

func LastTokenNode (n Node) Node {
	if n == nil || !n.IsNode() {
		return n
	}

	n = n.(NodeNode).LastChild()
	for n != nil && n.IsNode() {
		nn := LastTokenNode(n)
		if nn != nil {
			return nn
		}

		n = n.Prev()
	}

	return n
}

func NextTokenNode (n Node) Node {
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

	return FirstTokenNode(nn)
}

func PrevTokenNode (n Node) Node {
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

	return LastTokenNode(nn)
}

func Children (n Node) []Node {
	if n == nil || !n.IsNode() {
		return nil
	}

	res := make([]Node, 0)
	c := n.(NodeNode).FirstChild()
	for c != nil {
		res = append(res, c)
		c = c.Next()
	}
	return res
}


func Detach (n Node) {
	if n == nil || n.Parent() == nil {
		return
	}

	n.Parent().RemoveChild(n)
}

func Replace (old, n Node) {
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

func AppendSibling (prev, node Node) {
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

func PrependSibling (next, node Node) {
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

func AppendChild (parent NodeNode, node Node) {
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
	root, current Node
	flagStack     []WalkerFlags
	mode          WalkMode
}

func NewIterator (n Node, m WalkMode) *Iterator {
	return &Iterator{root: n, mode: m}
}

func (it *Iterator) Step (f WalkerFlags) Node {
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
			n = n.(NodeNode).LastChild()
		} else {
			n = n.(NodeNode).FirstChild()
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

func (it *Iterator) Next () Node {
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


type NodeVisitor func (n Node) WalkerFlags

func Walk (n Node, mode WalkMode, visitor NodeVisitor) {
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


type NodeFilter func (n Node) bool
type NodeExtractor func (n Node) []Node

type NodeSelector func (n Node) []Node

type Selector struct {
	selectors []NodeSelector
}

func NewSelector () *Selector {
	return &Selector{}
}

func (s *Selector) Apply (input ... Node) []Node {
	res := make([]Node, 0)
	index := make(map[Node]bool)
	hasTransformers := (len(s.selectors) > 0)

	for i, n := range input {
		if n == nil {
			continue
		}

		var ns []Node
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

func selectNodes (ns []Node, nss []NodeSelector) []Node {
	res := make([]Node, 0)
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
	return s.Use(func (n Node) []Node {
		if nf(n) {
			return []Node{n}
		} else {
			return nil
		}
	})
}

func (s *Selector) Extract (ne NodeExtractor) *Selector {
	return s.Use(func (n Node) []Node {
		return ne(n)
	})
}

func (s *Selector) search (nf NodeFilter, deepSearch bool) *Selector {
	flags := 0
	if !deepSearch {
		flags = WalkerSkipChildren
	}
	return s.Use(func (n Node) []Node {
		f := 0
		res := make([]Node, 0)
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
	return func (n Node) bool {
		return !f(n)
	}
}

func IsAny (fs ... NodeFilter) NodeFilter {
	return func (n Node) bool {
		for _, f := range fs {
			if f(n) {
				return true
			}
		}
		return false
	}
}

func IsAll (fs ... NodeFilter) NodeFilter {
	return func (n Node) bool {
		for _, f := range fs {
			if !f(n) {
				return false
			}
		}
		return true
	}
}

func IsA (names ... string) NodeFilter {
	return func (n Node) bool {
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
	return func (n Node) bool {
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
		return func(n Node) bool {
			it := NewIterator(n, WalkLtr)
			for nn := it.Next(); nn != nil; nn = it.Next() {
				if nf == nil || nf(nn) {
					return true
				}
			}
			return false
		}
	} else {
		return func (n Node) bool {
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
	return func (n Node) (res []Node) {
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
	return func (n Node) (res []Node) {
		for _, ne := range nes {
			res = append(res, ne(n) ...)
		}
		return
	}
}

func Ancestors (levels ... int) NodeExtractor {
	return func (n Node) []Node {
		res := make([]Node, 0)
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
	return func (n Node) []Node {
		res := make([]Node, 0)
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
	return func (n Node) []Node {
		res := make([]Node, 0)
		for _, i := range indexes {
			nn := NthSibling(n, i)
			if nn != nil {
				res = append(res, nn)
			}
		}
		return res
	}
}


type tokenNode struct {
	parent     NodeNode
	prev, next Node
	token      *lexer.Token
}

func NewTokenNode (t *lexer.Token) Node {
	return &tokenNode{token: t}
}

func (tn *tokenNode) IsNode() bool {
	return false
}

func (tn *tokenNode) TypeName () string {
	return tn.token.TypeName()
}

func (tn *tokenNode) Parent () NodeNode {
	return tn.parent
}

func (tn *tokenNode) Prev () Node {
	return tn.prev
}

func (tn *tokenNode) Next () Node {
	return tn.next
}

func (tn *tokenNode) Pos () source.Pos {
	return tn.token.Pos()
}

func (tn *tokenNode) Token () *lexer.Token {
	return tn.token
}

func (tn *tokenNode) SetParent (p NodeNode) {
	tn.parent = p
}

func (tn *tokenNode) SetPrev (p Node) {
	tn.prev = p
}

func (tn *tokenNode) SetNext (n Node) {
	tn.next = n
}

type nodeNode struct {
	typeName              string
	token                 *lexer.Token
	parent                NodeNode
	prev, next            Node
	firstChild, lastChild Node
}

func NewNodeNode (typeName string, tok *lexer.Token) NodeNode {
	return &nodeNode{typeName: typeName, token: tok}
}

func (ntn *nodeNode) IsNode() bool {
	return true
}

func (ntn *nodeNode) TypeName () string {
	return ntn.typeName
}

func (ntn *nodeNode) Token () *lexer.Token {
	return ntn.token
}

func (ntn *nodeNode) Parent () NodeNode {
	return ntn.parent
}

func (ntn *nodeNode) FirstChild () Node {
	return ntn.firstChild
}

func (ntn *nodeNode) LastChild () Node {
	return ntn.lastChild
}

func (ntn *nodeNode) Prev () Node {
	return ntn.prev
}

func (ntn *nodeNode) Next () Node {
	return ntn.next
}

func (ntn *nodeNode) SetParent (p NodeNode) {
	ntn.parent = p
}

func (ntn *nodeNode) AddChild (c, before Node) {
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

func (ntn *nodeNode) RemoveChild (c Node) {
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

func (ntn *nodeNode) SetPrev (p Node) {
	ntn.prev = p
}

func (ntn *nodeNode) SetNext (n Node) {
	ntn.next = n
}

func (ntn *nodeNode) Pos () source.Pos {
	if ntn.firstChild == nil {
		return source.Pos{}
	} else {
		return ntn.firstChild.Pos()
	}
}

type HookInstance struct {
	node NodeNode
}

func NewHookInstance (typeName string, tok *lexer.Token) *HookInstance {
	return &HookInstance{NewNodeNode(typeName, tok)}
}

func (hi *HookInstance) NewNode(node string, token *lexer.Token) error {
	return nil
}

func (hi *HookInstance) HandleNode(name string, result interface{}) error {
	node, is := result.(Node)
	if !is {
		return errors.New("node " + name + " is not a tree.Node")
	}

	hi.node.AddChild(node, nil)
	return nil
}

func (hi *HookInstance) HandleToken (token *lexer.Token) error {
	hi.node.AddChild(NewTokenNode(token), nil)
	return nil
}

func (hi *HookInstance) EndNode() (result interface{}, e error) {
	return hi.node, nil
}

func NodeHook (node string, tok *lexer.Token, pc *parser.ParseContext) (parser.NodeHookInstance, error) {
	return NewHookInstance(node, tok), nil
}
