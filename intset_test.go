package llx

import (
	"sort"
	"testing"
)

func assert (t *testing.T, cond bool) {
	if !cond {
		t.FailNow()
	}
}

func checkItems (t *testing.T, s IntSet, first, last int, items []int) {
	index := make(map[int]bool, len(items))
	for _, i := range items {
		index[i] = true
	}
	for i := first; i <= last; i++ {
		assert(t, s.Contains(i) == index[i])
	}
}

func assertItemSuite (t *testing.T, s IntSet, items []int, index int) {
	sort.Ints(items)
	slice := s.ToSlice()
	flag := (len(slice) == len(items))
	if flag {
		for i, item := range slice {
			if item != items[i] {
				flag = false
				break
			}
		}
	}
	if !flag {
		t.Fatalf("suit #%d: expected %v, got %v", index, items, slice)
	}
}

func assertItems (t *testing.T, s IntSet, items []int) {
	assertItemSuite(t, s, items, 0)
}

func TestIntSize (t *testing.T) {
	var realShift uint
	if ^0 == 0xffffffff {
		realShift = 5
	} else {
		realShift = 6
	}
	assert(t, realShift == IntSizeShift)
}

func TestEmpty (t *testing.T) {
	s := NewIntSet()
	assert(t, s.IsEmpty())
	s.Add(1)
	assert(t, !s.IsEmpty())
	s.Add(2)
	assert(t, !s.IsEmpty())
	s.Remove(1)
	assert(t, !s.IsEmpty())
	s.Remove(2)
	assert(t, s.IsEmpty())
}

func TestEqual (t *testing.T) {
	s := FromSlice([]int{-10, 0, 10})
	s2 := s.Copy()
	assert(t, s.IsEqual(s2))
	assert(t, s2.IsEqual(s))
	s.Remove(0)
	assert(t, !s.IsEqual(s2))
	assert(t, !s2.IsEqual(s))
	s.Add(0)
	assert(t, s.IsEqual(s2))
	assert(t, s2.IsEqual(s))
}

func TestFromSlice (t *testing.T) {
	items := []int{-65, 1, 66}
	s := FromSlice(items)
	checkItems(t, s, -129, 128, items)
}

func TestToSlice (t *testing.T) {
	suite := [][]int{
		{},
		{1},
		{123, -1000, -1, 0, 1},
	}
	for _, items := range suite {
		s := FromSlice(items)
		assertItems(t, s, items)
	}
}

func TestAddRemove (t *testing.T) {
	s := NewIntSet()
	s.Add(0)
	s.Add(1)
	s.Add(-1)
	s.Add(100)
	assertItems(t, s, []int{-1, 0, 1, 100})
	s.Remove(0)
	s.Remove(-2)
	s.Remove(100)
	assertItems(t, s, []int{-1, 1})
}

func TestAddAll (t *testing.T) {
	s := NewIntSet(-100, -3, 100, 2)
	s.Add(-1, 0, 1, 2, 100, -200)
	assertItems(t, s, []int{-200, -100, -3, -1, 0, 1, 2, 100})
}

func TestRemoveAll (t *testing.T) {
	s := NewIntSet(-100, -10, -1, 0, 1, 10, 100)
	s.Remove(-100, -1, 0, 10)
	assertItems(t, s, []int{-10, 1, 100})
}

func TestCopy (t *testing.T) {
	items := []int{-100, -10, -1, 0, 1, 10, 100}
	s := FromSlice(items)
	s2 := s.Copy()
	assertItems(t, s2, items)
	s.Add(12)
	s.Remove(-10)
	s2.Add(123)
	s2.Remove(-100)
	assertItems(t, s, []int{-100, -1, 0, 1, 10, 12, 100})
	assertItems(t, s2, []int{-10, -1, 0, 1, 10, 100, 123})
}

var (
	logicBase = []int{-64, -33, -1, 0, 1, 33, 63, 64, 65}
	logicExtra = [][]int{
		{}, // -
		{-129}, // <l <l
		{-129, -65}, // <l =l
		{-129, -65, -33, -32, 1, 2}, // <l >l
		{-129, -65, -33, -32, 1, 2, 62, 63, 65, 66}, // <l =h
		{-129, -65, -33, -32, 1, 2, 62, 63, 65, 66, 128}, // <l >h
		{-33, -32, 1, 2}, // =l >l
		{-33, -32, 1, 2, 62, 63, 65, 66}, // =l =h
		{-33, -32, 1, 2, 62, 63, 65, 66, 128}, // =l >h
		{1, 2, 62, 63}, // >l >l
		{1, 2, 62, 63, 65, 66}, // >l =h
		{1, 2, 62, 63, 65, 66, 128}, // >l >h
		{65, 66, 128}, // =h >h
		{128}, // >h >h
	}
	logicUnion = [][]int{
		{-64, -33, -1, 0, 1, 33, 63, 64, 65}, // -
		{-129, -64, -33, -1, 0, 1, 33, 63, 64, 65}, // <l <l
		{-129, -65, -64, -33, -1, 0, 1, 33, 63, 64, 65}, // <l =l
		{-129, -65, -64, -33, -32, -1, 0, 1, 2, 33, 63, 64, 65}, // <l >l
		{-129, -65, -64, -33, -32, -1, 0, 1, 2, 33, 62, 63, 64, 65, 66}, // <l =h
		{-129, -65, -64, -33, -32, -1, 0, 1, 2, 33, 62, 63, 64, 65, 66, 128}, // <l >h
		{-64, -33, -32, -1, 0, 1, 2, 33, 63, 64, 65}, // =l >l
		{-64, -33, -32, -1, 0, 1, 2, 33, 62, 63, 64, 65, 66}, // =l =h
		{-64, -33, -32, -1, 0, 1, 2, 33, 62, 63, 64, 65, 66, 128}, // =l >h
		{-64, -33, -1, 0, 1, 2, 33, 62, 63, 64, 65}, // >l >l
		{-64, -33, -1, 0, 1, 2, 33, 62, 63, 64, 65, 66}, // >l =h
		{-64, -33, -1, 0, 1, 2, 33, 62, 63, 64, 65, 66, 128}, // >l >h
		{-64, -33, -1, 0, 1, 33, 63, 64, 65, 66, 128}, // =h >h
		{-64, -33, -1, 0, 1, 33, 63, 64, 65, 128}, // >h >h
	}
	logicIntersect = [][]int{
		{}, // -
		{}, // <l <l
		{}, // <l =l
		{-33, 1}, // <l >l
		{-33, 1, 63, 65}, // <l =h
		{-33, 1, 63, 65}, // <l >h
		{-33, 1}, // =l >l
		{-33, 1, 63, 65}, // =l =h
		{-33, 1,  63, 65}, // =l >h
		{1, 63}, // >l >l
		{1, 63, 65}, // >l =h
		{1, 63, 65}, // >l >h
		{65}, // =h >h
		{}, // >h >h
	}
	logicSubtract = [][]int{
		{-64, -33, -1, 0, 1, 33, 63, 64, 65}, // -
		{-64, -33, -1, 0, 1, 33, 63, 64, 65}, // <l <l
		{-64, -33, -1, 0, 1, 33, 63, 64, 65}, // <l =l
		{-64, -1, 0, 33, 63, 64, 65}, // <l >l
		{-64, -1, 0, 33, 64}, // <l =h
		{-64, -1, 0, 33, 64}, // <l >h
		{-64, -1, 0, 33, 63, 64, 65}, // =l >l
		{-64, -1, 0, 33, 64}, // =l =h
		{-64, -1, 0, 33, 64}, // =l >h
		{-64, -33, -1, 0, 33, 64, 65}, // >l >l
		{-64, -33, -1, 0, 33, 64}, // >l =h
		{-64, -33, -1, 0, 33, 64}, // >l >h
		{-64, -33, -1, 0, 1, 33, 63, 64}, // =h >h
		{-64, -33, -1, 0, 1, 33, 63, 64, 65}, // >h >h
	}
)

func checkLogic (t *testing.T, expected [][]int, f func (base, extra IntSet) IntSet) {
	base := FromSlice(logicBase)
	for index, items := range logicExtra {
		extra := FromSlice(items)
		result := f(base, extra)
		assertItemSuite(t, result, expected[index], index)
	}
}

func TestUnion (t *testing.T) {
	checkLogic(t, logicUnion, func (s, t IntSet) IntSet {
		return Union(s, t)
	})
}

func TestUnionReverse (t *testing.T) {
	checkLogic(t, logicUnion, func (s, t IntSet) IntSet {
		return Union(t, s)
	})
}

func TestIntersect (t *testing.T) {
	checkLogic(t, logicIntersect, func (s, t IntSet) IntSet {
		return Intersect(s, t)
	})
}

func TestIntersectReverse (t *testing.T) {
	checkLogic(t, logicIntersect, func (s, t IntSet) IntSet {
		return Intersect(t, s)
	})
}

func TestSubtract (t *testing.T) {
	checkLogic(t, logicSubtract, func (s, t IntSet) IntSet {
		return Subtract(s, t)
	})
}

type logicFunc = func (s, t IntSet) IntSet

func TestLogicFunctionsHaveNoSideEffects (t *testing.T) {
	s1 := NewIntSet(1, 2, 3)
	s2 := NewIntSet(3, 4, 5)
	funcs := []logicFunc{Union, Intersect, Subtract}
	for n, f := range funcs {
		f(s1, s2)
		assertItemSuite(t, s1, []int{1, 2, 3}, n)
		assertItemSuite(t, s2, []int{3, 4, 5}, n)
	}
}

func TestLogicMethodsHaveSideEffects (t *testing.T) {
	unionFunc := func (s, t IntSet) IntSet {
		return s.Union(t)
	}
	intersectFunc := func (s, t IntSet) IntSet {
		return s.Intersect(t)
	}
	subtractFunc := func (s, t IntSet) IntSet {
		return s.Subtract(t)
	}

	samples := []struct {
		f logicFunc
		r []int
	}{
		{unionFunc, []int{1, 2, 3, 4, 5}},
		{intersectFunc, []int{3}},
		{subtractFunc, []int{1, 2}},
	}

	for n, s := range samples {
		s1 := NewIntSet(1, 2, 3)
		s2 := NewIntSet(3, 4, 5)
		r := s.f(s1, s2)
		assertItemSuite(t, r, s.r, n)
		assertItemSuite(t, s1, s.r, n)
		assertItemSuite(t, s2, []int{3, 4, 5}, n)
	}
}
