package bmap

import (
	. "github.com/ava12/llx/internal/test"
	"testing"
)

func TestEmptyMap(t *testing.T) {
	m := New[int](1)

	en, found := m.Get([]byte{})
	ExpectInt(t, 0, en)
	ExpectBool(t, found, false)

	en, found = m.Get([]byte{1, 2, 3})
	ExpectInt(t, 0, en)
	ExpectBool(t, found, false)
}

func TestEmptyKey(t *testing.T) {
	m := New[int](1)
	empty := []byte{}

	m.Set([]byte("foo"), 123)
	en, found := m.Get(empty)
	ExpectInt(t, 0, en)
	ExpectBool(t, found, false)

	m.Set(empty, 345)
	en, found = m.Get(empty)
	ExpectInt(t, 345, en)
	ExpectBool(t, found, true)
}

func TestKey(t *testing.T) {
	m := New[int](2)
	key := []byte{1, 2, 3}
	key2 := []byte{1, 2}

	m.Set(key, 111)
	m.Set(key2, 222)

	en, found := m.Get(key)
	ExpectInt(t, 111, en)
	ExpectBool(t, found, true)

	key = key[:2]
	en, found = m.Get(key)
	ExpectInt(t, 222, en)
	ExpectBool(t, found, true)
}

func TestOverflow(t *testing.T) {
	m := New[int](0)
	for i := 1; i <= minSize; i++ {
		m.Set([]byte{byte(i)}, i*10)
	}
	m.Set([]byte{1}, 30)

	defer func() {
		recover()
	}()
	m.Set([]byte{100}, 3)
	t.Error("panic expected")
}

func TestChaining(t *testing.T) {
	m := &BMap[int]{
		keys:  []byte("foobarbaz"),
		index: []int{1},
		values: []entry[int]{
			{},
			{keyOffset: 0, keyLen: 3, nextIndex: 2},
			{keyOffset: 3, keyLen: 3, nextIndex: 3},
			{keyOffset: 6, keyLen: 3},
		},
	}

	en, found := m.find([]byte("qux"), 0)
	ExpectBool(t, false, found)
	Expect(t, en == &m.values[3], &m.values[3], en)

	en, found = m.find([]byte("bar"), 0)
	ExpectBool(t, true, found)
	Expect(t, en == &m.values[2], &m.values[2], en)

	en, found = m.find([]byte("baz"), 0)
	ExpectBool(t, true, found)
	Expect(t, en == &m.values[3], &m.values[3], en)
}
