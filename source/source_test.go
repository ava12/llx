package source

import (
	"strconv"
	"testing"
)

type result struct {
	pos, line, col int
}

func TestSourceLineCol (t *testing.T) {
	samples := map[string][]result{
		"": {
			{0, 1, 1},
			{100, 1, 1},
			{100, 1, 1},
		},
		"\n": {
			{0, 1, 1},
			{1, 2, 1},
			{1, 2, 1},
			{1, 2, 1},
			{100, 2, 1},
			{100, 2, 1},
		},
		"0\n2\n4\n6789abcde\ng\ni\n": {
			{4, 3, 1},
			{5, 3, 2},
			{6, 4, 1},
			{7, 4, 2},
			{8, 4, 3},
			{9, 4, 4},
			{10, 4, 5},
			{11, 4, 6},
			{12, 4, 7},
			{13, 4, 8},
			{14, 4, 9},
			{19, 6, 2},
			{20, 7, 1},
			{9, 4, 4},
			{5, 3, 2},
		},
	}

	for text, results := range samples {
		source := New("", []byte(text))
		for _, res := range results {
			l, c := source.LineCol(res.pos)
			if l != res.line || c != res.col {
				t.Errorf("sample %q: expected %v, got line: %d, col: %d", text, res, l, c)
			}
		}
	}
}

func TestSourcePos (t *testing.T) {
	samples := map[string][]result{
		"": {
			{0, 0, 1},
			{0, 1, 0},
			{0, 1, 1},
			{0, 1, 2},
			{0, 2, 1},
		},
		" ": {
			{0, 0, 1},
			{0, 1, 0},
			{0, 1, 1},
			{1, 1, 2},
			{1, 2, 1},
		},
		"\n": {
			{0, 0, 1},
			{0, 1, 0},
			{0, 1, 1},
			{1, 1, 2},
			{1, 2, 1},
			{1, 2, 2},
			{1, 3, 1},
		},
		"hello\nworld\n": {
			{0, 0, 1},
			{0, 1, 0},
			{0, 1, 1},
			{1, 1, 2},
			{6, 2, 1},
			{7, 2, 2},
			{12, 2, 10},
			{12, 3, 1},
			{12, 3, 2},
			{12, 4, 1},
		},
	}

	for text, results := range samples {
		source := New("", []byte(text))
		for _, res := range results {
			p := source.Pos(res.line, res.col)
			if p != res.pos {
				t.Errorf("sample %q: expected %v, got pos: %d", text, res, p)
			}
		}
	}
}

func TestSkipNotAdvancesSource (t *testing.T) {
	q := NewQueue().Append(src("bar"))
	q.Skip(2)
	c, p := q.ContentPos()
	assert(t, string(c) == "bar", "expecting bar, got " + string(c))
	assert(t, p == 2, "expecting pos=2, got " + strconv.Itoa(p))

	q.Prepend(src("foo"))
	c, p = q.ContentPos()
	assert(t, string(c) == "foo", "expecting foo, got " + string(c))
	assert(t, p == 0, "expecting pos=0, got " + strconv.Itoa(p))

	q.Skip(4)
	c, p = q.ContentPos()
	assert(t, string(c) == "foo", "expecting foo, got " + string(c))
	assert(t, p == 3, "expecting pos=3, got " + strconv.Itoa(p))
}

func TestSeekAfterEof (t *testing.T) {
	q := NewQueue().Append(src("foo"))
	q.Seek(4)
	p := q.Pos()
	assert(t, p == 3, "expecting pos=3, got " + strconv.Itoa(p))
	assert(t, q.Eof(), "expecting EoF")

	q.Seek(2)
	p = q.Pos()
	assert(t, p == 2, "expecting pos=2, got " + strconv.Itoa(p))
	assert(t, !q.Eof(), "expecting no EoF")

	q.Skip(4)
	p = q.Pos()
	assert(t, p == 3, "expecting pos=3 again, got " + strconv.Itoa(p))
	assert(t, q.Eof(), "expecting EoF again")

	q.Rewind(2)
	p = q.Pos()
	assert(t, p == 1, "expecting pos=1, got " + strconv.Itoa(p))
	assert(t, !q.IsEmpty(), "expecting no EoF again")
}

func assert (t *testing.T, flag bool, message string) {
	if !flag {
		if message == "" {
			t.Fail()
		} else {
			t.Fatal(message)
		}
	}
}

func sourceChain (queue *Queue) []string {
	res := []string{}
	for {
		content, pos := queue.ContentPos()
		src := string(content[pos :])
		if src == "" {
			return res
		}

		res = append(res, src)
		queue.NextSource()
	}
}
/*
func nameChain (queue *Queue) []string {
	res := []string{}
	for {
		res = append(res, queue.Source().Name())
		content, pos := queue.ContentPos()
		src := string(content[pos :])
		if src == "" {
			return res
		}

		queue.Skip(len(src))
	}
}
*/
func cmp (a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func assertChain (t *testing.T, expected, got []string) {
	if cmp(expected, got) {
		return
	}

	t.Fatalf("expected: %v, got: %v", expected, got)
}

func src (content string) *Source {
	return New(content, []byte(content))
}

func TestSourceOrder (t *testing.T) {
	queue := NewQueue()
	queue.Append(src("bar")).Append(src("baz")).Prepend(src("foo"))
	assertChain(t, []string{"foo", "bar", "baz"}, sourceChain(queue))
	content, pos := queue.ContentPos()
	assert(t, string(content[pos :]) == "", "non-empty content after EoF")
	assert(t, queue.IsEmpty(), "queue not empty after EoF")
	src := queue.Source()
	assert(t, src == nil, "non-empty source after EoF")
}

func TestSourceInsert (t *testing.T) {
	queue := NewQueue()
	queue.Append(src("hello")).Append(src("world"))
	queue.Skip(3)
	queue.Prepend(src("hi"))
	assertChain(t, []string{"hi", "lo", "world"}, sourceChain(queue))
}


func TestEmptySource (t *testing.T) {
	queue := NewQueue()

	emptySrc := func (name string) *Source {
		return New(name, []byte{})
	}

	assertSourceName := func (name string) {
		src := queue.Source()
		if src == nil {
			t.Fatalf("expecting source \"%s\", got nil", name)
		}
		got := src.Name()
		if got != name {
			t.Fatalf("expecting source \"%s\", got \"%s\"", name, got)
		}
	}

	assert(t, queue.Source() == nil, "source is not nil")
	queue.Append(emptySrc("foo"))
	assert(t, queue.Source() != nil, "source is nil")
	assertSourceName("foo")
	queue.Prepend(emptySrc("bar"))
	assertSourceName("bar")
	queue.Append(emptySrc("baz"))
	assertSourceName("baz")
}

func TestResizeSource (t *testing.T) {
	queue := NewQueue()
	queue.Append(src("c")).Append(src("d")).Append(src("e")).Append(src("f")).
		Append(src("g")).Prepend(src("b")).Append(src("h")).Prepend(src("a"))
	assertChain(t, []string{"a", "b", "c", "d", "e", "f", "g", "h"}, sourceChain(queue))
}

func TestNextSource (t *testing.T) {
	sources := []string {
		"foo",
		"bar",
		"baz",
	}

	queue := NewQueue()
	for _, src := range sources {
		queue.Append(New(src, []byte(src)))
	}

	for i := 1; i < len(sources); i++ {
		f := queue.NextSource()
		if !f {
			t.Fatalf("unexpected false returned from NextSource()")
		}

		pos := queue.SourcePos()
		if pos.SourceName() != sources[i] || pos.Pos() != 0 {
			t.Fatalf("expecting %q source at pos 0, got %q at pos %d", sources[i], pos.SourceName(), pos.pos)
		}
	}

	f := queue.NextSource()
	eoi := queue.IsEmpty()
	if f || !eoi {
		t.Fatalf("expecting (false, true), got (%v, %v)", f, eoi)
	}
}

func TestAddSourceAfterEof (t *testing.T) {
	queue := NewQueue().Append(New("dropped", []byte("-")))
	queue.NextSource()
	queue.Append(New("appended", []byte("foo")))
	pos := queue.SourcePos()
	if pos.SourceName() != "appended" || pos.pos != 0 {
		t.Errorf("expecting appended source pos 0, got %s source pos %d", pos.SourceName(), pos.pos)
	}

	queue = NewQueue().Append(New("dropped", []byte("-")))
	queue.NextSource()
	queue.Prepend(New("prepended", []byte("bar")))
	pos = queue.SourcePos()
	if pos.SourceName() != "prepended" || pos.pos != 0 {
		t.Errorf("expecting prepended source pos 0, got %s source pos %d", pos.SourceName(), pos.pos)
	}
}

func TestNormalizeNls (t *testing.T) {
	samples := []struct {
		src, res string
	}{
		{"foo bar\tbaz", "foo bar\tbaz"},
		{"\nfoo\nbar\n\nbaz\n", "\nfoo\nbar\n\nbaz\n"},
		{"\rfoo\rbar\r\rbaz\r", "\nfoo\nbar\n\nbaz\n"},
		{"\r\nfoo\r\nbar\r\n\r\nbaz\r\n", "\nfoo\nbar\n\nbaz\n"},
		{"\r\nfoo\nbar\r\r\nbaz\r\n", "\nfoo\nbar\n\nbaz\n"},
		{"\n\r\r\r\n\n\n\r\n\r", "\n\n\n\n\n\n\n\n"},
	}

	for i, s := range samples {
		c := []byte(s.src)
		NormalizeNls(&c)
		cs := string(c)
		if cs != s.res {
			t.Errorf("sample #%d: expecting %q, got %q", i, s.res, cs)
		}
	}
}
