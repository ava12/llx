package queue

import (
	"fmt"
	"testing"

	. "github.com/ava12/llx/internal/test"
)

func TestComputeSize (t *testing.T) {
	for i := 0; i <= 33; i++ {
		name := fmt.Sprintf("%d elements", i)
		t.Run(name, func (t *testing.T) {
			size := computeSize(i)
			Assert(t, size >= minSize, "expecting at least %d, got %d", minSize, size)
			Assert(t, size & (size + 1) == 0, "expecting 2^n - 1, got %b", size)
			Assert(t, size >= i, "expecting size >= %d, got %d", i, size)
			if size > minSize {
				Assert(t, (size >> 1) < i, "expecting size/2 < %d, got size %d", i, size)
			}
		})
	}
}

func TestEmpty (t *testing.T) {
	q := New[int]()
	ExpectInt(t, minSize + 1, len(q.items))
	ExpectInt(t, 0, q.head)
	ExpectInt(t, 0, q.tail)
	ExpectInt(t, minSize, q.size)
}

func TestPrefilled (t *testing.T) {
	items := make([]int, minSize + 1)
	for i := range items {
		items[i] = i
	}

	q := New[int](items[: minSize]...)
	ExpectInt(t, 0, q.head)
	ExpectInt(t, minSize, q.tail)
	ExpectInt(t, minSize, q.size)
	ExpectInt(t, minSize + 1, len(q.items))
	for i := range items[: minSize] {
		ExpectInt(t, i, q.items[i])
	}

	q = New[int](items...)
	ExpectInt(t, 0, q.head)
	ExpectInt(t, minSize + 1, q.tail)
	ExpectInt(t, (minSize << 1) + 1, q.size)
	ExpectInt(t, (minSize << 1) + 2, len(q.items))
	for i := range items {
		ExpectInt(t, i, q.items[i])
	}
}

func TestGrow (t *testing.T) {
	items := make([]int, minSize)
	q := New[int](items ...)
	ExpectInt(t, minSize, q.size)
	q.Append(1)
	newSize := (minSize << 1) + 1
	ExpectInt(t, newSize, q.size)
	for i := 0; i < minSize; i++ {
		q.Append(i)
		ExpectInt(t, newSize, q.size)
	}
	q.Append(1)
	ExpectInt(t, (newSize << 1) + 1, q.size)
}

func TestShrink (t *testing.T) {
	halfSize := (minSize << 1) + 1
	fullSize := (halfSize << 1) + 1
	items := make([]int, fullSize)
	q := New[int](items ...)
	ExpectInt(t, fullSize, q.size)

	q.tail = minSize + 1
	q.head = fullSize
	q.First()
	ExpectInt(t, fullSize, q.size)

	q.tail = minSize
	q.head = fullSize - 1
	q.First()
	ExpectInt(t, fullSize, q.size)
	q.First()
	ExpectInt(t, halfSize, q.size)

	q.tail = 1
	q.head = q.size
	q.First()
	ExpectInt(t, minSize, q.size)
}

func TestIsEmpty (t *testing.T) {
	q := New[int]()
	ExpectBool(t, true, q.IsEmpty())
	q.Append(1)
	ExpectBool(t, false, q.IsEmpty())
	q.First()
	ExpectBool(t, true, q.IsEmpty())
	q = New[int](1)
	ExpectBool(t, false, q.IsEmpty())
}

func TestLen (t *testing.T) {
	l := (minSize << 1) + 2
	samples := []struct {
		head, tail, l int
	}{
		{0, 1, 1},
		{1, 1, 0},
		{l - 2, 1, 3},
	}

	items := make([]int, l - 1)
	q := New[int](items ...)
	for i, s := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func (t *testing.T) {
			q.head = s.head
			q.tail = s.tail
			ExpectInt(t, s.l, q.Len())
		})
	}
}

func TestItems (t *testing.T) {
	l := (minSize << 1) + 2
	samples := []struct {
		head, tail, l int
	}{
		{0, 1, 1},
		{1, 1, 0},
		{2, 0, l - 2},
		{l - 2, 2, 4},
	}

	items := make([]int, l)
	for i := range items {
		items[i] = i
	}
	q := New[int]()
	q.items = items
	q.size = l - 1

	for i, s := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func (t *testing.T) {
			q.head = s.head
			q.tail = s.tail
			items := q.Items()
			ExpectInt(t, s.l, len(items))
			v := s.head
			for _, i := range items {
				ExpectInt(t, v, i)
				v = (v + 1) & q.size
			}
		})
	}
}

func TestAppend (t *testing.T) {
	q := New[int]()

	q.Append(11)
	ExpectInt(t, 0, q.head)
	ExpectInt(t, 1, q.tail)
	ExpectInt(t, 11, q.items[0])

	q.Append(12)
	ExpectInt(t, 0, q.head)
	ExpectInt(t, 2, q.tail)
	ExpectInt(t, 12, q.items[1])

	q.head = minSize
	q.tail = minSize
	q.Append(13)
	ExpectInt(t, minSize, q.head)
	ExpectInt(t, 0, q.tail)
	ExpectInt(t, 13, q.items[minSize])

	q.head = 1
	q.tail = 0
	q.Append(14)
	ExpectInt(t, (minSize << 1) + 1, q.size)
	ExpectInt(t, 0, q.head)
	ExpectInt(t, minSize + 1, q.tail)
	ExpectInt(t, 12, q.items[0])
	ExpectInt(t, 14, q.items[minSize])
}

func TestPrepend (t *testing.T) {
	q := New[int]()

	q.Prepend(11)
	ExpectInt(t, minSize, q.head)
	ExpectInt(t, 0, q.tail)
	ExpectInt(t, 11, q.items[minSize])

	q.Prepend(12)
	ExpectInt(t, minSize - 1, q.head)
	ExpectInt(t, 0, q.tail)
	ExpectInt(t, 12, q.items[q.head])

	q.head = 1
	q.tail = 0
	q.Prepend(13)
	ExpectInt(t, (minSize << 1) + 1, q.size)
	ExpectInt(t, 0, q.head)
	ExpectInt(t, minSize + 1, q.tail)
	ExpectInt(t, 13, q.items[q.head])
}

func TestFirst (t *testing.T) {
	q := New[int]()
	for i := range q.items {
		q.items[i] = i + 10
	}

	i, f := q.First()
	ExpectInt(t, 0, i)
	ExpectBool(t, false, f)

	q.tail = 2
	i, f = q.First()
	ExpectInt(t, 10, i)
	ExpectBool(t, true, f)
	ExpectInt(t, 1, q.head)
	ExpectInt(t, 2, q.tail)

	q.tail = q.head
	i, f = q.First()
	ExpectInt(t, 0, i)
	ExpectBool(t, false, f)

	q.head = minSize
	q.tail = 1
	i, f = q.First()
	ExpectInt(t, 10 + minSize, i)
	ExpectBool(t, true, f)
	ExpectInt(t, 0, q.head)
	ExpectInt(t, 1, q.tail)
}

func TestLast (t *testing.T) {
	q := New[int]()
	for i := range q.items {
		q.items[i] = i + 10
	}

	i, f := q.Last()
	ExpectInt(t, 0, i)
	ExpectBool(t, false, f)

	q.tail = 2
	i, f = q.Last()
	ExpectInt(t, 11, i)
	ExpectBool(t, true, f)
	ExpectInt(t, 0, q.head)
	ExpectInt(t, 1, q.tail)

	q.tail = q.head
	i, f = q.Last()
	ExpectInt(t, 0, i)
	ExpectBool(t, false, f)

	q.head = minSize
	q.tail = 1
	i, f = q.Last()
	ExpectInt(t, 10, i)
	ExpectBool(t, true, f)
	ExpectInt(t, minSize, q.head)
	ExpectInt(t, 0, q.tail)
}
