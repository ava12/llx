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

func (c *variantChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) error {
	bypass := false
	for _, chunk := range c.chunks {
		if !bypass && chunk.IsOptional() {
			bypass = true
			bypassRule(g, stateIndex, nextIndex)
		}
		e := chunk.BuildStates(g, stateIndex, nextIndex)
		if e != nil {
			return e
		}
	}
	return nil
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

func (c *groupChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) error {
	needBypass := (c.isRepeated || c.IsOptional())

	tailIndex := nextIndex
	if c.isRepeated {
		tailIndex = stateIndex
	}

	totalChunks := len(c.chunks)
	nextStates := make([]int, totalChunks)
	for i := 0; i < (totalChunks - 1); i++ {
		nextStates[i], _ = addState(g)
	}
	nextStates[len(nextStates) - 1] = tailIndex

	currentIndex := stateIndex
	for i, chunk := range c.chunks {
		if needBypass && !chunk.IsOptional() {
			bypassRule(g, currentIndex, nextIndex)
			needBypass = false
		}
		e := chunk.BuildStates(g, currentIndex, nextStates[i])
		if e != nil {
			return e
		}

		currentIndex = nextStates[i]
	}
	if needBypass && totalChunks > 1 {
		if c.isRepeated {
			return emptyRepeatableError("")
		} else {
			bypassRule(g, nextStates[totalChunks - 2], nextIndex)
		}
	}
	return nil
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

func (c tokenChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) error {
	addRule(&g.States[stateIndex], []int{int(c)}, nextIndex, grammar.SameNonTerm)
	return nil
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

func (c *nonTermChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) error {
	firstTokens := c.FirstTokens().ToSlice()
	addRule(&g.States[stateIndex], firstTokens, nextIndex, c.item.Index)
	return nil
}

func addState (g *parseResult) (stateIndex int, state *stateEntry) {
	stateIndex = len(g.States)
	g.States = append(g.States, stateEntry{
		noGroup,
		map[int]grammar.Rule{},
		map[int][]grammar.Rule{},
	})
	state = &g.States[stateIndex]
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

func bypassRule (g *parseResult, stateIndex, nextIndex int) {
	if stateIndex >= 0 {
		g.States[stateIndex].Rules[grammar.AnyToken] = grammar.Rule{0, nextIndex, grammar.SameNonTerm}
	}
}
