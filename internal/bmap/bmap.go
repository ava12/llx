// Package bmap implements basic map with []byte key type.
package bmap

import (
	"unsafe"
)

// BMap implements generic hashmap with []byte key type.
// It is intended to store a small fixed set of keys and it has some limitations:
// keys cannot be deleted.
// Added keys are copied into internal byte slice for safety.
// Implementation is intended to be as simple (and bug-free) as possible.
// Uses map with string keys internally.
type BMap[T any] struct {
	keys []byte
	smap map[string]T
}

// New create bytes map. size defines maximum number of stored keys (not counting empty key).
func New[T any](size int) *BMap[T] {
	return &BMap[T]{
		smap: make(map[string]T, size),
	}
}

// Get returns stored value by key and a flag telling whether this key is stored in the map.
// Returns zero value if the key is not present.
func (m *BMap[T]) Get(key []byte) (T, bool) {
	skey := ""
	if len(key) != 0 {
		skey = unsafe.String(&key[0], len(key))
	}
	result, has := m.smap[skey]
	return result, has
}

// Set adds or rewrites value for given key.
func (m *BMap[T]) Set(key []byte, value T) {
	skey := ""
	_, has := m.Get(key)
	if !has && len(key) != 0 {
		ofs := len(m.keys)
		m.keys = append(m.keys, key...)
		key = m.keys[ofs : ofs+len(key)]
	}

	if len(key) != 0 {
		skey = unsafe.String(&key[0], len(key))
	}
	m.smap[skey] = value
}
