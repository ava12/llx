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
	ntTree  *nonTermRec
	nonTerm *nonTermRec
	inited  bool
}

func createBranches (pc *ParseContext, nt *nonTermRec, ars []grammar.Rule) *branch {
	ntCopy := &nonTermRec{nil, nil, nil, nt.group, nt.index, nt.state}
	result := &branch{nil, 1, pc, []grammar.Rule{ars[0]}, nil, ntCopy, false}
	result.split(ars)
	return result
}

func (b *branch) split (ars []grammar.Rule) {
	prev := b
	nt := b.nonTerm
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
		ntCopy := &nonTermRec{nt.prev, nil, nil, nt.group, nt.index, nt.state}
		current := &branch{prev.next, b.index + i, b.pc, nars, b.ntTree, ntCopy, false}
		prev.next = current
		prev = current
	}
}

func (b *branch) applyToken (tok *lexer.Token) (success bool) {
	if b.nonTerm == nil {
		return false
	}

	if b.pc.isAsideToken(tok) {
		b.applied = append(b.applied, grammar.Rule{tok.Type(), repeatState, grammar.SameNonTerm})
		return true
	}

	var ars []grammar.Rule
	gr := b.pc.parser.grammar
	for b.nonTerm != nil {
		if b.inited {
			ars = b.pc.findRules(tok, gr.States[b.nonTerm.state])
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
		isSame := (ar.NonTerm == grammar.SameNonTerm)

		b.nonTerm.state = ar.State
		if ar.State != grammar.FinalState {
			b.nonTerm.group = gr.States[ar.State].Group
		}

		if isFinal && isSame {
			for b.nonTerm != nil && b.nonTerm.state == grammar.FinalState {
				ntr := b.ntTree
				if ntr == nil {
					b.nonTerm = nil
				} else {
					b.nonTerm = &nonTermRec{ntr.prev, nil, nil, ntr.group, ntr.index, ntr.state}
					b.ntTree = ntr.prev
				}
			}
		}

		if isSame && !isWildcard {
			return true
		}

		if isWildcard && b.nonTerm == nil {
			return false
		}

		if !isSame {
			gr := b.pc.parser.grammar
			nt := gr.NonTerms[ar.NonTerm]
			b.ntTree = b.nonTerm
			b.nonTerm = &nonTermRec{b.nonTerm, nil, nil, gr.States[nt.FirstState].Group, ar.NonTerm, nt.FirstState}
		}
	}

	return true
}

func (b *branch) nextGroup () int {
	if b.nonTerm != nil {
		return b.nonTerm.group
	}

	if b.next != nil {
		return b.next.nextGroup()
	} else {
		return 0
	}
}
