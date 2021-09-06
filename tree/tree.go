package tree

import (
	"errors"
	"github.com/ava12/llx/parser"

	"github.com/ava12/llx/lexer"
)

type StringWriter interface {
	WriteString (string) (int, error)
}

type Node interface {
	IsNonTerm () bool
	TypeName () string
	Token () *lexer.Token
	Parent () NonTermNode
	Prev () Node
	Next () Node
	SetParent (NonTermNode)
	SetPrev (Node)
	SetNext (Node)
	Pos () lexer.SourcePos
}

type NonTermNode interface {
	Node
	FirstChild () Node
	LastChild () Node
	SetFirstChild (Node)
	AppendChild (Node)
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
	if n == nil || !n.IsNonTerm() {
		return nil
	}

	nn := n.(NonTermNode)
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
	if parent == nil || !parent.IsNonTerm() {
		return 0
	}

	c := parent.(NonTermNode).FirstChild()
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
	if n == nil || !n.IsNonTerm() {
		return n
	}

	n = n.(NonTermNode).FirstChild()
	for n != nil && n.IsNonTerm() {
		nn := FirstTokenNode(n)
		if nn != nil {
			return nn
		}

		n = n.Next()
	}

	return n
}

func LastTokenNode (n Node) Node {
	if n == nil || !n.IsNonTerm() {
		return n
	}

	n = n.(NonTermNode).LastChild()
	for n != nil && n.IsNonTerm() {
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
	if n == nil || !n.IsNonTerm() {
		return nil
	}

	res := make([]Node, 0)
	c := n.(NonTermNode).FirstChild()
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

	np := n.Prev()
	nn := n.Next()

	if np == nil {
		p := n.Parent()
		if p != nil {
			p.SetFirstChild(nn)
		}
	} else {
		np.SetNext(nn)
		n.SetPrev(nil)
	}
	if nn != nil {
		nn.SetPrev(np)
		n.SetNext(nil)
	}
	n.SetParent(nil)
}

func Replace (old, n Node) {
	if n == nil || old == nil {
		Detach(old)
		return
	}

	pa := old.Parent()
	pr := old.Prev()
	ne := old.Next()
	Detach(old)
	Detach(n)
	if pr != nil {
		AppendSibling(pr, n)
	} else if ne != nil {
		PrependSibling(ne, n)
	} else {
		n.SetParent(pa)
		pa.SetFirstChild(n)
	}
}

func AppendSibling (prev, node Node) {
	if node == nil || prev == nil {
		return
	}

	Detach(node)
	next := prev.Next()
	node.SetParent(prev.Parent())
	node.SetPrev(prev)
	node.SetNext(next)
	prev.SetNext(node)
	if next != nil {
		next.SetPrev(node)
	}
}

func PrependSibling (next, node Node) {
	if node == nil || next == nil {
		return
	}

	Detach(node)
	prev := next.Prev()
	node.SetParent(next.Parent())
	node.SetPrev(prev)
	node.SetNext(next)
	next.SetPrev(node)
	if prev == nil {
		node.Parent().SetFirstChild(node)
	} else {
		prev.SetNext(node)
	}
}

func AppendChild (parent NonTermNode, node Node) {
	if parent == nil || node == nil {
		return
	}

	Detach(node)
	parent.AppendChild(node)
}

type NodeVisitor func (n Node) (walkChildren, walkSiblings bool)

type WalkMode int
const (
	WalkLtr WalkMode = 0
	WalkRtl WalkMode = 1
)

func Walk (n Node, mode WalkMode, visitor NodeVisitor) {
	if n != nil {
		visitNode(n, visitor, (mode & WalkRtl) != 0)
	}
}

func visitNode (n Node, v NodeVisitor, rtl bool) (visitSiblings bool) {
	vc, vs := v(n)
	if vc && n.IsNonTerm() {
		if rtl {
			n = n.(NonTermNode).LastChild()
			for n != nil && vc {
				vc = visitNode(n, v, true)
				n = n.Prev()
			}
		} else {
			n = n.(NonTermNode).FirstChild()
			for n != nil && vc {
				vc = visitNode(n, v, false)
				n = n.Next()
			}
		}
	}

	return vs
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

func (s *Selector) Search (nf NodeFilter, deepSearch bool) *Selector {
	return s.Use(func (n Node) []Node {
		res := make([]Node, 0)
		visitNode(n, func (nn Node) (vc, vs bool) {
			if nf(nn) {
				res = append(res, nn)
				return deepSearch, true
			} else {
				return true, true
			}
		}, false)
		return res
	})
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
		if n.IsNonTerm() {
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

func Any (nss ...NodeExtractor) NodeExtractor {
	return func (n Node) (res []Node) {
		for _, ns := range nss {
			res = ns(n)
			if len(res) > 0 {
				break
			}
		}
		return
	}
}

func All (nss ...NodeExtractor) NodeExtractor {
	return func (n Node) (res []Node) {
		for _, ns := range nss {
			res = append(res, ns(n) ...)
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
	parent     NonTermNode
	prev, next Node
	token      *lexer.Token
}

func NewTokenNode (t *lexer.Token) Node {
	return &tokenNode{token: t}
}

func (tn *tokenNode) IsNonTerm () bool {
	return false
}

func (tn *tokenNode) TypeName () string {
	return tn.token.TypeName()
}

func (tn *tokenNode) Parent () NonTermNode {
	return tn.parent
}

func (tn *tokenNode) Prev () Node {
	return tn.prev
}

func (tn *tokenNode) Next () Node {
	return tn.next
}

func (tn *tokenNode) Pos () lexer.SourcePos {
	return tn.token
}

func (tn *tokenNode) Token () *lexer.Token {
	return tn.token
}

func (tn *tokenNode) SetParent (p NonTermNode) {
	tn.parent = p
}

func (tn *tokenNode) SetPrev (p Node) {
	tn.prev = p
}

func (tn *tokenNode) SetNext (n Node) {
	tn.next = n
}

type nonTermNode struct {
	typeName              string
	token                 *lexer.Token
	parent                NonTermNode
	prev, next            Node
	firstChild, lastChild Node
}

func NewNonTermNode (typeName string, tok *lexer.Token) NonTermNode {
	return &nonTermNode{typeName: typeName, token: tok}
}

func (ntn *nonTermNode) IsNonTerm () bool {
	return true
}

func (ntn *nonTermNode) TypeName () string {
	return ntn.typeName
}

func (ntn *nonTermNode) Token () *lexer.Token {
	return ntn.token
}

func (ntn *nonTermNode) Parent () NonTermNode {
	return ntn.parent
}

func (ntn *nonTermNode) FirstChild () Node {
	return ntn.firstChild
}

func (ntn *nonTermNode) LastChild () Node {
	return ntn.lastChild
}

func (ntn *nonTermNode) Prev () Node {
	return ntn.prev
}

func (ntn *nonTermNode) Next () Node {
	return ntn.next
}

func (ntn *nonTermNode) SetParent (p NonTermNode) {
	ntn.parent = p
}

func (ntn *nonTermNode) SetFirstChild (c Node) {
	ntn.firstChild = c
	if ntn.lastChild == nil {
		ntn.lastChild = c
	}
	if c != nil {
		c.SetParent(ntn)
	}
}

func (ntn *nonTermNode) AppendChild (c Node) {
	if ntn.firstChild == nil {
		ntn.SetFirstChild(c)
	} else {
		AppendSibling(ntn.lastChild, c)
		ntn.lastChild = c
	}
}

func (ntn *nonTermNode) SetPrev (p Node) {
	ntn.prev = p
}

func (ntn *nonTermNode) SetNext (n Node) {
	ntn.next = n
}

func (ntn *nonTermNode) Pos () lexer.SourcePos {
	if ntn.firstChild == nil {
		return nil
	} else {
		return ntn.firstChild.Pos()
	}
}

type HookInstance struct {
	node NonTermNode
}

func NewHookInstance (typeName string, tok *lexer.Token) *HookInstance {
	return &HookInstance{NewNonTermNode(typeName, tok)}
}

func (hi *HookInstance) HandleNonTerm (nonTerm string, result interface{}) error {
	node, is := result.(Node)
	if !is {
		return errors.New("non-terminal " + nonTerm + " is not a tree.Node")
	}

	hi.node.AppendChild(node)
	return nil
}

func (hi *HookInstance) HandleToken (token *lexer.Token) error {
	hi.node.AppendChild(NewTokenNode(token))
	return nil
}

func (hi *HookInstance) EndNonTerm () (result interface{}, e error) {
	return hi.node, nil
}

func NonTermHook (nonTerm string, tok *lexer.Token, pc *parser.ParseContext) (parser.NonTermHookInstance, error) {
	return NewHookInstance(nonTerm, tok), nil
}
