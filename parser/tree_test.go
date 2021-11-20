package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

type treeNode struct {
	isNonTerm  bool
	name, text string
	children   []*treeNode
}

func nonTermNode (name string) *treeNode {
	return &treeNode{true, name, "", make([]*treeNode, 0)}
}

func tokenNode (name, content string) *treeNode {
	return &treeNode{false, name, content, nil}
}

func (n *treeNode) NewNonTerm (nonTerm string, token *lexer.Token) error {
	return nil
}

func (n *treeNode) HandleNonTerm (nonTerm string, result interface{}) error {
	n.children = append(n.children, result.(*treeNode))
	return nil
}

func (n *treeNode) HandleToken (token *lexer.Token) error {
	n.children = append(n.children, tokenNode(token.TypeName(), token.Text()))
	return nil
}

func (n *treeNode) EndNonTerm () (result interface{}, e error) {
	return n, nil
}

func nodeHook (nonTerm string, t *lexer.Token, pc *ParseContext) (NonTermHookInstance, error) {
	return nonTermNode(nonTerm), nil
}

var testNodeHooks = NonTermHooks{AnyNonTerm: nodeHook}


type stackNode struct {
	parent        *stackNode
	node          *treeNode
	length, index int
}

type treeValidator struct {
	sn   *stackNode
	cmds []string
}

var exprRe = regexp.MustCompile("\\(|\\)|'.+?'|[^\\s()]+")

func newTreeValidator (n *treeNode, expr string) *treeValidator {
	return &treeValidator{&stackNode{nil, n, len(n.children), 0}, exprRe.FindAllString(expr, -1)}
}

func (tv *treeValidator) newError (message string, params ...interface{}) error {
	if len(params) > 0 {
		message = fmt.Sprintf(message, params...)
	}
	path := make([]int, 0)
	csn := tv.sn
	for csn != nil {
		path = append([]int{csn.index}, path...)
		csn = csn.parent
	}
	pathString := fmt.Sprintf("path %v, %s: ", path, tv.sn.node.name)
	return errors.New(pathString + message)
}

func (tv *treeValidator) exprError (msg string) error {
	return tv.newError("error in validator expression: " + msg)
}

func (tv *treeValidator) matchName (name string) error {
	if name[0] == '\'' {
		name = name[1 : len(name) - 1]
	}
	node := tv.sn.node
	if tv.sn.index < 0 {
		if node.name != name {
			return tv.newError("expecting %s non-terminal, got %s", name, node.name)
		}

	} else {
		if tv.sn.index >= tv.sn.length {
			return tv.newError("expecting %s token, got end of non-terminal", name)
		}

		child := node.children[tv.sn.index]
		if child.isNonTerm {
			return tv.newError("expecting %s token, got %s non-terminal", name, child.name)
		}

		if child.name != name && child.text != name {
			return tv.newError("expecting %s token, got %s(%s)", name, child.name, child.text)
		}
	}

	tv.sn.index++
	return nil
}

func (tv *treeValidator) matchNtStart () error {
	if tv.sn.index >= tv.sn.length {
		return tv.newError("expecting child non-terminal, got end of non-terminal")
	}

	child := tv.sn.node.children[tv.sn.index]
	if !child.isNonTerm {
		return tv.newError("expecting child non-terminal, got %s token", child.name)
	}

	tv.sn = &stackNode{tv.sn, child, len(child.children), -1}
	return nil
}

func (tv *treeValidator) matchNtEnd () error {
	if tv.sn.parent == nil {
		return tv.exprError("excessive )")
	}

	if tv.sn.index != tv.sn.length {
		return tv.newError("expecting end of non-terminal, got %s", tv.sn.node.children[tv.sn.index].name)
	}

	tv.sn = tv.sn.parent
	tv.sn.index++
	return nil
}

func (tv *treeValidator) validate () error {
	var e error
	for _, cmd := range tv.cmds {
		switch cmd {
		case "(":
			e = tv.matchNtStart()
		case ")":
			e = tv.matchNtEnd()
		default:
			e = tv.matchName(cmd)
		}

		if e != nil {
			return e
		}
	}

	if tv.sn.parent == nil {
		return nil
	} else {
		return tv.exprError("missing )")
	}
}

func parseAsTestNode (g *grammar.Grammar, src string, ths, lhs TokenHooks) (*treeNode, error) {
	hs := &Hooks{ths, lhs, testNodeHooks}
	parser := New(g)
	q := source.NewQueue().Append(source.New("sample", []byte(src)))
	r, e := parser.Parse(q, hs)
	if e == nil {
		return r.(*treeNode), nil
	} else {
		return nil, e
	}
}


type exprErrSample struct {
	expr, err string
}

func TestParseTreeExpr (t *testing.T) {
	grammarSrc := "!aside $space; $space=/\\s+/; $any=/\\S+/; g={foo|bar|baz};foo='foo';bar='bar';baz='baz';"
	src := "baz foo"
	samples := []exprErrSample{
		{"(baz baz)(foo foo)", ""},
		{"(baz baz))(foo foo)", "excessive )"},
		{"(baz baz)(foo foo", "missing )"},
	}

	g, e := langdef.ParseString("", grammarSrc)
	var n *treeNode
	if e == nil {
		n, e = parseAsTestNode(g, src, nil, nil)
	}
	if e != nil {
		t.Fatalf("unexpected error: %s", e.Error())
	}

	for i, sample := range samples {
		e := newTreeValidator(n, sample.expr).validate()
		if sample.err == "" {
			if e != nil {
				t.Errorf("sample #%d: unexpected error: %s", i, e.Error())
			}
		} else {
			if e == nil {
				t.Errorf("sample #%d: expected error, got success", i)
			} else if !strings.Contains(e.Error(), sample.err) {
				t.Errorf("sample #%d: unexpected error type: %s", i, e.Error())
			}
		}
	}
}
