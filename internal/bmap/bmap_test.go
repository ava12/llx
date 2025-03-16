package bmap

import (
	"testing"

	. "github.com/ava12/llx/internal/test"
)

func TestEmptyMap(t *testing.T) {
	m := New[int](1)

	en, found := m.Get([]byte{})
	ExpectInt(t, 0, en)
	ExpectBool(t, found, false)

	en, found = m.GetString("")
	ExpectInt(t, 0, en)
	ExpectBool(t, found, false)

	en, found = m.Get([]byte("123"))
	ExpectInt(t, 0, en)
	ExpectBool(t, found, false)

	en, found = m.GetString("123")
	ExpectInt(t, 0, en)
	ExpectBool(t, found, false)
}

func TestEmptyKey(t *testing.T) {
	m := New[int](1)
	empty := []byte{}

	m.Set([]byte("foo"), 123)
	m.SetString("bar", 234)
	en, found := m.Get(empty)
	ExpectInt(t, 0, en)
	ExpectBool(t, found, false)

	m.Set(empty, 345)
	en, found = m.Get(empty)
	ExpectInt(t, 345, en)
	ExpectBool(t, found, true)
	en, found = m.GetString("")
	ExpectInt(t, 345, en)
	ExpectBool(t, found, true)

	m.SetString("", 456)
	en, found = m.Get(empty)
	ExpectInt(t, 456, en)
	ExpectBool(t, found, true)
	en, found = m.GetString("")
	ExpectInt(t, 456, en)
	ExpectBool(t, found, true)
}

func TestKey(t *testing.T) {
	m := New[int](2)
	key := []byte("123")
	key2 := "12"

	m.Set(key, 111)
	m.SetString(key2, 222)

	en, found := m.Get(key)
	ExpectInt(t, 111, en)
	ExpectBool(t, found, true)

	key = key[:2]
	en, found = m.Get(key)
	ExpectInt(t, 222, en)
	ExpectBool(t, found, true)
}
