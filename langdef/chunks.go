package langdef

import (
	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/util/intset"
)

type variantChunk struct {
	chunks []chunk
}

func newVariantChunk () *variantChunk {
	return &variantChunk{make([]chunk, 0)}
}

func (c *variantChunk) FirstTerms () intset.T {
	result := intset.New()
	for _, ch := range c.chunks {
		result.Union(ch.FirstTerms())
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

func (c *groupChunk) FirstTerms () intset.T {
	result := intset.New()
	for _, ch := range c.chunks {
		result.Union(ch.FirstTerms())
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

type termChunk int

func newTermChunk (t int) termChunk {
	return termChunk(t)
}

func (c termChunk) FirstTerms () intset.T {
	return intset.New(int(c))
}

func (c termChunk) IsOptional () bool {
	return false
}

func (c termChunk) BuildStates (g *grammar.Grammar, stateIndex, nextIndex int) {
	addRule(&g.States[stateIndex], []int{int(c)}, nextIndex, grammar.SameNonTerm)
}


type nonTermChunk struct {
	name string
	item *nonTermItem
}

func newNonTermChunk (name string, item *nonTermItem) *nonTermChunk {
	return &nonTermChunk{name, item}
}

func (c *nonTermChunk) FirstTerms () intset.T {
	return c.item.FirstTerms
}

func (c *nonTermChunk) IsOptional () bool {
	return false
}

func (c *nonTermChunk) BuildStates (g *grammar.Grammar, stateIndex, nextIndex int) {
	firstTerms := c.FirstTerms().ToSlice()
	addRule(&g.States[stateIndex], firstTerms, nextIndex, c.item.Index)
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

func addRule (st *grammar.State, terms []int, state, nt int) {
	for _, term := range terms {
		rule, hasSingle := st.Rules[term]
		_, hasAmbiguous := st.MultiRules[term]
		if !hasSingle && !hasAmbiguous {
			st.Rules[term] = grammar.Rule{state, nt}
		} else if !hasAmbiguous {
			delete(st.Rules, term)
			st.MultiRules[term] = []grammar.Rule{
				rule,
				{state, nt},
			}
		} else {
			st.MultiRules[term] = append(st.MultiRules[term], grammar.Rule{state, nt})
		}
	}
}

func bypassRule (g *grammar.Grammar, stateIndex, nextIndex int) {
	if stateIndex >= 0 {
		g.States[stateIndex].Rules[grammar.AnyTerm] = grammar.Rule{nextIndex, grammar.SameNonTerm}
	}
}
