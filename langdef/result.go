package langdef

import (
	"sort"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/internal/ints"
)

type stateEntry struct {
	Group int
	Rules map[int][]grammar.Rule
}

type parseResult struct {
	Tokens []grammar.Token
	Nodes  []grammar.Node
	States []*stateEntry
	NIndex nodeIndex
}

func newParseResult() *parseResult {
	return &parseResult{make([]grammar.Token, 0), make([]grammar.Node, 0), make([]*stateEntry, 0), make(nodeIndex)}
}

func (pr *parseResult) AddState() (stateIndex int, st *stateEntry) {
	stateIndex = len(pr.States)
	st = &stateEntry{noGroup, map[int][]grammar.Rule{}}
	pr.States = append(pr.States, st)
	return
}

func (pr *parseResult) BuildGrammar() *grammar.Grammar {
	pr.dropUnusedStates()
	g := &grammar.Grammar{Tokens: pr.Tokens, Nodes: pr.Nodes, States: make([]grammar.State, len(pr.States))}
	for si, se := range pr.States {
		se.BuildGrammarState(g, si)
	}
	return g
}

func (pr *parseResult) dropUnusedStates() {
	usedStates := ints.NewSet()

	for _, nt := range pr.Nodes {
		usedStates.Add(nt.FirstState)
	}
	for _, se := range pr.States {
		for _, rs := range se.Rules {
			for _, r := range rs {
				if r.State >= 0 {
					usedStates.Add(r.State)
				}
			}
		}
	}

	uss := usedStates.ToSlice()
	if len(uss) == len(pr.States) {
		return
	}

	newIndexes := make([]int, len(pr.States))
	currentIndex := 0
	for _, i := range uss {
		pr.States[currentIndex] = pr.States[i]
		newIndexes[i] = currentIndex
		currentIndex++
	}
	pr.States = pr.States[:currentIndex]

	for i, nt := range pr.Nodes {
		pr.Nodes[i].FirstState = newIndexes[nt.FirstState]
	}
	for _, se := range pr.States {
		for _, rs := range se.Rules {
			for i, r := range rs {
				if r.State >= 0 {
					rs[i].State = newIndexes[r.State]
				}
			}
		}
	}
}

func (se *stateEntry) AddRule(state, nt int, tokens ...int) {
	rule := grammar.Rule{0, state, nt}
	for _, k := range tokens {
		rule.Token = k
		if k >= 0 {
			se.Rules[k] = append(se.Rules[k], rule)
		} else {
			se.Rules[k] = []grammar.Rule{rule}
		}
	}
}

func (se *stateEntry) BypassRule(nextState int) {
	se.AddRule(nextState, grammar.SameNode, grammar.AnyToken)
}

func (se *stateEntry) CopyRules(from *stateEntry) {
	for k, rs := range from.Rules {
		if k >= 0 {
			se.Rules[k] = append(se.Rules[k], rs...)
		} else {
			se.Rules[k] = rs
		}
	}
}

func (se *stateEntry) BuildGrammarState(g *grammar.Grammar, si int) {
	rkeys, mkeys := se.rmKeys()
	erlen := len(rkeys)
	emlen := len(mkeys)
	se.writeGrammarState(g, si, erlen, emlen)

	for _, k := range rkeys {
		r := se.Rules[k][0]
		g.Rules = append(g.Rules, r)
	}

	rstart := len(g.Rules)
	for _, k := range mkeys {
		rs := se.Rules[k]
		mrlen := len(rs)
		g.Rules = append(g.Rules, rs...)
		g.MultiRules = append(g.MultiRules, grammar.MultiRule{k, rstart, rstart + mrlen})
		rstart += mrlen
	}
}

func (se *stateEntry) rmKeys() (rkeys, mkeys []int) {
	ermlen := len(se.Rules)
	rkeys = make([]int, 0, ermlen)
	mkeys = make([]int, 0, ermlen)
	for k, rs := range se.Rules {
		if len(rs) > 1 {
			mkeys = append(mkeys, k)
		} else {
			rkeys = append(rkeys, k)
		}
	}
	sort.Ints(rkeys)
	sort.Ints(mkeys)
	return
}

func (se *stateEntry) writeGrammarState(g *grammar.Grammar, si int, erlen, emlen int) {
	rstart := 0
	mstart := 0
	if erlen > 0 {
		rstart = len(g.Rules)
	}
	if emlen > 0 {
		mstart = len(g.MultiRules)
	}
	g.States[si] = grammar.State{se.Group, mstart, mstart + emlen, rstart, rstart + erlen}
}
