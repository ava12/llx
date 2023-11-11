package ints

const IntSizeShift = 5 + (^uint(0) >> 32 & 1)
const IntSize = 1 << IntSizeShift

type Set struct {
	lowItem, highItem int
	chunks            []uint
}

func countBits(chunk uint) int {
	result := 0
	for chunk != 0 {
		result++
		chunk &= (chunk - 1)
	}
	return result
}

func NewSet(items ...int) *Set {
	result := &Set{0, 0, []uint{}}
	if len(items) > 0 {
		result.Add(items...)
	}
	return result
}

func FromSlice(items []int) *Set {
	return NewSet(items...)
}

func (s *Set) ToSlice() []int {
	bitCnt := 0
	for _, chunk := range s.chunks {
		bitCnt += countBits(chunk)
	}
	result := make([]int, bitCnt)
	item := s.lowItem
	index := 0
	for _, chunk := range s.chunks {
		for i := IntSize; i > 0; i-- {
			if chunk&1 != 0 {
				result[index] = item
				index++
			}
			item++
			chunk = chunk >> 1
		}
	}
	return result
}

func (s *Set) baseItem(item int) int {
	return item & ^(IntSize - 1)
}

func (s *Set) allocate(low, high int) {
	lowItem := s.baseItem(low)
	highItem := s.baseItem(high) + IntSize
	if lowItem >= s.lowItem && highItem <= s.highItem {
		return
	}

	if lowItem > s.lowItem {
		lowItem = s.lowItem
	}
	if highItem < s.highItem {
		highItem = s.highItem
	}

	chunkCnt := (highItem - lowItem) >> IntSizeShift
	chunks := make([]uint, chunkCnt)
	if s.lowItem != 0 || s.highItem != 0 {
		offset := (s.lowItem - lowItem) >> IntSizeShift
		copy(chunks[offset:], s.chunks)
	}
	s.chunks = chunks
	s.lowItem = lowItem
	s.highItem = highItem
}

func (s *Set) chunkIndex(item int) int {
	return (item - s.lowItem) >> IntSizeShift
}

func bitMask(item int) uint {
	return 1 << (uint(item) & (IntSize - 1))
}

func (s *Set) doSet(item int, invert bool) {
	if invert {
		s.chunks[s.chunkIndex(item)] &= ^bitMask(item)
	} else {
		s.chunks[s.chunkIndex(item)] |= bitMask(item)
	}
}

func minMax(items []int) (min, max int) {
	min = items[0]
	max = items[0]
	for i := 1; i < len(items); i++ {
		item := items[i]
		if item < min {
			min = item
		}
		if item > max {
			max = item
		}
	}
	return
}

func (s *Set) Add(items ...int) *Set {
	if len(items) == 0 {
		return s
	}

	min, max := minMax(items)
	s.allocate(min, max)
	for _, item := range items {
		s.doSet(item, false)
	}
	return s
}

func (s *Set) Remove(items ...int) *Set {
	if len(items) == 0 {
		return s
	}

	min, max := minMax(items)
	s.allocate(min, max)
	for _, item := range items {
		s.doSet(item, true)
	}
	return s
}

func (s *Set) Contains(item int) bool {
	if item < s.lowItem || item >= s.highItem {
		return false
	} else {
		return (s.chunks[s.chunkIndex(item)]&bitMask(item) != 0)
	}
}

func (s *Set) Copy() *Set {
	items := make([]uint, len(s.chunks))
	copy(items, s.chunks)
	return &Set{s.lowItem, s.highItem, items}
}

func isEmpty(chunks []uint) bool {
	for _, chunk := range chunks {
		if chunk != 0 {
			return false
		}
	}

	return true
}

func (s *Set) IsEmpty() bool {
	return isEmpty(s.chunks)
}

func (s *Set) IsEqual(t *Set) bool {
	var low, high, i int

	if s.lowItem < t.lowItem {
		low = t.lowItem
		if !isEmpty(s.chunks[:(low-s.lowItem)>>IntSizeShift]) {
			return false
		}
	} else {
		low = s.lowItem
		if !isEmpty(t.chunks[:(low-t.lowItem)>>IntSizeShift]) {
			return false
		}
	}

	if s.highItem > t.highItem {
		high = t.highItem
		i = len(s.chunks) - ((s.highItem - high) >> IntSizeShift)
		if !isEmpty(s.chunks[i:]) {
			return false
		}
	} else {
		high = s.highItem
		i = len(t.chunks) - ((t.highItem - high) >> IntSizeShift)
		if !isEmpty(t.chunks[i:]) {
			return false
		}
	}

	firstIndex := (low - s.lowItem) >> IntSizeShift
	lastIndex := firstIndex + ((high - low) >> IntSizeShift)
	offset := (low - t.lowItem) >> IntSizeShift
	for i = firstIndex; i < lastIndex; i++ {
		if s.chunks[i] != t.chunks[offset] {
			return false
		}

		offset++
	}
	return true
}

func (s *Set) fill(t *Set) {
	s.lowItem = t.lowItem
	s.highItem = t.highItem
	s.chunks = t.chunks
}

func (s *Set) Union(t *Set) *Set {
	s.fill(Union(s, t))
	return s
}

func Union(s, t *Set) *Set {
	result := NewSet()

	var low, high int
	if s.lowItem < t.lowItem {
		low = s.lowItem
	} else {
		low = t.lowItem
	}
	if s.highItem > t.highItem {
		high = s.highItem
	} else {
		high = t.highItem
	}

	if low == high {
		return result
	}

	result.allocate(low, high-1)
	offset := (s.lowItem - low) >> IntSizeShift
	copy(result.chunks[offset:], s.chunks)
	offset = (t.lowItem - low) >> IntSizeShift
	for _, chunk := range t.chunks {
		result.chunks[offset] |= chunk
		offset++
	}
	return result
}

func (s *Set) Intersect(t *Set) *Set {
	s.fill(Intersect(s, t))
	return s
}

func Intersect(s, t *Set) *Set {
	result := NewSet()

	var low, high int
	if s.lowItem > t.lowItem {
		low = s.lowItem
	} else {
		low = t.lowItem
	}
	if s.highItem < t.highItem {
		high = s.highItem
	} else {
		high = t.highItem
	}

	if low == high {
		return result
	}

	result.allocate(low, high-1)
	offset := (low - s.lowItem) >> IntSizeShift
	copy(result.chunks, s.chunks[offset:])
	offset = (low - t.lowItem) >> IntSizeShift
	for i := range result.chunks {
		result.chunks[i] &= t.chunks[offset]
		offset++
	}
	return result
}

func (s *Set) Subtract(t *Set) *Set {
	s.fill(Subtract(s, t))
	return s
}

func Subtract(s, t *Set) *Set {
	result := s.Copy()

	var low, high int
	if s.lowItem > t.lowItem {
		low = s.lowItem
	} else {
		low = t.lowItem
	}
	if s.highItem < t.highItem {
		high = s.highItem
	} else {
		high = t.highItem
	}

	if low == high {
		return result
	}

	offset := (low - t.lowItem) >> IntSizeShift
	firstIndex := (low - s.lowItem) >> IntSizeShift
	lastIndex := firstIndex + (high-low)>>IntSizeShift
	for i := firstIndex; i < lastIndex; i++ {
		result.chunks[i] &= ^t.chunks[offset]
		offset++
	}
	return result
}
