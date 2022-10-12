//go:generate ../../../bin/llxgen grammar.llx
package internal

import (
	"errors"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/source"
	"github.com/ava12/llx/tree"
)

const defaultSectionName = ""

const (
	secNameToken = "sec-name"
	nameToken    = "name"
	valueToken   = "value"
	opToken      = "op"
	nlToken      = "nl"

	defSectionNt = "def-section"
	sectionNt    = "section"
	entryNt      = "entry"
	headerNt     = "header"
	sepNt        = "sep"
)

type Conf struct {
	RootNode tree.NonTermNode
	Sections map[string]*Section
	updated  bool
}

type Section struct {
	Nodes   []tree.NonTermNode
	Entries map[string]*Entry
	Updated bool
}

type Entry struct {
	Node      tree.NonTermNode
	ValueNode tree.Node
	Value     string
}

var (
	secNameSelector, nameSelector, valueSelector *tree.Selector
	secNameRe *regexp.Regexp
)

func init () {
	secNameSelector = tree.NewSelector().Search(tree.IsA(secNameToken))
	nameSelector = tree.NewSelector().Search(tree.IsA(nameToken))
	valueSelector = tree.NewSelector().Search(tree.IsA(valueToken))

	for _, t := range confGrammar.Tokens {
		if t.Name == secNameToken {
			secNameRe = regexp.MustCompile("^" + t.Re + "$")
		}
	}
}

func tokenNode (typ, text string) tree.Node {
	return tree.NewTokenNode(lexer.NewToken(0, typ, text, source.Pos{}))
}

func nlNode () tree.Node {
	return tree.NewTokenNode(lexer.NewToken(0, nlToken, "\n", source.Pos{}))
}

func sepNode () tree.NonTermNode {
	result := tree.NewNonTermNode(sepNt, nil)
	result.AddChild(nlNode(), nil)
	return result
}

func splitName (name string) (sec, val string) {
	i := strings.LastIndex(name, ".")
	if i < 0 {
		val = name
	} else {
		sec = name[: i]
		val = name[i + 1 :]
	}
	return
}

func NewConf (root tree.NonTermNode) *Conf {
	return &Conf{RootNode: root.(tree.NonTermNode), Sections: make(map[string]*Section)}
}

func (c *Conf) Updated () bool {
	if c.updated {
		return true
	}

	for _, s := range c.Sections {
		if s.Updated {
			return true
		}
	}

	return false
}

func (c *Conf) AddSectionNode (n tree.NonTermNode) *Section {
	name := defaultSectionName
	if n.TypeName() != defSectionNt {
		name = secNameSelector.Apply(n)[0].Token().Text()
	}
	result := c.Sections[name]
	if result == nil {
		result = &Section{Nodes: []tree.NonTermNode{n}, Entries: make(map[string]*Entry)}
		c.Sections[name] = result
	}
	return result
}

func (c *Conf) AddSection (name string) *Section {
	result := c.Sections[name]
	if result != nil {
		return result
	}

	var (
		node tree.NonTermNode
		child tree.Node
	)

	if name == defaultSectionName {
		node = tree.NewNonTermNode(defSectionNt, nil)
		child = c.RootNode.LastChild()
		if child == nil {
			c.RootNode.AddChild(node, nil)
		} else if child.TypeName() != defSectionNt {
			child = c.RootNode.FirstChild()
			for child != nil && !child.IsNonTerm() {
				child = child.Next()
			}
			if child == nil {
				c.RootNode.AddChild(node, nil)
			} else {
				node.AddChild(sepNode(), nil)
				tree.PrependSibling(child, node)
			}
		}
	} else {
		node = tree.NewNonTermNode(sectionNt, nil)
		header := tree.NewNonTermNode(headerNt, nil)
		header.AddChild(tokenNode(opToken, "["), nil)
		header.AddChild(tokenNode(secNameToken, name), nil)
		header.AddChild(tokenNode(opToken, "]"), nil)
		header.AddChild(nlNode(), nil)
		node.AddChild(header, nil)
		child = c.RootNode.LastChild()
		if child != nil && child.IsNonTerm() {
			child := child.(tree.NonTermNode)
			if child.LastChild().TypeName() != sepNt {
				child.AddChild(sepNode(), nil)
			}
		}
		c.RootNode.AddChild(node, nil)
	}

	c.updated = true
	return c.AddSectionNode(node)
}

func (c *Conf) RemoveSection (name string) {
	sec := c.Sections[name]
	if sec != nil {
		c.updated = true
		for _, n := range sec.Nodes {
			p := n.Prev()
			if p != nil {
				p = p.(tree.NonTermNode).LastChild()
				if p != nil && p.TypeName() == sepNt {
					tree.Detach(p)
				}
			}
			tree.Detach(n)
		}
		delete(c.Sections, name)
	}
}

func (c *Conf) RemoveEntry (name string) {
	sn, en := splitName(name)
	s := c.Sections[sn]
	if s != nil {
		s.RemoveEntry(en)
	}
}

func (c *Conf) SetEntry (name, value string) {
	sn, en := splitName(name)
	s := c.AddSection(sn)
	s.SetEntry(en, value)
}

func (s *Section) AddEntryNode (n tree.NonTermNode) *Entry {
	var value string
	name := nameSelector.Apply(n)[0].Token().Text()
	result := s.Entries[name]
	if result == nil {
		result = &Entry{}
	} else {
		tree.Detach(result.Node)
		s.Updated = true
	}

	result.Node = n
	valueNodes := valueSelector.Apply(n)
	if len(valueNodes) != 0 {
		result.ValueNode = valueNodes[0]
		value = strings.TrimSpace(valueNodes[0].Token().Text())
	}
	result.Value = value
	s.Entries[name] = result

	return result
}

func (s *Section) SetEntry (name, value string) *Entry {
	result := s.Entries[name]
	var vnode tree.Node

	if result != nil {
		if result.Value == value {
			return result
		}

		if value != "" {
			vnode = tokenNode(valueToken, value)
		}

		s.Updated = true
		if result.ValueNode == nil {
			child := result.Node.FirstChild().Next()
			for child.TypeName() != opToken {
				child = child.Next()
			}
			tree.AppendSibling(child, vnode)
		} else {
			tree.Replace(result.ValueNode, vnode)
		}
		result.ValueNode = vnode
		result.Value = value

		return result
	}

	s.Updated = true
	node := tree.NewNonTermNode(entryNt, nil)
	if value != "" {
		vnode = tokenNode(valueToken, value)
	}
	node.AddChild(tokenNode(nameToken, name), nil)
	node.AddChild(tokenNode(opToken, "="), nil)
	node.AddChild(vnode, nil)
	node.AddChild(nlNode(), nil)
	snode := s.Nodes[len(s.Nodes) - 1]
	last := snode.LastChild()
	for last != nil && (!last.IsNonTerm() || last.TypeName() == sepNt) {
		last = last.Prev()
	}
	if last == nil {
		snode.AddChild(node, snode.FirstChild())
	} else {
		tree.AppendSibling(last, node)
	}

	return s.AddEntryNode(node)
}

func (s *Section) RemoveEntry (name string) {
	entry := s.Entries[name]
	if entry != nil {
		s.Updated = true
		tree.Detach(entry.Node)
		delete(s.Entries, name)
	}
}

func Parse (name string, src *[]byte) (*Conf, error) {
	source.NormalizeNls(src)
	if len(*src) > 0 && (*src)[len(*src) - 1] != '\n' {
		*src = append(*src, '\n')
	}

	queue := source.NewQueue().Append(source.New(name, *src))
	p := parser.New(confGrammar)
	hs := parser.Hooks{
		Tokens: parser.TokenHooks{
			parser.AnyToken: func (*lexer.Token, *parser.ParseContext) (bool, error) {
				return true, nil
			},
		},
		NonTerms: parser.NonTermHooks{
			parser.AnyNonTerm: tree.NonTermHook,
		},
	}
	root, e := p.Parse(queue, &hs)
	if e != nil {
		return nil, e
	}

	rootNode := root.(tree.NonTermNode)
	result := NewConf(rootNode)
	children := tree.Children(rootNode)
	for _, child := range children {
		if !child.IsNonTerm() {
			continue
		}

		s := result.AddSectionNode(child.(tree.NonTermNode))
		entries := tree.Children(child)
		for _, entry := range entries {
			if entry.TypeName() == entryNt {
				s.AddEntryNode(entry.(tree.NonTermNode))
			}
		}
	}

	return result, nil
}

func ParseFile (name string) (*Conf, error) {
	file, e := os.Open(name)
	if e != nil {
		if !os.IsNotExist(e) {
			return nil, e
		}

		content := make([]byte, 0)
		return Parse(name, &content)
	}

	defer file.Close()
	stat, e := file.Stat()
	if e != nil {
		return nil, e
	}

	fsize := stat.Size()
	if fsize > (1 << 20) {
		return nil, errors.New("only accept files no longer than 1 MB")
	}

	content := make([]byte, fsize + 1)
	bytes, e := file.Read(content)
	if bytes != int(fsize) {
		return nil, errors.New("error reading file")
	}
	content = content[: fsize]
	return Parse(name, &content)
}

func Serialize (root tree.Node, w io.Writer) (written int, err error) {
	visitor := func (n tree.Node) tree.WalkerFlags {
		if !n.IsNonTerm() {
			i, e := w.Write([]byte(n.Token().Text()))
			if e == nil {
				written += i
			} else {
				err = e
				return tree.WalkerStop
			}
		}

		return 0
	}

	tree.Walk(root, tree.WalkLtr, visitor)
	return
}

func SaveFile (name string, root tree.Node) (int, error) {
	f, e := os.OpenFile(name, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0644)
	if e != nil {
		return 0, e
	}

	defer f.Close()
	return Serialize(root, f)
}

func IsValidName (name string) bool {
	return secNameRe.MatchString(name)
}
