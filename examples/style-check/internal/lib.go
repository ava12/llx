//go:generate ../../../bin/llxgen grammar.llx
package internal

import (
	"errors"
	"os"
	"sort"
	"strings"

	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/source"
	"github.com/ava12/llx/tree"
)

const (
	indentType  = "indent"
	spaceType   = "space"
	commentType = "comment"
	//	numberType = "number"
	//	nameType = "name"
	opType = "op"
)

const (
	rootNt    = "cDataGrammar"
	varDefNt  = "var-def"
	typeDefNt = "type-def"
	typeNt    = "type"
	nameNt    = "name"
	//	simpleTypeNt = "simple-type"
	structTypeNt = "struct-type"

	// sizeDefNt = "size-def"
)

const (
	ErrAlign      = "inconsistent struct field align"
	ErrIndent     = "incorrect indent"
	ErrLiteral    = "literal struct types are not allowed, use typedef"
	ErrNoEofNl    = "no newline at file end"
	ErrNoSpace    = "missing space"
	ErrSameLine   = "must be on a separate line"
	ErrSpaces     = "spacing must be exactly one space symbol"
	ErrTab        = "no tabs allowed"
	ErrTrailSpace = "trailing spaces are not allowed"
	ErrWrongSpace = "excess space"
)

type ReportLine struct {
	Line, Col int
	Message   string
}

type reports struct {
	rm           map[int]map[int]string
	totalReports int
}

func newReports() *reports {
	return &reports{rm: make(map[int]map[int]string)}
}

func (rs *reports) report(line, col int, message string) {
	lm := rs.rm[line]
	if lm == nil {
		rs.rm[line] = map[int]string{col: message}
	} else {
		lm[col] = message
	}
	rs.totalReports++
}

func (rs *reports) reportToken(t *lexer.Token, message string) {
	rs.report(t.Line(), t.Col(), message)
}

func (rs *reports) flatten() []ReportLine {
	res := make([]ReportLine, 0, rs.totalReports)
	lines := make([]int, 0, len(rs.rm))
	for line := range rs.rm {
		lines = append(lines, line)
	}
	sort.Ints(lines)

	for _, line := range lines {
		lm := rs.rm[line]
		cols := make([]int, 0, len(lm))
		for col := range lm {
			cols = append(cols, col)
		}
		sort.Ints(cols)

		for _, col := range cols {
			res = append(res, ReportLine{line, col, lm[col]})
		}
	}

	return res
}

func Check(s *source.Source) ([]ReportLine, error) {
	st, e := parseSource(s)
	if e == nil {
		return inspectCode(st).flatten(), nil
	} else {
		return nil, e
	}
}

func CheckFile(name string) ([]ReportLine, error) {
	file, e := os.Open(name)
	if e != nil {
		return nil, e
	}

	defer file.Close()
	stat, e := file.Stat()
	if e != nil {
		return nil, e
	}

	fsize := stat.Size()
	if fsize > (1 << 20) {
		return nil, errors.New("only accept files no longer than 1 MB")
	}

	content := make([]byte, fsize+1)
	bytes, e := file.Read(content)
	if bytes != int(fsize) {
		return nil, errors.New("error reading file")
	}
	content = content[:fsize]
	source.NormalizeNls(&content)
	return Check(source.New(name, content))
}

func parseSource(s *source.Source) (tree.Element, error) {
	q := source.NewQueue().Append(s)
	p, e := parser.New(cDataGrammar)
	if e != nil {
		return nil, e
	}
	hs := &parser.Hooks{
		Tokens: parser.TokenHooks{parser.AnyToken: handleToken},
		Nodes:  parser.NodeHooks{parser.AnyNode: tree.NodeHook},
	}
	res, e := p.Parse(q, hs)
	if e == nil {
		return res.(tree.Element), nil
	} else {
		return nil, e
	}
}

func handleToken(token *lexer.Token, pc *parser.ParseContext) (emit bool, e error) {
	tn := token.TypeName()
	if token.Line() == 1 && token.Col() == 1 && tn == spaceType {
		return false, pc.EmitToken(lexer.NewToken(0, indentType, append([]byte{'\n'}, token.Content()...), token.Pos()))
	}

	return (tn != commentType), nil
}

func inspectCode(st tree.Element) *reports {
	res := newReports()

	checks := []func(tree.Element, *reports){
		reportNoFinalNl,
		reportMultipleSpaces,
		reportIncorrectSpaces,
		reportInconsistentStructAlign,
		reportLiteralStructs,
		reportIncorrectIndents,
		reportTrailingSpaces,
		reportTabs,
	}

	for _, check := range checks {
		check(st, res)
	}

	return res
}

func reportNoFinalNl(st tree.Element, rs *reports) {
	lt := tree.LastTokenElement(st).Token()
	if lt.TypeName() != indentType {
		line := lt.Line()
		col := lt.Col() + len(lt.Text())
		rs.report(line, col, ErrNoEofNl)
	}
}

func reportTabs(st tree.Element, rs *reports) {
	hasTab := func(n tree.Element) bool {
		return strings.ContainsAny(n.Token().Text(), "\t")
	}
	sel := tree.NewSelector().
		Search(tree.IsA(indentType, spaceType)).
		Filter(hasTab)
	for _, n := range sel.Apply(st) {
		rs.reportToken(n.Token(), ErrTab)
	}
}

func reportTrailingSpaces(st tree.Element, rs *reports) {
	indentFollows := func(n tree.Element) bool {
		nn := n.Next()
		return (nn == nil || nn.TypeName() == indentType)
	}

	spaceSel := tree.NewSelector().
		Search(tree.IsA(spaceType)).
		Filter(indentFollows)

	indentSel := tree.NewSelector().
		Search(tree.IsA(indentType)).
		Filter(tree.IsNot(tree.IsALiteral("\n"))).
		Filter(indentFollows)

	for _, n := range spaceSel.Apply(st) {
		rs.reportToken(n.Token(), ErrTrailSpace)
	}
	for _, n := range indentSel.Apply(st) {
		rs.report(n.Token().Line()+1, 1, ErrTrailSpace)
	}
}

func isStructFieldAlign(n tree.Element) bool {
	parent := n.Parent()
	if parent == nil {
		return false
	}

	nodes := []tree.Element{
		n.Prev(),
		n.Next(),
		parent,
		parent.Parent(),
	}
	expectedTypes := []string{typeNt, nameNt, varDefNt, structTypeNt}

	for i, nn := range nodes {
		if nn == nil || !nn.IsNode() || nn.TypeName() != expectedTypes[i] {
			return false
		}
	}
	return true
}

func reportMultipleSpaces(st tree.Element, rs *reports) {
	sel := tree.NewSelector().
		Search(tree.IsA(spaceType)).
		Filter(tree.IsNot(
			tree.IsAny(
				tree.IsALiteral(" "),
				isStructFieldAlign)))

	for _, n := range sel.Apply(st) {
		rs.reportToken(n.Token(), ErrSpaces)
	}
}

func reportInconsistentStructAlign(st tree.Element, rs *reports) {
	structs := tree.NewSelector().DeepSearch(tree.IsA(structTypeNt)).Apply(st)

	selector := func(n tree.Element) []tree.Element {
		n = n.(tree.NodeElement).FirstChild()
		for n.TypeName() != nameNt {
			n = n.Next()
		}
		return []tree.Element{n}
	}

	nameSelector := tree.NewSelector().Search(tree.IsA(varDefNt)).Extract(selector)

	for _, s := range structs {
		names := nameSelector.Apply(s)
		if len(names) == 0 {
			continue
		}

		defaultCol := tree.FirstTokenElement(names[0]).Token().Col()
		for i := 1; i < len(names); i++ {
			col := tree.FirstTokenElement(names[i]).Token().Col()
			if col != defaultCol {
				rs.reportToken(tree.FirstTokenElement(names[i]).Token(), ErrAlign)
				break
			}
		}
	}
}

func reportIncorrectSpaces(st tree.Element, rs *reports) {
	messages := [2]string{ErrNoSpace, ErrWrongSpace}
	ops := tree.NewSelector().Search(tree.IsA(opType)).Apply(st)

	const (
		wrongTrail = 1 << iota
		wrongLead
		gotTrail
		gotLead
	)

	for _, n := range ops {
		prev := tree.PrevTokenElement(n)
		next := tree.NextTokenElement(n)
		flags := 0
		if prev != nil && prev.TypeName() == spaceType {
			flags |= (gotLead | wrongLead)
		}
		if next != nil && next.TypeName() == spaceType {
			flags |= (gotTrail | wrongTrail)
		}
		tok := n.Token()

		text := tok.Text()
		switch text {
		case "[", "]", ";":
		case ",", "}":
			flags ^= wrongTrail
		case "{":
			flags ^= wrongLead
		}

		if flags&(wrongLead|wrongTrail) != 0 {
			if (flags & wrongLead) != 0 {
				if flags&gotLead != 0 {
					tok = prev.Token()
				}
				rs.reportToken(tok, messages[(flags&gotLead)/gotLead]+" before "+text)
			}
			if (flags & wrongTrail) != 0 {
				if next != nil {
					tok = next.Token()
				}
				rs.reportToken(tok, messages[(flags&gotTrail)/gotTrail]+" after "+text)
			}
		}
	}
}

func reportLiteralStructs(st tree.Element, rs *reports) {
	literalStructs := tree.NewSelector().
		DeepSearch(tree.IsA(varDefNt)).
		Search(tree.IsA(structTypeNt)).
		Apply(st)

	for _, n := range literalStructs {
		rs.reportToken(tree.FirstTokenElement(n).Token(), ErrLiteral)
	}
}

func reportIncorrectIndents(st tree.Element, rs *reports) {
	indentItems := []string{varDefNt, typeDefNt}
	indentContainers := []string{rootNt, structTypeNt}

	blockIndentSize := func(n tree.Element) (size int, firstOnLine bool) {
		firstOnLine = true
		p := tree.PrevTokenElement(n)
		for p != nil && p.TypeName() != indentType {
			firstOnLine = false
			p = tree.PrevTokenElement(p)
		}

		if p == nil {
			return
		}

		for _, c := range p.Token().Text() {
			switch c {
			case '\n':
			case '\t':
				size = ((size + 7) & -8) + 1
			default:
				size++
			}
		}
		return
	}

	containers := tree.NewSelector().DeepSearch(tree.IsA(indentContainers...)).Apply(st)
	indentSelector := tree.NewSelector().Search(tree.IsA(indentItems...))
	for _, container := range containers {
		var expectedIndent int
		if container.TypeName() != rootNt {
			expectedIndent, _ = blockIndentSize(container)
			expectedIndent += 4
		}
		items := indentSelector.Apply(container)
		for _, item := range items {
			indent, isFirst := blockIndentSize(item)
			if !isFirst {
				rs.reportToken(tree.FirstTokenElement(item).Token(), ErrSameLine)
				continue
			}

			if indent != expectedIndent {
				rs.reportToken(tree.FirstTokenElement(item).Token(), ErrIndent)
				break
			}
		}
	}
}
