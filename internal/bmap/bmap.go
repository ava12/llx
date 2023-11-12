// Package bmap implements basic map with []byte key type.
package bmap

import (
	"bytes"
	"hash/maphash"
	"math/bits"
)

var seed = maphash.MakeSeed()

const minSize = 6

type entry[T any] struct {
	keyOffset, keyLen int
	nextIndex         int
	value             T
}

// BMap implements generic hashmap with []byte key type.
// It is intended to store a small fixed set of keys and it has some limitations:
// keys cannot be deleted, number of keys cannot exceed defined limit.
// Added keys are copied into internal byte slice for safety.
// Implementation is intended to be as simple (and bug-free) as possible.
// Uses custom implementation of hashmap, collisions are resolved by chaining.
type BMap[T any] struct {
	keys       []byte
	values     []entry[T]
	index      []int
	mask       uint64
	zero       T
	maxSize    int
	hasZeroKey bool
}

// New create bytes map. size defines maximum number of stored keys (not counting empty key).
func New[T any](size int) *BMap[T] {
	if size < minSize {
		size = minSize
	}
	l := bits.Len(uint(size)*8/3 - 1)
	return &BMap[T]{
		values:  make([]entry[T], 1, size+1),
		index:   make([]int, 1<<l),
		mask:    (1 << l) - 1,
		maxSize: size,
	}
}

func (m *BMap[T]) find(key []byte, hash uint64) (*entry[T], bool) {
	var en *entry[T]

	i := m.index[int(hash&m.mask)]
	for i != 0 {
		en = &m.values[i]
		found := bytes.Compare(key, m.keys[en.keyOffset:en.keyOffset+en.keyLen]) == 0
		if found {
			return en, true
		}

		i = en.nextIndex
	}

	return en, false
}

// Get returns stored value by key and a flag telling whether this key is stored in the map.
// Returns zero value if the key is not present.
func (m *BMap[T]) Get(key []byte) (T, bool) {
	if len(key) == 0 {
		return m.values[0].value, m.hasZeroKey
	}

	hash := maphash.Bytes(seed, key)
	en, found := m.find(key, hash)
	if found {
		return en.value, true
	} else {
		return m.zero, false
	}
}

// Set adds or rewrites value for given key.
// Panics if trying to add a new key to the map containing maximum number of keys.
func (m *BMap[T]) Set(key []byte, value T) {
	if len(key) == 0 {
		m.values[0].value = value
		m.hasZeroKey = true
		return
	}

	hash := maphash.Bytes(seed, key)
	en, found := m.find(key, hash)
	if found {
		en.value = value
		return
	}

	i := len(m.values)
	if i > m.maxSize {
		panic("bytes map is full")
	}

	o := len(m.keys)
	m.keys = append(m.keys, key...)
	m.values = append(m.values, entry[T]{
		keyOffset: o,
		keyLen:    len(key),
		value:     value,
	})
	if en == nil {
		m.index[hash&m.mask] = i
	} else {
		en.nextIndex = i
	}
}
