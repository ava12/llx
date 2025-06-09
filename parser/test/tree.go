package test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/source"
)

type TreeNode struct {
	isNode     bool
	name, text string
	children   []*TreeNode
}

func NodeNode(name string) *TreeNode {
	return &TreeNode{true, name, "", make([]*TreeNode, 0)}
}

func TokenNode(name, content string) *TreeNode {
	return &TreeNode{false, name, content, nil}
}

func (n *TreeNode) NewNode(node string, token *lexer.Token) error {
	return nil
}

func (n *TreeNode) HandleNode(node string, result any) error {
	n.children = append(n.children, result.(*TreeNode))
	return nil
}

func (n *TreeNode) HandleToken(token *lexer.Token) error {
	n.children = append(n.children, TokenNode(token.TypeName(), token.Text()))
	return nil
}

func (n *TreeNode) EndNode() (result any, e error) {
	return n, nil
}

func nodeHook(ctx context.Context, node string, t *lexer.Token, _ *parser.NodeContext) (parser.NodeHookInstance, error) {
	return NodeNode(node), nil
}

var testNodeHooks = parser.NodeHooks{parser.AnyNode: nodeHook}

type stackNode struct {
	parent        *stackNode
	node          *TreeNode
	length, index int
}

type TreeValidator struct {
	sn   *stackNode
	cmds []string
}

var exprRe = regexp.MustCompile("\\(|\\)|'.+?'|[^\\s()]+")

func NewTreeValidator(n *TreeNode, expr string) *TreeValidator {
	return &TreeValidator{&stackNode{nil, n, len(n.children), 0}, exprRe.FindAllString(expr, -1)}
}

func (tv *TreeValidator) newError(message string, params ...any) error {
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

func (tv *TreeValidator) exprError(msg string) error {
	return tv.newError("error in validator expression: " + msg)
}

func (tv *TreeValidator) matchName(name string) error {
	if name[0] == '\'' {
		name = name[1 : len(name)-1]
	}
	node := tv.sn.node
	if tv.sn.index < 0 {
		if node.name != name {
			return tv.newError("expecting %s node, got %s", name, node.name)
		}

	} else {
		if tv.sn.index >= tv.sn.length {
			return tv.newError("expecting %s token, got end of node", name)
		}

		child := node.children[tv.sn.index]
		if child.isNode {
			return tv.newError("expecting %s token, got %s node", name, child.name)
		}

		if child.name != name && child.text != name {
			return tv.newError("expecting %s token, got %s(%s)", name, child.name, child.text)
		}
	}

	tv.sn.index++
	return nil
}

func (tv *TreeValidator) matchNtStart() error {
	if tv.sn.index >= tv.sn.length {
		return tv.newError("expecting child node, got end of node")
	}

	child := tv.sn.node.children[tv.sn.index]
	if !child.isNode {
		return tv.newError("expecting child node, got %s token", child.name)
	}

	tv.sn = &stackNode{tv.sn, child, len(child.children), -1}
	return nil
}

func (tv *TreeValidator) matchNtEnd() error {
	if tv.sn.parent == nil {
		return tv.exprError("excessive )")
	}

	if tv.sn.index != tv.sn.length {
		return tv.newError("expecting end of node, got %s", tv.sn.node.children[tv.sn.index].name)
	}

	tv.sn = tv.sn.parent
	tv.sn.index++
	return nil
}

func (tv *TreeValidator) Validate() error {
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

func ParseAsTestNode(ctx context.Context, p *parser.Parser, src string, ths, lhs parser.TokenHooks, opts ...parser.ParseOption) (*TreeNode, error) {
	hs := parser.Hooks{ths, lhs, testNodeHooks}
	q := source.NewQueue().Append(source.New("sample", []byte(src)))
	r, e := p.Parse(ctx, q, hs, opts...)
	if e == nil {
		return r.(*TreeNode), nil
	} else {
		return nil, e
	}
}

type exprErrSample struct {
	expr, err string
}

func TestParseTreeExpr(t *testing.T) {
	grammarSrc := "!aside $space; $space=/\\s+/; $any=/\\S+/; g={foo|bar|baz};foo='foo';bar='bar';baz='baz';"
	src := "baz foo"
	samples := []exprErrSample{
		{"(baz baz)(foo foo)", ""},
		{"(baz baz))(foo foo)", "excessive )"},
		{"(baz baz)(foo foo", "missing )"},
	}

	g, e := langdef.ParseString("", grammarSrc)
	var n *TreeNode
	if e == nil {
		p, _ := parser.New(g)
		n, e = ParseAsTestNode(context.Background(), p, src, nil, nil)
	}
	if e != nil {
		t.Fatalf("unexpected error: %s", e.Error())
	}

	for i, sample := range samples {
		e := NewTreeValidator(n, sample.expr).Validate()
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
