package langdef

import (
	"github.com/ava12/llx"
	"github.com/ava12/llx/grammar"
)

type variantChunk struct {
	chunks []chunk
}

func newVariantChunk () *variantChunk {
	return &variantChunk{make([]chunk, 0)}
}

func (c *variantChunk) FirstTokens () llx.IntSet {
	result := llx.NewIntSet()
	for _, ch := range c.chunks {
		result.Union(ch.FirstTokens())
	}
	return result
}

func (c *variantChunk) IsOptional () bool {
	for _, ch := range c.chunks {
		if ch.IsOptional() {
			return true
		}
	}

	return false
}

func (c *variantChunk) Append (ch chunk) {
	c.chunks = append(c.chunks, ch)
}

func (c *variantChunk) BuildStates (g *grammar.Grammar, stateIndex, nextIndex int) {
	bypass := false
	for _, chunk := range c.chunks {
		if !bypass && chunk.IsOptional() {
			bypass = true
			bypassRule(g, stateIndex, nextIndex)
		}
		chunk.BuildStates(g, stateIndex, nextIndex)
	}
}


type groupChunk struct {
	chunks     []chunk
	isOptional bool
	isRepeated bool
}

func newGroupChunk (isOptional, isRepeated bool) *groupChunk {
	return &groupChunk {[]chunk{}, isOptional, isRepeated}
}

func (c *groupChunk) FirstTokens () llx.IntSet {
	result := llx.NewIntSet()
	for _, ch := range c.chunks {
		result.Union(ch.FirstTokens())
		if !ch.IsOptional() {
			break
		}
	}

	return result
}

func (c *groupChunk) IsOptional () bool {
	if c.isOptional || len(c.chunks) == 0 {
		return true
	}
	for _, ch := range c.chunks {
		if !ch.IsOptional() {
			return false
		}
	}
	return true
}

func (c *groupChunk) Append (ch chunk) {
	c.chunks = append(c.chunks, ch)
}

func (c *groupChunk) BuildStates (g *grammar.Grammar, stateIndex, nextIndex int) {
	if c.isRepeated || c.IsOptional() {
		bypassRule(g, stateIndex, nextIndex)
	}

	tailIndex := nextIndex
	if c.isRepeated {
		tailIndex = stateIndex
	}

	nextStates := make([]int, len(c.chunks))
	for i := 0; i < (len(c.chunks) - 1); i++ {
		nextStates[i], _ = addState(g)
	}
	nextStates[len(nextStates) - 1] = tailIndex

	currentIndex := stateIndex
	for i, chunk := range c.chunks {
		chunk.BuildStates(g, currentIndex, nextStates[i])
		currentIndex = nextStates[i]
	}
}

type tokenChunk int

func newTokenChunk (t int) tokenChunk {
	return tokenChunk(t)
}

func (c tokenChunk) FirstTokens () llx.IntSet {
	return llx.NewIntSet(int(c))
}

func (c tokenChunk) IsOptional () bool {
	return false
}

func (c tokenChunk) BuildStates (g *grammar.Grammar, stateIndex, nextIndex int) {
	addRule(&g.States[stateIndex], []int{int(c)}, nextIndex, grammar.SameNonTerm)
}


type nonTermChunk struct {
	name string
	item *nonTermItem
}

func newNonTermChunk (name string, item *nonTermItem) *nonTermChunk {
	return &nonTermChunk{name, item}
}

func (c *nonTermChunk) FirstTokens () llx.IntSet {
	return c.item.FirstTokens
}

func (c *nonTermChunk) IsOptional () bool {
	return false
}

func (c *nonTermChunk) BuildStates (g *grammar.Grammar, stateIndex, nextIndex int) {
	firstTokens := c.FirstTokens().ToSlice()
	addRule(&g.States[stateIndex], firstTokens, nextIndex, c.item.Index)
}

func addState (g *grammar.Grammar) (stateIndex int, state *grammar.State) {
	stateIndex = len(g.States)
	g.States = append(g.States, grammar.State {
		noGroup,
		map[int]grammar.Rule{},
		map[int][]grammar.Rule{},
	})
	state = &g.States[stateIndex]
	return
}

func addRule (st *grammar.State, tokens []int, state, nt int) {
	for _, token := range tokens {
		rule, hasSingle := st.Rules[token]
		_, hasAmbiguous := st.MultiRules[token]
		if !hasSingle && !hasAmbiguous {
			st.Rules[token] = grammar.Rule{state, nt}
		} else if !hasAmbiguous {
			delete(st.Rules, token)
			st.MultiRules[token] = []grammar.Rule{
				rule,
				{state, nt},
			}
		} else {
			st.MultiRules[token] = append(st.MultiRules[token], grammar.Rule{state, nt})
		}
	}
}

func bypassRule (g *grammar.Grammar, stateIndex, nextIndex int) {
	if stateIndex >= 0 {
		g.States[stateIndex].Rules[grammar.AnyToken] = grammar.Rule{nextIndex, grammar.SameNonTerm}
	}
}
