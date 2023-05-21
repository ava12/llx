package parser

import (
	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/lexer"
)

type branch struct {
	next    *branch
	index   int
	pc      *ParseContext
	applied []grammar.Rule
	ntTree  *nodeRec
	node    *nodeRec
	inited  bool
}

func createBranches (pc *ParseContext, nt *nodeRec, ars []grammar.Rule) *branch {
	ntCopy := &nodeRec{nil, nil, nil, nt.group, nt.index, nt.state}
	result := &branch{nil, 1, pc, []grammar.Rule{ars[0]}, nil, ntCopy, false}
	result.split(ars)
	return result
}

func (b *branch) split (ars []grammar.Rule) {
	prev := b
	nt := b.node
	b.index *= 100
	ruleCnt := len(b.applied)
	for i := 1; i < len(ars); i++ {
		nars := make([]grammar.Rule, ruleCnt)
		copy(nars, b.applied)
		if b.inited {
			nars = append(nars, ars[i])
		} else {
			nars[ruleCnt - 1] = ars[i]
		}
		ntCopy := &nodeRec{nt.prev, nil, nil, nt.group, nt.index, nt.state}
		current := &branch{prev.next, b.index + i, b.pc, nars, b.ntTree, ntCopy, false}
		prev.next = current
		prev = current
	}
}

func (b *branch) applyToken (tok *lexer.Token) (success bool) {
	if b.node == nil {
		return false
	}

	if b.pc.isAsideToken(tok) {
		b.applied = append(b.applied, grammar.Rule{tok.Type(), repeatState, grammar.SameNode})
		return true
	}

	var ars []grammar.Rule
	gr := b.pc.parser.grammar
	for b.node != nil {
		if b.inited {
			ars = b.pc.findRules(tok, gr.States[b.node.state])
		} else {
			ars = b.applied[len(b.applied) - 1 :]
		}
		cnt := len(ars)

		if cnt == 0 {
			return false
		}

		if cnt > 1 {
			b.split(ars)
		}

		ar := ars[0]
		if b.inited {
			b.applied = append(b.applied, ar)
		} else {
			b.inited = true
		}

		isWildcard := (ar.Token == grammar.AnyToken)
		isFinal := (ar.State == grammar.FinalState)
		isSame := (ar.Node == grammar.SameNode)

		b.node.state = ar.State
		if ar.State != grammar.FinalState {
			b.node.group = gr.States[ar.State].Group
		}

		if isFinal && isSame {
			for b.node != nil && b.node.state == grammar.FinalState {
				ntr := b.ntTree
				if ntr == nil {
					b.node = nil
				} else {
					b.node = &nodeRec{ntr.prev, nil, nil, ntr.group, ntr.index, ntr.state}
					b.ntTree = ntr.prev
				}
			}
		}

		if isSame && !isWildcard {
			return true
		}

		if isWildcard && b.node == nil {
			return false
		}

		if !isSame {
			gr := b.pc.parser.grammar
			nt := gr.Nodes[ar.Node]
			b.ntTree = b.node
			b.node = &nodeRec{b.node, nil, nil, gr.States[nt.FirstState].Group, ar.Node, nt.FirstState}
		}
	}

	return true
}

func (b *branch) nextGroup () int {
	if b.node != nil {
		return b.node.group
	}

	if b.next != nil {
		return b.next.nextGroup()
	} else {
		return 0
	}
}
