package tree

import (
	"testing"
)

func TestNodeElement_AddChild(t *testing.T) {
	ntn := &nodeElement{}
	tn1 := &tokenElement{}
	tn2 := &tokenElement{}
	tn3 := &tokenElement{}

	ntn.AddChild(tn1, nil)
	assert(t, tn1.parent == ntn)
	assert(t, ntn.firstChild == tn1)
	assert(t, ntn.lastChild == tn1)

	ntn.AddChild(tn2, tn3)
	assert(t, tn2.parent == nil)
	assert(t, ntn.firstChild == tn1)
	assert(t, ntn.lastChild == tn1)

	ntn.AddChild(nil, tn1)
	assert(t, ntn.firstChild == tn1)
	assert(t, ntn.lastChild == tn1)

	ntn.AddChild(tn2, nil)
	assert(t, tn2.parent == ntn)
	assert(t, ntn.firstChild == tn1)
	assert(t, ntn.lastChild == tn2)
	assert(t, tn1.next == tn2)
	assert(t, tn2.prev == tn1)

	ntn.AddChild(tn3, tn2)
	assert(t, tn3.parent == ntn)
	assert(t, ntn.firstChild == tn1)
	assert(t, ntn.lastChild == tn2)
	assert(t, tn1.next == tn3)
	assert(t, tn3.next == tn2)
	assert(t, tn3.prev == tn1)
	assert(t, tn2.prev == tn3)
}

func TestNodeElement_RemoveChild(t *testing.T) {
	ntn := &nodeElement{}
	tn1 := &tokenElement{}
	tn2 := &tokenElement{}
	tn3 := &tokenElement{}

	ntn.AddChild(tn1, nil)
	ntn.RemoveChild(nil)
	assert(t, ntn.firstChild == tn1)
	assert(t, ntn.lastChild == tn1)

	ntn.RemoveChild(tn2)
	assert(t, ntn.firstChild == tn1)
	assert(t, ntn.lastChild == tn1)

	ntn.AddChild(tn2, nil)
	ntn.AddChild(tn3, nil)

	ntn.RemoveChild(tn2)
	assert(t, ntn.firstChild == tn1)
	assert(t, ntn.lastChild == tn3)
	assert(t, tn1.next == tn3)
	assert(t, tn3.prev == tn1)

	ntn.RemoveChild(tn1)
	assert(t, ntn.firstChild == tn3)
	assert(t, ntn.lastChild == tn3)
	assert(t, tn3.next == nil)
	assert(t, tn3.prev == nil)

	ntn.RemoveChild(tn3)
	assert(t, ntn.firstChild == nil)
	assert(t, ntn.lastChild == nil)
}
