// Package tree provides basic functions for building, manipulating, and traversing syntax trees.
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
	// Prev returns previous sibling for element, nil for first child or tree root.
	Prev () Element
	// Next returns the next sibling for element, nil for last child or tree root.
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

// FirstTokenElement returns element itself if it's not a node
// or returns the first token element captured by node or its descendant nodes.
// Returns nil if neither node nor its descendants contain token elements.
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

// LastTokenElement returns element itself if it's not a node
// or returns the last token element captured by node or its descendant nodes.
// Returns nil if neither node nor its descendants contain token elements. 
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

// NextTokenElement returns the first token element after given one, if there are any.
// I.e. the first token element in the next sibling or its parent's next sibling etc.
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

// PrevTokenElement returns the last token element before given one, if there are any.
// I.e. the last token element in the previous sibling or its parent's previous sibling etc.
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

// Children returns child elements (if there are any) or nil if given element is not a node.
// Child elements are returned in left-to-right order.
func Children (n Element) []Element {
	if n == nil || !n.IsNode() {
		return nil
	}

	var res []Element
	c := n.(NodeElement).FirstChild()
	for c != nil {
		res = append(res, c)
		c = c.Next()
	}
	return res
}


// Detach removes element from tree. Parent and sibling references are removed, but descendants are kept.
// Does nothing if the element is the tree root.
func Detach (n Element) {
	if n == nil || n.Parent() == nil {
		return
	}

	n.Parent().RemoveChild(n)
}

// Replace replaces old element with new one or simply removes old element if the new one is nil.
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

// AppendSibling places new element after target one.
// Target becomes new element's previous sibling, target's next sibling (if any) becomes new element's next sibling.
// Does nothing if either new or target element is nil.
func AppendSibling (prev, el Element) {
	if el == nil || prev == nil {
		return
	}

	Detach(el)
	next := prev.Next()
	parent := prev.Parent()
	if parent == nil {
		el.SetPrev(prev)
		el.SetNext(next)
		prev.SetNext(el)
		if next != nil {
			next.SetPrev(el)
		}
	} else {
		parent.AddChild(el, next)
	}
}

// PrependSibling places new element before target one.
// Target becomes new element's next sibling, target's previous sibling (if any) becomes new element's previous sibling.
// Does nothing if either new or target element is nil.
func PrependSibling (next, el Element) {
	if el == nil || next == nil {
		return
	}

	Detach(el)
	parent := next.Parent()
	if parent == nil {
		prev := next.Prev()
		el.SetPrev(prev)
		el.SetNext(next)
		next.SetPrev(el)
		if prev != nil {
			prev.SetNext(el)
		}
	} else {
		parent.AddChild(el, next)
	}
}

// AppendChild places new element as a child of target one.
// New element becomes target's last child.
func AppendChild (parent NodeElement, el Element) {
	if parent == nil || el == nil {
		return
	}

	Detach(el)
	parent.AddChild(el, nil)
}

// WalkerFlags instruct walker which parts of subtree must be skipped.
type WalkerFlags = int
const (
	WalkerStop = 1 << iota // Stop traversing.
	WalkerSkipChildren     // Skip traversing child subtrees of the last fetched element. Ignored if no elements fetched yet.
	WalkerSkipSiblings     // Skip sibling subtrees of the last fetched element. Ignored if no elements fetched yet.
)

// WalkMode defines subtree traversing mode.
type WalkMode int
const (
	WalkLtr WalkMode = 0 // Left to right, parent first, depth-first.
	WalkRtl WalkMode = 1 // Right to left, parent first, depth-first.
)

// Visitor processes element fetched by walker and responds which parts of subtree must be skipped on next step.
type Visitor func (n Element) WalkerFlags

// Walker traverses given subtree in specified order.
// References given subtree, not reusable.
// Subtree should not be modified while walker is in use.
type Walker struct {
	root, current Element
	flagStack     []WalkerFlags
	mode          WalkMode
}

// NewWalker creates walker for given subtree.
func NewWalker (root Element, m WalkMode) *Walker {
	return &Walker{root: root, mode: m}
}

// Step returns the next element omitting specified parts of subtree.
// Returns nil if WalkerStop is passed or traversal is finished.
// When this method returns nil all future calls wil return nil.
func (w *Walker) Step (f WalkerFlags) Element {
	if (f & WalkerStop) != 0 {
		w.root = nil
		w.flagStack = nil
	}

	if w.root == nil {
		return nil
	}

	if w.current == nil {
		w.current = w.root
		return w.current
	}

	n := w.current
	rtl := (w.mode & WalkRtl) != 0
	if n.IsNode() && (f & WalkerSkipChildren) == 0 {
		if rtl {
			n = n.(NodeElement).LastChild()
		} else {
			n = n.(NodeElement).FirstChild()
		}
		if n != nil {
			w.pushFlags(f)
			w.current = n
			return n
		}
	}

	for w.current != w.root {
		if (f & WalkerSkipSiblings) == 0 {
			if rtl {
				n = w.current.Prev()
			} else {
				n = w.current.Next()
			}
			if n != nil {
				w.current = n
				return n
			}
		}

		n = w.current.Parent()
		if n == nil || len(w.flagStack) < 2 {
			break
		}

		f = w.popFlags()
		w.current = n
	}

	w.root = nil
	w.flagStack = nil
	return nil
}

// Next returns next element, same as Step(0).
func (w *Walker) Next () Element {
	return w.Step(0)
}

func (w *Walker) pushFlags (f WalkerFlags) {
	w.flagStack = append(w.flagStack, f &^ WalkerSkipChildren)
}

func (w *Walker) popFlags () (f WalkerFlags) {
	l := len(w.flagStack) - 1
	f = w.flagStack[l]
	w.flagStack = w.flagStack[: l]
	return
}

// Walk traverses subtree using Step().
// Visitor is called for each fetched element.
func (w *Walker) Walk (visitor Visitor) {
	flags := 0
	el := w.Step(flags)
	for el != nil {
		flags = visitor(el)
		if (flags & WalkerStop) != 0 {
			return
		}

		el = w.Step(flags)
	}
}

// Walk traverses given subtree in specified order calling visitor for each fetched element.
func Walk (root Element, mode WalkMode, visitor Visitor) {
	NewWalker(root, mode).Walk(visitor)
}

// Filter examines given non-nil element and decides whether it is accepted and must be kept in element list (true)
// or rejected and must be removed from list (false).
type Filter func (n Element) bool

// Extractor returns a list of elements related to given non-nil element.
// Result must not contain nil elements.
type Extractor func (n Element) []Element

// Selector transforms list of elements to another list using provided transformers (extractors and filters).
// Contains list of transformers, configurable and reusable.
type Selector struct {
	extractors []Extractor
	unique bool
}

// NewSelector creates selector with empty list of transformers.
func NewSelector () *Selector {
	return &Selector{}
}

// Apply applies stored list of transformers to passed list of elements and returns output.
// Transformers are used in FIFO order, output is used as an input for the next transformer.
func (s *Selector) Apply (input ...Element) []Element {
	res := make([]Element, 0)
	var index map[Element]struct{}
	if s.unique {
		index = make(map[Element]struct{})
	}
	hasExtractors := len(s.extractors) > 0

	for i, n := range input {
		if n == nil {
			continue
		}

		var ns []Element
		if hasExtractors {
			ns = selectNodes(input[i : i + 1], s.extractors)
		} else {
			ns = input[i : i + 1]
		}

		if s.unique {
			for _, tn := range ns {
				_, has := index[tn]
				if !has {
					index[tn] = struct{}{}
					res = append(res, tn)
				}
			}
		} else {
			res = append(res, ns...)
		}
	}

	return res
}

func selectNodes (ns []Element, nss []Extractor) []Element {
	res := make([]Element, 0)
	s := nss[0]
	nss = nss[1 :]
	goDeeper := len(nss) > 0
	for _, n := range ns {
		if goDeeper {
			res = append(res, selectNodes(s(n), nss)...)
		} else {
			res = append(res, s(n)...)
		}
	}
	return res
}

// Unique instructs selector to remove duplicated elements from output of Apply method, by default output is returned as is.
// Modifies selector, chainable.
func (s *Selector) Unique () *Selector {
	s.unique = true
	return s
}

// Extract adds extractor to transformer list.
// Modifies selector, chainable.
func (s *Selector) Extract (ex Extractor) *Selector {
	if ex != nil {
		s.extractors = append(s.extractors, ex)
	}
	return s
}

// Filter adds filter to transformer list.
// Modifies selector, chainable.
func (s *Selector) Filter (nf Filter) *Selector {
	return s.Extract(func (n Element) []Element {
		if nf(n) {
			return []Element{n}
		} else {
			return nil
		}
	})
}

func (s *Selector) search (nf Filter, deepSearch bool) *Selector {
	flags := 0
	if !deepSearch {
		flags = WalkerSkipChildren
	}
	return s.Extract(func (n Element) []Element {
		f := 0
		res := make([]Element, 0)
		w := NewWalker(n, WalkLtr)
		for {
			nn := w.Step(f)
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

// Search adds transformer that treats input element as a subtree root and traverses this subtree
// extracting elements that are accepted by given filter.
// Extracted elements are not traversed, so if accepted element contains acceptable descendants,
// those descendants will not be output.
// Modifies selector, chainable.
func (s *Selector) Search (nf Filter) *Selector {
	return s.search(nf, false)
}

// DeepSearch adds transformer that treats input element as a subtree root and traverses this subtree
// extracting elements that are accepted by given filter.
// Extracted elements are also traversed, so if accepted element contains acceptable descendants,
// those descendants will be output as well.
// Modifies selector, chainable.
func (s *Selector) DeepSearch (nf Filter) *Selector {
	return s.search(nf, true)
}

// IsNot creates filter that inverts result of given filter (i.e. rejects accepted element and accepts rejected).
func IsNot (f Filter) Filter {
	return func (n Element) bool {
		return !f(n)
	}
}

// IsAny creates filter that accepts element if it is accepted by any of given filters and rejects otherwise.
func IsAny (fs ...Filter) Filter {
	return func (n Element) bool {
		for _, f := range fs {
			if f(n) {
				return true
			}
		}
		return false
	}
}

// IsAll creates filter that accepts element if it is accepted by all of given filters and rejects otherwise.
func IsAll (fs ...Filter) Filter {
	return func (n Element) bool {
		for _, f := range fs {
			if !f(n) {
				return false
			}
		}
		return true
	}
}

// IsA creates filter that accepts elements with given type names and rejects others.
func IsA (names ... string) Filter {
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

// IsALiteral creates filter that accepts token elements with given content and rejects others.
func IsALiteral (texts ... string) Filter {
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


// Any creates extractor that applies given extractors to input element in order until it gets non-empty output.
// Returns first non-empty output or nil if all outputs were empty.
func Any (nes ...Extractor) Extractor {
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

// All creates extractor that applies all given extractors to input element.
// Returns merged outputs, may contain duplicate elements.
func All (nes ...Extractor) Extractor {
	return func (n Element) (res []Element) {
		for _, ne := range nes {
			res = append(res, ne(n) ...)
		}
		return
	}
}

// Nth creates extractor that applies given extractor to input element and returns
// output elements with given indexes (if present).
func Nth(ex Extractor, indexes... int) Extractor {
	return func (el Element) []Element {
		var res []Element
		els := ex(el)
		l := len(els)
		for _, i := range indexes {
			if i >= 0 && i < l {
				res = append(res, els[i])
			}
		}
		return res
	}
}

// Ancestors returns ancestors of given element in closest-to-farthest order,
// i.e. the first output element is given element's parent and the last one is the tree root.
func Ancestors (el Element) []Element {
	if el == nil {
		return nil
	}

	var res []Element
	for {
		el = el.Parent()
		if el == nil {
			break
		}

		res = append(res, el)
	}
	return res
}

// PrevSiblings returns preceding siblings of given element in closest-to-farthest order,
// i.e. the first output element is given element's previous sibling and the last one is the first sibling.
func PrevSiblings(el Element) []Element {
	if el == nil {
		return nil
	}

	var res []Element
	for {
		el = el.Prev()
		if el == nil {
			break
		}

		res = append(res, el)
	}
	return res
}

// NextSiblings returns following siblings of given element in closest-to-farthest order,
// i.e. the first output element is given element's next sibling and the last one is the last sibling.
func NextSiblings(el Element) []Element {
	if el == nil {
		return nil
	}

	var res []Element
	for {
		el = el.Next()
		if el == nil {
			break
		}

		res = append(res, el)
	}
	return res
}


type tokenElement struct {
	parent     NodeElement
	prev, next Element
	token      *lexer.Token
}

// NewTokenElement creates token element for given token.
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

// NewNodeElement creates node element of given type with given initial token, token may be nil.
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

// NodeHook implements parser.NodeHook and builds syntax tree.
// Intended to be used as node hook for parser.AnyNode.
func NodeHook (node string, tok *lexer.Token, pc *parser.ParseContext) (parser.NodeHookInstance, error) {
	return NewHookInstance(node, tok), nil
}
