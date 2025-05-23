// Package source defines source file an source queue for parsers.
package source

import (
	"bytes"
	"unicode/utf8"

	"github.com/ava12/llx/internal/queue"
)

// Source represents single source file.
type Source struct {
	name          string
	content       []byte
	lineStarts    []int
	prevLineIndex int
}

// New creates new source.
// Name may be any string identifying the source, does not have to be unique, may be empty.
// Content should be a valid UTF-8 encoded text, lines should be separated by "\n" rune.
// Content should not be modified.
func New(name string, content []byte) *Source {
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

// Name returns source name.
func (s *Source) Name() string {
	return s.name
}

// Content returns source content.
func (s *Source) Content() []byte {
	return s.content
}

// Len returns source content length in bytes.
func (s *Source) Len() int {
	return len(s.content)
}

// LineCol returns line and column number (both 1-based) of rune starting at given position in the source content.
// Negative position is treated as 0, position equal to or higher than length of content is treated
// as position right after EoF.
func (s *Source) LineCol(pos int) (line, col int) {
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
	return lineIndex + 1, utf8.RuneCount(s.content[lineStart:pos]) + 1
}

// Pos returns position in source content corresponding to given line and column.
// Returns 0 for lines or columns < 1. Returns content length for line exceeding total number of lines.
// Returns position after the last rune in line for column exceeding number of runes in line.
func (s *Source) Pos(line, col int) int {
	if line <= 0 || col <= 0 {
		return 0
	}

	l := len(s.content)
	if line > len(s.lineStarts) {
		return l
	}

	res := s.lineStarts[line-1]
	for col > 1 && res < l {
		r, rl := utf8.DecodeRune(s.content[res:])
		if r == '\n' {
			break
		}

		res += rl
		col--
	}
	if res > l {
		res = l
	}
	return res
}

func (s *Source) findLineIndex(pos int) int {
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

// Pos combines captured source, position, line, and column number corresponding to that position.
// Zero value means no source and position information available.
type Pos struct {
	src            *Source
	pos, line, col int
}

// NewPos returns Pos structure. Returns zero value if s is nil.
func NewPos(s *Source, pos int) Pos {
	if s == nil {
		return Pos{}
	}

	l, c := s.LineCol(pos)
	return Pos{s, pos, l, c}
}

// Source returns captured source or nil.
func (p Pos) Source() *Source {
	return p.src
}

// SourceName returns captured source name or empty string.
func (p Pos) SourceName() string {
	if p.src == nil {
		return ""
	} else {
		return p.src.Name()
	}
}

// Pos returns captured position in source or 0.
func (p Pos) Pos() int {
	return p.pos
}

// Line returns captured 1-based line number or 0.
func (p Pos) Line() int {
	return p.line
}

// Col returns captured 1-based column number or 0.
func (p Pos) Col() int {
	return p.col
}

type queueItem struct {
	source *Source
	pos    int
}

// QueueSnapshot contains data used to restore source queue state.
type QueueSnapshot struct {
	items  []queueItem
	source *Source
	pos    int
}

// Queue represents a queue of source files to be processed.
type Queue struct {
	q      *queue.Queue[queueItem]
	source *Source
	pos    int
}

// NewQueue creates empty queue.
func NewQueue() *Queue {
	return &Queue{queue.New[queueItem](), nil, 0}
}

// Source returns current (i.e. first) source in the queue or nil if the queue is empty.
func (q *Queue) Source() *Source {
	return q.source
}

// SourceName returns the name of current source or empty string if the queue is empty.
func (q *Queue) SourceName() string {
	if q.source == nil {
		return ""
	} else {
		return q.source.Name()
	}
}

// Pos returns current position in current source or 0 if the queue is empty.
func (q *Queue) Pos() int {
	return q.pos
}

// SourcePos returns current source and current position in it.
// Returns zero value if the queue is empty.
func (q *Queue) SourcePos() Pos {
	res := Pos{q.source, q.pos, 0, 0}
	if q.source != nil {
		res.line, res.col = q.source.LineCol(q.pos)
	}
	return res
}

// NextSource discards current source from the queue.
// The next source (if there is one) becomes the current one and its saved current position is restored.
// Returns true if the queue is not empty.
func (q *Queue) NextSource() bool {
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

// Append adds new source to the end of the queue.
// Does nothing if s is nil. Does not add empty source if the queue is not empty.
func (q *Queue) Append(s *Source) *Queue {
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

// Prepend adds new source to the beginning of the queue.
// Current position for current source (if there is one) is saved, added source becomes the current one.
// Does nothing if s is nil. Does not add empty source if the queue is not empty.
func (q *Queue) Prepend(s *Source) *Queue {
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

// IsEmpty returns true if the queue is empty (contains no sources) and false otherwise.
func (q *Queue) IsEmpty() bool {
	return q.source == nil
}

// Eof returns true if the queue is empty or current source position is beyond the end of current source.
func (q *Queue) Eof() bool {
	return q.source == nil || q.pos >= q.source.Len()
}

// ContentPos returns content of current source and current position, or (nil, 0) if the queue is empty.
func (q *Queue) ContentPos() ([]byte, int) {
	if q.source == nil {
		return nil, 0
	} else {
		return q.source.Content(), q.pos
	}
}

// Skip increases current source position by given amount of bytes.
// New position will not exceed current source length.
// Does nothing if size is ≤ 0 or the queue is empty.
func (q *Queue) Skip(size int) {
	if q.source == nil || size <= 0 {
		return
	}

	q.pos += size
	if q.pos >= q.source.Len() {
		q.pos = q.source.Len()
	}
}

// Rewind decreases current source position by given amount of bytes.
// New position will be ≥ 0.
// Does nothing if size is ≤ 0 or the queue is empty.
func (q *Queue) Rewind(size int) {
	if q.source == nil || size <= 0 {
		return
	}

	if q.pos <= size {
		q.pos = 0
	} else {
		q.pos -= size
	}
}

// Seek changes current source position to given value.
// New position is adjusted so that 0 ≤ position ≤ content length.
// Does nothing if the queue is empty.
func (q *Queue) Seek(pos int) {
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

// LineCol returns line and column number for current source position.
// Returns (0, 0) if the queue is empty.
func (q *Queue) LineCol(pos int) (line, col int) {
	if q.source == nil {
		return 0, 0
	} else {
		return q.source.LineCol(pos)
	}
}

// Snapshot saves source queue state.
func (q *Queue) Snapshot() QueueSnapshot {
	return QueueSnapshot{q.q.Items(), q.source, q.pos}
}

// Restore restores source queue state.
func (q *Queue) Restore(s QueueSnapshot) {
	q.source = s.source
	q.pos = s.pos
	q.q.Fill(s.items)
}

// NormalizeNls replaces all occurrences of "\r" and "\r\n" with "\n".
func NormalizeNls(content *[]byte) {
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
					copy((*content)[wPos:], (*content)[rPos:i])
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
		copy((*content)[wPos:], (*content)[rPos:l])
	}
	*content = (*content)[:l-rPos+wPos]
}
