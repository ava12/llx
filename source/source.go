package source

import (
	"bytes"
	"unicode/utf8"

	"github.com/ava12/llx/internal/queue"
)

type Source struct {
	name          string
	content       []byte
	lineStarts    []int
	prevLineIndex int
}

func New (name string, content []byte) *Source {
	s := &Source{name: name, content: content, prevLineIndex: -1}
	lineCnt := bytes.Count(content, []byte("\n")) + 1
	s.lineStarts = make([]int, lineCnt, lineCnt)
	s.lineStarts[0] = 0
	j := 1
	for i := 0; i < len(content) && j < lineCnt; i++ {
		if content[i] == '\n' {
			s.lineStarts[j] = i + 1
			j++
		}
	}

	return s
}

func (s *Source) Name () string {
	return s.name
}

func (s *Source) Content () []byte {
	return s.content
}

func (s *Source) Len () int {
	return len(s.content)
}

func (s *Source) LineCol (pos int) (line, col int) {
	var lineIndex int
	if pos < 0 {
		pos = 0
		lineIndex = 0
	} else if pos >= len(s.content) {
		pos = len(s.content)
		lineIndex = len(s.lineStarts) - 1
	} else {
		lineIndex = s.findLineIndex(pos)
	}

	lineStart := s.lineStarts[lineIndex]
	return lineIndex + 1, utf8.RuneCount(s.content[lineStart : pos]) + 1
}

func (s *Source) Pos (line, col int) int {
	if line <= 0 || col <= 0 {
		return 0
	}

	l := len(s.content)
	if line > len(s.lineStarts) {
		return l
	}

	res := s.lineStarts[line - 1] + col - 1
	if res > l {
		return l
	} else {
		return res
	}
}

func (s *Source) findLineIndex (pos int) int {
	if s.prevLineIndex >= 0 && s.lineStarts[s.prevLineIndex] <= pos {
		lineIndex := s.prevLineIndex
		last := len(s.lineStarts) - 1
		for lineIndex <= last && s.lineStarts[lineIndex] <= pos {
			lineIndex++
		}
		lineIndex--
		s.prevLineIndex = lineIndex
		return lineIndex
	}

	lineStart := 0
	leftIndex := 0
	rightIndex := len(s.lineStarts) - 1
	index := 0
	if s.prevLineIndex >= 0 {
		lineStart = s.lineStarts[s.prevLineIndex]
		rightIndex = s.prevLineIndex
	}
	for leftIndex < rightIndex {
		index = (leftIndex + rightIndex + 1) >> 1
		lineStart = s.lineStarts[index]
		if lineStart == pos {
			return index
		}

		if lineStart < pos {
			leftIndex = index
		} else {
			rightIndex = index - 1
			index = rightIndex
		}
	}
	s.prevLineIndex = index
	return index
}

type Pos struct {
	src            *Source
	pos, line, col int
}

func (p Pos) Source () *Source {
	return p.src
}

func (p Pos) SourceName () string {
	return p.src.Name()
}

func (p Pos) Pos () int {
	return p.pos
}

func (p Pos) Line () int {
	return p.line
}

func (p Pos) Col () int {
	return p.col
}


type queueItem struct {
	source *Source
	pos    int
}

type Queue struct {
	q      *queue.Queue[queueItem]
	source *Source
	pos    int
}

func NewQueue () *Queue {
	return &Queue{queue.New[queueItem](), nil, 0}
}

func (q *Queue) Source () *Source {
	return q.source
}

func (q *Queue) SourceName () string {
	if q.source == nil {
		return ""
	} else {
		return q.source.Name()
	}
}

func (q *Queue) Pos () int {
	return q.pos
}

func (q *Queue) SourcePos () Pos {
	res := Pos{q.source, q.pos, 0, 0}
	if q.source != nil {
		res.line, res.col = q.source.LineCol(q.pos)
	}
	return res
}

func (q *Queue) NextSource () bool {
	qi, fetched := q.q.First()
	if !fetched {
		q.source = nil
		q.pos = 0
	} else {
		q.source = qi.source
		q.pos = qi.pos
	}
	return fetched
}

func (q *Queue) Append (s *Source) *Queue {
	if s == nil || s.Len() == 0 && q.source != nil && q.source.Len() != 0 {
		return q
	}

	if q.source == nil || q.source.Len() == 0 {
		q.source = s
		q.pos = 0
	} else {
		q.q.Append(queueItem{s, 0})
	}
	return q
}

func (q *Queue) Prepend (s *Source) *Queue {
	if s == nil || s.Len() == 0 && q.source != nil && q.source.Len() > 0 {
		return q
	}

	if q.source != nil && q.source.Len() > 0 {
		q.q.Prepend(queueItem{q.source, q.pos})
	}

	q.source = s
	q.pos = 0

	return q
}

func (q *Queue) IsEmpty () bool {
	return q.source == nil
}

func (q *Queue) Eof () bool {
	return q.source == nil || q.pos >= q.source.Len()
}

func (q *Queue) ContentPos () ([]byte, int) {
	if q.source == nil {
		return nil, 0
	} else {
		return q.source.Content(), q.pos
	}
}

func (q *Queue) Skip (size int) {
	if q.source == nil || size <= 0 {
		return
	}

	q.pos += size
	if q.pos >= q.source.Len() {
		q.pos = q.source.Len()
	}
}

func (q *Queue) Rewind (size int) {
	if q.source == nil {
		return
	}

	if q.pos <= size {
		q.pos = 0
	} else {
		q.pos -= size
	}
}

func (q *Queue) Seek (pos int) {
	if q.source == nil {
		return
	}

	if pos <= 0 {
		q.pos = 0
	} else {
		size := q.source.Len()
		if pos > size {
			q.pos = size
		} else {
			q.pos = pos
		}
	}
}

func (q *Queue) LineCol (pos int) (line, col int) {
	if q.source == nil {
		return 0, 0
	} else {
		return q.source.LineCol(pos)
	}
}


func NormalizeNls (content *[]byte) {
	const (
		LF = 10
		CR = 13
	)

	wPos := 0
	rPos := 0
	crFound := false

	for i, b := range *content {
		switch b {
		case LF:
			if crFound {
				crFound = false
				if rPos != 0 {
					copy((*content)[wPos :], (*content)[rPos : i])
				}
				wPos += i - rPos
				rPos = i + 1
			}

		case CR:
			crFound = true
			(*content)[i] = LF

		default:
			crFound = false
		}
	}

	l := len(*content)
	if rPos != 0 && rPos < l {
		copy((*content)[wPos :], (*content)[rPos : l])
	}
	*content = (*content)[ : l - rPos + wPos]
}
