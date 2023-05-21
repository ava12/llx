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
		states[0], _ = g.AddState()
	} else {
		states[0] = stateIndex
	}
	for i := 1; i < totalChunks; i++ {
		states[i], _ = g.AddState()
	}
	states[totalChunks] = tailIndex

	for i, chunk := range c.chunks {
		chunk.BuildStates(g, states[i], states[i + 1])
	}

	if c.chunks[totalChunks - 1].IsOptional() {
		g.States[states[totalChunks - 1]].BypassRule(tailIndex)
	}
	for i := totalChunks - 2; i >= 0; i-- {
		if c.chunks[i].IsOptional() {
			g.States[states[i]].CopyRules(g.States[states[i + 1]])
		}
	}
	if needExtraRule {
		g.States[stateIndex].CopyRules(g.States[states[0]])
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
	g.States[stateIndex].AddRule(nextIndex, grammar.SameNode, int(c))
}


type nodeChunk struct {
	name string
	item *nodeItem
}

func newNodeChunk (name string, item *nodeItem) *nodeChunk {
	return &nodeChunk{name, item}
}

func (c *nodeChunk) FirstTokens () *ints.Set {
	return c.item.FirstTokens
}

func (c *nodeChunk) IsOptional () bool {
	return false
}

func (c *nodeChunk) BuildStates (g *parseResult, stateIndex, nextIndex int) {
	firstTokens := c.FirstTokens().ToSlice()
	g.States[stateIndex].AddRule(nextIndex, c.item.Index, firstTokens ...)
}
