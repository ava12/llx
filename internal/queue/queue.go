package queue

const minSize = 3

type Queue[T any] struct {
	items      []T
	size       int
	head, tail int
	zero       T
}

func New[T any] (items ...T) *Queue[T] {
	result := &Queue[T]{}
	l := len(items)
	result.tail = l
	result.size = computeSize(l)
	result.items = make([]T, result.size + 1)
	copy(result.items, items)
	return result
}

func (q *Queue[T]) IsEmpty () bool {
	return q.head == q.tail
}

func (q *Queue[T]) Len () int {
	return (q.tail + q.size + 1 - q.head) & q.size
}

func (q *Queue[T]) Items () []T {
	if q.tail >= q.head {
		return q.items[q.head : q.tail]
	}

	l := q.Len()
	result := make([]T, l)
	copy(result, q.items[q.head : q.size + 1])
	copy(result[q.size - q.head + 1 :], q.items[: q.tail])
	return result
}

func (q *Queue[T]) Append (item T) *Queue[T] {
	q.items[q.tail] = item
	q.tail = (q.tail + 1) & q.size
	if q.tail == q.head {
		q.grow()
	}
	return q
}

func (q *Queue[T]) Prepend (item T) *Queue[T] {
	q.head = (q.head - 1) & q.size
	q.items[q.head] = item
	if q.head == q.tail {
		q.grow()
	}
	return q
}

func (q *Queue[T]) First () (T, bool) {
	if q.head == q.tail {
		return q.zero, false
	}

	result := q.items[q.head]
	q.items[q.head] = q.zero
	q.head = (q.head + 1) & q.size

	if q.head == 0 && q.size > minSize && (q.tail << 2) <= q.size {
		q.size = computeSize(q.tail << 1)
		items := make([]T, q.size + 1)
		copy(items, q.items[: q.tail])
		q.items = items
	}

	return result, true
}

func (q *Queue[T]) Last () (T, bool) {
	if q.head == q.tail {
		return q.zero, false
	}

	q.tail = (q.tail - 1) & q.size
	result := q.items[q.tail]
	q.items[q.tail] = q.zero
	return result, true
}

func computeSize (length int) (size int) {
	if length <= minSize {
		size = minSize
	} else {
		length |= length >> 1
		length |= length >> 2
		length |= length >> 4
		length |= length >> 8
		size = length | length >> 16
	}
	return
}

func (q *Queue[T]) grow () {
	items := make([]T, (q.size + 1) << 1)
	copy(items, q.items[q.head :])
	if q.head > 0 {
		copy(items[q.size + 1 - q.head :], q.items[0 : q.head])
	}
	q.head = 0
	q.tail = q.size + 1
	q.size = q.size + q.tail
	q.items = items
}
