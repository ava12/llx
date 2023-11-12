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
	m := New[int](2)
	m.Set([]byte{1}, 1)
	m.Set([]byte{2}, 2)
	m.Set([]byte{1}, 3)

	defer func() {
		recover()
	}()
	m.Set([]byte{3}, 3)
	t.Error("panic expected")
}
