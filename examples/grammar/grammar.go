/*
Package grammar provides a generic parser to check correctness of grammar definitions.
The parser takes file names for a grammar definition and a source, tries to parse source file
and displays a parse tree or an error. Source file may contain multiple source samples.
Maximum length of grammar definition and source file is 1 MB.

Usage is

	grammar [-e] [{-m | -s <prefix>}] <grammar_file> <source_file>

Flag -e means that source file contains syntax errors. The program returns non-zero error code
if the source is parsed successfully.

Flag -m treats source file as multiple sources delimited by separators, the first line is the separator.
Each separator line starts with the same sequence of non-spacing characters.
The rest of the separator line (after one or more spacing chars) is ignored and may be used as a comment.

Argument -s <prefix> treats the source file as multiple sources if it starts with the given prefix.

NB: the last LF character preceding the separator is not included in the sample.
To add trailing LF to the sample add empty line.

The parser returns following error codes:

	1: wrong command line arguments
	2: error reading a file
	3: a file is not a valid UTF-8 encoded text
	4: invalid grammar definition
	5: error creating a parser (regexp error, layer initialization error)
	6: syntax error (or missing expected syntax error)

Error messages are printed to STDERR.
*/
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/parser"
	_ "github.com/ava12/llx/parser/layers"
	"github.com/ava12/llx/source"
	"github.com/ava12/llx/tree"
)

const (
	errUsage = iota + 1
	errFile
	errContent
	errGrammar
	errParser
	errSyntax
)

const maxFileSize = 1 << 20

const (
	maxLineLength  = 78
	maxTokenLength = 20
	indentSize     = 2
)

func main() {
	var (
		lineWidth       int
		multiSample     bool
		expectError     bool
		sampleSeparator string
	)

	var exitCode = 0

	flag.Usage = printHelp
	flag.BoolVar(&expectError, "e", false, "source file contains syntax errors")
	flag.BoolVar(&multiSample, "m", false, "source file contains multiple samples, first line is the separator")
	flag.StringVar(&sampleSeparator, "s", "", "treat source file as multiple samples if starts with this string")
	flag.IntVar(&lineWidth, "w", maxLineLength, "maximum output line width, runes")

	flag.Parse()
	if flag.NArg() < 2 {
		printHelp()
	}

	grammarSrc := makeSource(flag.Arg(0), loadFile(flag.Arg(0)))
	targetSrc := makeSources(flag.Arg(1), loadFile(flag.Arg(1)), multiSample, []byte(sampleSeparator))
	g := parseGrammar(grammarSrc)
	p := makeParser(g)
	for _, src := range targetSrc {
		fmt.Println(src.Name())
		t, e := parse(p, src)
		if e != nil {
			if expectError {
				fmt.Println("  *** error:", e.Error())
			} else {
				reportError(0, "  *** error: %s", e.Error())
				exitCode = errSyntax
			}
		} else {
			if expectError {
				reportError(0, "  *** expecting error, got success in %s", src.Name())
				exitCode = errSyntax
			} else {
				pr := newPrinter(indentSize, lineWidth).Indent()
				printTreeNode(t, pr)
				pr.Newline()
			}
		}
	}
	os.Exit(exitCode)
}

func printHelp() {
	fmt.Fprintln(os.Stderr, "Usage is  grammar [-e] [{-m | -s <prefix>}] <grammar_file> <source_file>")
	flag.PrintDefaults()
	os.Exit(errUsage)
}

func reportError(exitCode int, message string, args ...any) {
	if len(args) != 0 {
		message = fmt.Sprintf(message, args...)
	}
	fmt.Fprintln(os.Stderr, message)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func loadFile(name string) []byte {
	file, e := os.Open(name)
	if e != nil {
		reportError(errFile, e.Error())
	}

	defer file.Close()

	stat, e := file.Stat()
	if e != nil {
		reportError(errFile, e.Error())
	}

	size := stat.Size()
	if size > maxFileSize || size == 0 {
		reportError(errFile, "stat %s: invalid size (%d bytes)", name, size)
	}

	content, e := io.ReadAll(file)
	if e != nil {
		reportError(errFile, e.Error())
	}

	return content
}

func makeSource(name string, content []byte) *source.Source {
	if !utf8.Valid(content) {
		reportError(errContent, "check %s: not a valid UTF-8 encoded text", name)
	}

	source.NormalizeNls(&content)
	return source.New(name, content)
}

type lineEntry struct {
	firstPos, lastPos int
}

func makeSources(name string, content []byte, multiSample bool, separator []byte) []*source.Source {
	if len(separator) != 0 && bytes.HasPrefix(content, separator) {
		multiSample = true
	}

	if !multiSample {
		return []*source.Source{makeSource(name, content)}
	}

	source.NormalizeNls(&content)
	lines := contentLines(content)
	if len(lines) == 0 {
		return nil
	}

	var result []*source.Source

	separator = linePrefix(content[lines[0].firstPos:lines[0].lastPos])
	sampleIndex := 1
	lineIndex := 1
	for lineIndex < len(lines) {
		sample, lineCnt := sourceSample(content, lines[lineIndex:], separator)
		sourceName := fmt.Sprintf("### %s, sample #%d, (lines %d-%d)",
			name, sampleIndex, lineIndex+1, lineIndex+lineCnt)
		result = append(result, source.New(sourceName, sample))
		sampleIndex++
		lineIndex += lineCnt + 1
	}

	return result
}

func contentLines(content []byte) []lineEntry {
	var result []lineEntry
	pos := 0
	for pos < len(content) {
		newPos := bytes.IndexByte(content[pos:], '\n')
		if newPos <= 0 {
			result = append(result, lineEntry{pos, len(content)})
			break

		} else {
			result = append(result, lineEntry{pos, pos + newPos})
			pos += newPos + 1
		}
	}
	return result
}

func linePrefix(line []byte) []byte {
	for i, b := range line {
		if b <= ' ' {
			return line[:i]
		}
	}

	return line
}

func sourceSample(content []byte, lines []lineEntry, separator []byte) ([]byte, int) {
	if len(lines) == 0 {
		return nil, 0
	}

	for i, entry := range lines {
		if bytes.HasPrefix(content[entry.firstPos:entry.lastPos], separator) {
			return content[lines[0].firstPos:entry.firstPos], i
		}
	}

	return content[lines[0].firstPos:], len(lines)
}

func parseGrammar(s *source.Source) *grammar.Grammar {
	g, e := langdef.Parse(s)
	if e != nil {
		reportError(errGrammar, e.Error())
	}

	return g
}

func makeParser(g *grammar.Grammar) *parser.Parser {
	p, e := parser.New(g)
	if e != nil {
		reportError(errParser, e.Error())
	}

	return p
}

func parse(p *parser.Parser, src *source.Source) (tree.NodeElement, error) {
	q := source.NewQueue().Append(src)
	t, e := p.Parse(context.Background(), q, parser.Hooks{
		Nodes: parser.NodeHooks{
			parser.AnyNode: tree.NodeHook,
		},
	}, parser.WithFullSource())
	if e != nil {
		return nil, e
	}

	return t.(tree.NodeElement), nil
}

func printTreeNode(tn tree.NodeElement, p *printer) {
	label := tn.TypeName()
	children := tree.Children(tn)
	for len(children) == 1 && children[0].IsNode() {
		tn = children[0].(tree.NodeElement)
		children = tree.Children(tn)
		label = label + ":" + tn.TypeName()
	}
	p.Print(label).Print("{").Newline().Indent()

	for _, child := range children {
		if child.IsNode() {
			printTreeNode(child.(tree.NodeElement), p)
		} else {
			printTreeToken(child, p)
		}
	}

	p.Newline().Dedent().Print("}")
}

func printTreeToken(te tree.Element, p *printer) {
	content := te.Token().Content()
	tt := te.TypeName()
	if utf8.RuneCount(content) <= maxLineLength {
		p.Print(fmt.Sprintf("%s(%q)", tt, string(content)))
	} else {
		tailPos := 0
		for i := maxLineLength - 3; i > 0; i-- {
			_, pos := utf8.DecodeRune(content[tailPos:])
			tailPos += pos
		}
		p.Print(fmt.Sprintf("%s(%q...)", tt, string(content[:tailPos])))
	}
}

type printer struct {
	indentSize, maxCol       int
	indentLevel, col         int
	indent, indentTpl, space string
	printed                  bool
}

func newPrinter(indentSize, maxLineLength int) *printer {
	return &printer{
		indentSize: indentSize,
		maxCol:     maxLineLength - 1,
		indentTpl:  "        ",
	}
}

func (p *printer) Print(s string) *printer {
	strlen := utf8.RuneCountInString(s)
	if strlen+p.col+1 > p.maxCol {
		p.Newline()
	}
	fmt.Printf("%s%s", p.space, s)
	p.col += len(p.space) + strlen
	p.space = " "
	p.printed = true
	return p
}

func (p *printer) Newline() *printer {
	if !p.printed {
		return p
	}

	fmt.Println()
	p.space = p.indent
	p.printed = false
	p.col = len(p.space)
	return p
}

func (p *printer) Indent() *printer {
	p.indentLevel++
	size := p.indentLevel * p.indentSize
	for len(p.indentTpl) < size {
		p.indentTpl = p.indentTpl + p.indentTpl
	}
	p.indent = p.indentTpl[0:size]
	if !p.printed {
		p.space = p.indent
	}
	return p
}

func (p *printer) Dedent() *printer {
	p.indentLevel--
	p.indent = p.indentTpl[0 : p.indentLevel*p.indentSize]
	if !p.printed {
		p.space = p.indent
	}
	return p
}
