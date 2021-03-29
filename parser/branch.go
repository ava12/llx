package parser

import (
	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/lexer"
)

type branch struct {
	next *branch
	index int
	pc *ParseContext
	applied []appliedRule
	ntTree, nonTerm *nonTermRec
	inited bool
}

func createBranches (pc *ParseContext, nt *nonTermRec, ars []appliedRule) *branch {
	ntCopy := &nonTermRec{nil, nt.group, nt.states, nil, nt.index, nt.state}
	result := &branch{nil, 1, pc, []appliedRule{ars[0]}, nil, ntCopy, false}
	result.split(ars)
	return result
}

func (b *branch) split (ars []appliedRule) {
	prev := b
	nt := b.nonTerm
	b.index *= 100
	ruleCnt := len(b.applied)
	for i := 1; i < len(ars); i++ {
		nars := make([]appliedRule, ruleCnt)
		copy(nars, b.applied)
		if b.inited {
			nars = append(nars, ars[i])
		} else {
			nars[ruleCnt - 1] = ars[i]
		}
		ntCopy := &nonTermRec{nt.prev, nt.group, nt.states, nil, nt.index, nt.state}
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
		b.applied = append(b.applied, appliedRule{tok.Type(), repeatState, grammar.SameNonTerm})
		return true
	}

	var ars []appliedRule
	for b.nonTerm != nil {
		if b.inited {
			ars = b.pc.findRules(tok, b.nonTerm.states[b.nonTerm.state])
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

		isWildcard := (ar.term == grammar.AnyTerm)
		isFinal := (ar.state == grammar.FinalState)
		isSame := (ar.nonTerm == grammar.SameNonTerm)

		b.nonTerm.state = ar.state
		if ar.state != grammar.FinalState {
			b.nonTerm.group = b.nonTerm.states[ar.state].Group
		}

		if isFinal && isSame {
			for b.nonTerm != nil && b.nonTerm.state == grammar.FinalState {
				ntr := b.ntTree
				if ntr == nil {
					b.nonTerm = nil
				} else {
					b.nonTerm = &nonTermRec{ntr.prev, ntr.group, ntr.states, nil, ntr.index, ntr.state}
					b.ntTree = ntr.prev
				}
			}
		}

		if isSame && !isWildcard {
			return true
		}

		if !isSame {
			nt := b.pc.parser.grammar.NonTerms[ar.nonTerm]
			b.ntTree = b.nonTerm
			b.nonTerm = &nonTermRec{b.nonTerm, nt.States[0].Group, nt.States, nil, ar.nonTerm, grammar.InitialState}
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
