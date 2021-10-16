package langdef

import (
	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/internal/ints"
)

type variantChunk struct {
	chunks []chunk
}

func newVariantChunk () *variantChunk {
	return &variantChunk{make([]chunk, 0)}
}

func (c *variantChunk) FirstTokens () *ints.Set {
	result := ints.NewSet()
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

func (c *variantChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) {
	for _, chunk := range c.chunks {
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

func (c *groupChunk) FirstTokens () *ints.Set {
	result := ints.NewSet()
	for _, ch := range c.chunks {
		result.Union(ch.FirstTokens())
		if !ch.IsOptional() {
			break
		}
	}

	return result
}

func (c *groupChunk) IsOptional () bool {
	if c.isOptional {
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

func (c *groupChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) {
	tailIndex := nextIndex
	totalChunks := len(c.chunks)
	needExtraRule := (c.isRepeated && totalChunks > 1)
	states := make([]int, totalChunks + 1)
	if c.isRepeated {
		tailIndex = stateIndex
	}
	if needExtraRule {
		states[0] = addState(g)
	} else {
		states[0] = stateIndex
	}
	for i := 1; i < totalChunks; i++ {
		states[i] = addState(g)
	}
	states[totalChunks] = tailIndex

	for i, chunk := range c.chunks {
		chunk.BuildStates(g, states[i], states[i + 1])
	}

	if c.chunks[totalChunks - 1].IsOptional() {
		bypassRule(g, states[totalChunks - 1], tailIndex)
	}
	for i := totalChunks - 2; i >= 0; i-- {
		if c.chunks[i].IsOptional() {
			copyRules(g, states[i], states[i + 1])
		}
	}
	if needExtraRule {
		copyRules(g, stateIndex, states[0])
	}
}

type tokenChunk int

func newTokenChunk (t int) tokenChunk {
	return tokenChunk(t)
}

func (c tokenChunk) FirstTokens () *ints.Set {
	return ints.NewSet(int(c))
}

func (c tokenChunk) IsOptional () bool {
	return false
}

func (c tokenChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) {
	addRule(&g.States[stateIndex], []int{int(c)}, nextIndex, grammar.SameNonTerm)
}


type nonTermChunk struct {
	name string
	item *nonTermItem
}

func newNonTermChunk (name string, item *nonTermItem) *nonTermChunk {
	return &nonTermChunk{name, item}
}

func (c *nonTermChunk) FirstTokens () *ints.Set {
	return c.item.FirstTokens
}

func (c *nonTermChunk) IsOptional () bool {
	return false
}

func (c *nonTermChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) {
	firstTokens := c.FirstTokens().ToSlice()
	addRule(&g.States[stateIndex], firstTokens, nextIndex, c.item.Index)
}

func addState (g *parseResult) (stateIndex int) {
	stateIndex = len(g.States)
	g.States = append(g.States, stateEntry{
		noGroup,
		map[int]grammar.Rule{},
		map[int][]grammar.Rule{},
	})
	return
}

func addRule (st *stateEntry, tokens []int, state, nt int) {
	for _, token := range tokens {
		rule, hasSingle := st.Rules[token]
		_, hasAmbiguous := st.MultiRules[token]
		if !hasSingle && !hasAmbiguous {
			st.Rules[token] = grammar.Rule{0, state, nt}
		} else if !hasAmbiguous {
			delete(st.Rules, token)
			st.MultiRules[token] = []grammar.Rule{
				rule,
				{0, state, nt},
			}
		} else {
			st.MultiRules[token] = append(st.MultiRules[token], grammar.Rule{0, state, nt})
		}
	}
}

func copyRules (g *parseResult, toState, fromState int) {
	to := &g.States[toState]
	from := g.States[fromState]
	for t, r := range from.Rules {
		addRule(to, []int{t}, r.State, r.NonTerm)
	}
	for t, rs := range from.MultiRules {
		for _, r := range rs {
			addRule(to, []int{t}, r.State, r.NonTerm)
		}
	}
}

func bypassRule (g *parseResult, stateIndex, bypassIndex int) {
	if stateIndex >= 0 {
		g.States[stateIndex].Rules[grammar.AnyToken] = grammar.Rule{0, bypassIndex, grammar.SameNonTerm}
	}
}
