package llx

type queueRec struct {
	items      []int
	size       int
	head, tail int
}

type IntQueue = *queueRec

func NewIntQueue (items ...int) IntQueue {
	result := &queueRec{}
	l := len(items)
	if l == 0 {
		result.size = 3
	} else {
		result.tail = l
		l++
		l |= l >> 1
		l |= l >> 2
		l |= l >> 4
		l |= l >> 8
		result.size = l | l >> 16
	}
	result.items = make([]int, result.size + 1)
	copy(result.items, items)
	return result
}

func (q *queueRec) IsEmpty () bool {
	return (q.head == q.tail)
}

func (q *queueRec) Len () int {
	return (q.tail + q.size + 1 - q.head) & q.size
}

func (q *queueRec) Items () []int {
	if q.tail >= q.head {
		return q.items[q.head : q.tail]
	}

	l := q.Len()
	result := make([]int, l)
	copy(result, q.items[q.head : q.size + 1])
	copy(result[q.size - q.head :], q.items[: q.tail])
	return result
}

func (q *queueRec) resize () {
	items := make([]int, (q.size + 1) << 1)
	copy(items, q.items[q.head :])
	if q.head > 0 {
		copy(items[q.size + 1 - q.head :], q.items[0 : q.head])
	}
	q.head = 0
	q.tail = q.size + 1
	q.size = q.size + q.tail
	q.items = items
}

func (q *queueRec) Append (item int) IntQueue {
	q.items[q.tail] = item
	q.tail = (q.tail + 1) & q.size
	if q.tail == q.head {
		q.resize()
	}
	return q
}

func (q *queueRec) Prepend (item int) IntQueue {
	q.head = (q.head - 1) & q.size
	q.items[q.head] = item
	if q.head == q.tail {
		q.resize()
	}
	return q
}

func (q *queueRec) Head () int {
	if q.head == q.tail {
		return 0
	}

	result := q.items[q.head]
	q.head = (q.head + 1) & q.size
	return result
}

func (q *queueRec) Tail () int {
	if q.head == q.tail {
		return 0
	}

	q.tail = (q.tail - 1) & q.size
	return q.items[q.tail]
}
