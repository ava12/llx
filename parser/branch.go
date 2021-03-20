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
	ntTree, nonterm *nontermRec
	inited bool
}

func createBranches (pc *ParseContext, nt *nontermRec, ars []appliedRule) *branch {
	ntCopy := &nontermRec{nil, nt.lexer, nt.states, nil, nt.index, nt.state}
	result := &branch{nil, 1, pc, []appliedRule{ars[0]}, nil, ntCopy, false}
	result.split(ars)
	return result
}

func (b *branch) split (ars []appliedRule) {
	prev := b
	nt := b.nonterm
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
		ntCopy := &nontermRec{nt.prev, nt.lexer, nt.states, nil, nt.index, nt.state}
		current := &branch{prev.next, b.index + i, b.pc, nars, b.ntTree, ntCopy, false}
		prev.next = current
		prev = current
	}
}

func (b *branch) applyToken (tok *lexer.Token) (success bool) {
	if b.nonterm == nil {
		return false
	}

	if b.pc.isAsideToken(tok) {
		b.applied = append(b.applied, appliedRule{tok.Type(), repeatState, grammar.SameNonterm})
		return true
	}

	var ars []appliedRule
	for b.nonterm != nil {
		if b.inited {
			ars = b.pc.findRules(tok, b.nonterm.states[b.nonterm.state])
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
		isSame := (ar.nonterm == grammar.SameNonterm)

		b.nonterm.state = ar.state

		if isFinal && isSame {
			for b.nonterm != nil && b.nonterm.state == grammar.FinalState {
				ntr := b.ntTree
				if ntr == nil {
					b.nonterm = nil
				} else {
					b.nonterm = &nontermRec{ntr.prev, ntr.lexer, ntr.states, nil, ntr.index, ntr.state}
					b.ntTree = ntr.prev
				}
			}
		}

		if isSame && !isWildcard {
			return true
		}

		if !isSame {
			nt := b.pc.parser.grammar.Nonterms[ar.nonterm]
			b.ntTree = b.nonterm
			b.nonterm = &nontermRec{b.nonterm, b.pc.lexers[nt.States[0].Group], nt.States, nil, ar.nonterm, grammar.InitialState}
		}
	}

	return true
}
