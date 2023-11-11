//go:generate ../../../bin/llxgen grammar.llx
package internal

import (
	"math"
	"strconv"

	"github.com/ava12/llx"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/source"
)

const (
	UnknownVarError = iota + 301
	UnknownFuncError
	WrongArgNumberError
	ArgDefinedError
	UnexpectedInputError
)

func unknownVarError(name string) *llx.Error {
	return llx.FormatError(UnknownVarError, "unknown variable: %s", name)
}

func unknownFuncError(name string) *llx.Error {
	return llx.FormatError(UnknownFuncError, "unknown function: %s", name)
}

func wrongArgNumberError(name string, expected, got int) *llx.Error {
	return llx.FormatError(WrongArgNumberError, "wrong number of arguments for %s: expecting %d, got %d", name, expected, got)
}

func argDefinedError(name string) *llx.Error {
	return llx.FormatError(ArgDefinedError, "argument %s already defined", name)
}

func unexpectedInputError(pos llx.SourcePos, text string) *llx.Error {
	return llx.FormatErrorPos(pos, UnexpectedInputError, "unexpected %q", text)
}

type function struct {
	name     string
	argNames []string
	body     expr
}

func newFunc(name string, body expr, argNames ...string) *function {
	return &function{name, argNames, body}
}

func (f *function) call(c *context, args []float64) (float64, error) {
	argc := len(args)
	if argc != len(f.argNames) {
		return 0.0, wrongArgNumberError(f.name, len(f.argNames), argc)
	}

	newc := newContext(c)
	for i, name := range f.argNames {
		newc.vars[name] = args[i]
	}
	return f.body.Compute(newc)
}

type context struct {
	parent    *context
	vars      map[string]float64
	functions map[string]*function
}

func newContext(parent *context) *context {
	return &context{parent, make(map[string]float64), make(map[string]*function)}
}

func (c *context) variable(name string) (res float64, e error) {
	var f bool
	res, f = c.vars[name]
	if !f {
		if c.parent != nil {
			res, e = c.parent.variable(name)
		} else {
			e = unknownVarError(name)
		}
	}
	return
}

func (c *context) function(name string) (res *function, e error) {
	var f bool
	res, f = c.functions[name]
	if !f {
		if c.parent != nil {
			res, e = c.parent.function(name)
		} else {
			e = unknownFuncError(name)
		}
	}
	return
}

type expr interface {
	IsNumber() bool
	Compute(*context) (float64, error)
}

type number struct {
	value float64
}

func newNumber(value float64) *number {
	return &number{value}
}

func (n number) IsNumber() bool {
	return true
}

func (n number) Compute(*context) (float64, error) {
	return n.value, nil
}

type opVal struct {
	op    rune
	value expr
}

type chain struct {
	defaultOp, lastOp rune
	value             float64
	opVals            []opVal
}

func newChain(defaultOp rune) *chain {
	res := &chain{defaultOp, defaultOp, 0.0, make([]opVal, 0)}
	if defaultOp == '*' {
		res.value = 1.0
	}
	return res
}

func (ch *chain) IsNumber() bool {
	return (len(ch.opVals) == 0)
}

func (ch *chain) Compute(c *context) (res float64, e error) {
	res = ch.value
	var val float64
	for _, ov := range ch.opVals {
		val, e = ov.value.Compute(c)
		if e != nil {
			return 0.0, e
		}

		res = ch.update(res, val, ov.op)
	}
	return
}

func (ch *chain) update(res, val float64, op rune) float64 {
	switch op {
	case '+':
		res += val
	case '-':
		res -= val
	case '*':
		res *= val
	case '/':
		res /= val
	}
	return res
}

func (ch *chain) NewNode(node string, token *parser.Token) error {
	return nil
}

func (ch *chain) HandleNode(node string, result interface{}) error {
	exp := result.(expr)
	if exp.IsNumber() {
		val, _ := exp.Compute(nil)
		ch.value = ch.update(ch.value, val, ch.lastOp)
	} else {
		ch.opVals = append(ch.opVals, opVal{ch.lastOp, exp})
	}
	return nil
}

func (ch *chain) HandleToken(token *parser.Token) error {
	ch.lastOp = rune(token.Text()[0])
	return nil
}

func (ch *chain) EndNode() (result interface{}, e error) {
	if ch.IsNumber() {
		return newNumber(ch.value), nil
	} else {
		return ch, nil
	}
}

type power struct {
	base, exp expr
}

func newPower() *power {
	return &power{exp: newNumber(1.0)}
}

func (p *power) IsNumber() bool {
	return (p.base.IsNumber() && p.exp.IsNumber())
}

func (p *power) Compute(c *context) (res float64, e error) {
	var exp float64
	res, e = p.base.Compute(c)
	if e == nil {
		exp, e = p.exp.Compute(c)
		if e == nil {
			res = math.Pow(res, exp)
		}
	}
	return
}

func (p *power) NewNode(node string, token *parser.Token) error {
	return nil
}

func (p *power) HandleNode(node string, result interface{}) error {
	x := result.(expr)
	if p.base == nil {
		p.base = x
	} else {
		p.exp = x
	}
	return nil
}

func (p *power) HandleToken(token *parser.Token) error {
	return nil
}

func (p *power) EndNode() (result interface{}, e error) {
	if p.IsNumber() {
		var x float64
		x, e = p.Compute(nil)
		if e == nil {
			result = newNumber(x)
		}
	} else {
		result = p
	}
	return
}

type varName struct {
	name string
}

func newVarName(name string) *varName {
	return &varName{name}
}

func (v *varName) IsNumber() bool {
	return false
}

func (v *varName) Compute(c *context) (float64, error) {
	return c.variable(v.name)
}

type assignment struct {
	name  string
	value expr
}

func newAssignment() *assignment {
	return &assignment{}
}

func (a *assignment) IsNumber() bool {
	return false
}

func (a *assignment) Compute(c *context) (res float64, e error) {
	res, e = a.value.Compute(c)
	if e == nil {
		c.vars[a.name] = res
	}
	return
}

func (a *assignment) NewNode(node string, token *parser.Token) error {
	return nil
}

func (a *assignment) HandleNode(node string, result interface{}) error {
	if node == "expr" {
		a.value = result.(expr)
	}
	return nil
}

func (a *assignment) HandleToken(token *parser.Token) error {
	if token.TypeName() == "name" {
		a.name = token.Text()
	}
	return nil
}

func (a *assignment) EndNode() (result interface{}, e error) {
	return a, nil
}

type funcDef struct {
	name      string
	argNames  []string
	body      expr
	nameIndex map[string]bool
}

func newFuncDef() *funcDef {
	return &funcDef{argNames: make([]string, 0), nameIndex: make(map[string]bool)}
}

func (fd *funcDef) IsNumber() bool {
	return false
}

func (fd *funcDef) Compute(c *context) (res float64, e error) {
	c.functions[fd.name] = newFunc(fd.name, fd.body, fd.argNames...)
	return 0.0, nil
}

func (fd *funcDef) NewNode(node string, token *parser.Token) error {
	return nil
}

func (fd *funcDef) HandleNode(node string, result interface{}) error {
	if node == "expr" {
		fd.body = result.(expr)
	}
	return nil
}

func (fd *funcDef) HandleToken(token *parser.Token) (e error) {
	if token.TypeName() != "name" {
		return
	}

	name := token.Text()
	if name == "func" {
		return
	}

	if fd.name == "" {
		fd.name = name
		return
	}

	if fd.nameIndex[name] {
		return argDefinedError(name)
	}

	fd.argNames = append(fd.argNames, token.Text())
	fd.nameIndex[name] = true
	return
}

func (fd *funcDef) EndNode() (result interface{}, e error) {
	return fd, nil
}

type funcCall struct {
	name string
	args []expr
}

func newFuncCall() *funcCall {
	return &funcCall{}
}

func (fc *funcCall) IsNumber() bool {
	return false
}

func (fc *funcCall) Compute(c *context) (float64, error) {
	f, e := c.function(fc.name)
	args := make([]float64, len(fc.args))
	var res float64
	if e == nil {
		for i, expr := range fc.args {
			res, e = expr.Compute(c)
			if e == nil {
				args[i] = res
			} else {
				break
			}
		}
	}
	if e == nil {
		res, e = f.call(c, args)
	}
	return res, e
}

func (fc *funcCall) NewNode(node string, token *parser.Token) error {
	return nil
}

func (fc *funcCall) HandleNode(node string, result interface{}) error {
	if node == "expr" {
		fc.args = append(fc.args, result.(expr))
	}
	return nil
}

func (fc *funcCall) HandleToken(token *parser.Token) error {
	if token.TypeName() == "name" {
		fc.name = token.Text()
	}
	return nil
}

func (fc *funcCall) EndNode() (result interface{}, e error) {
	return fc, nil
}

type rootNT struct {
	body expr
}

func newRootNT() *rootNT {
	return &rootNT{}
}

func (r *rootNT) NewNode(node string, token *parser.Token) error {
	return nil
}

func (r *rootNT) HandleNode(node string, result interface{}) error {
	r.body = result.(expr)
	return nil
}

func (r *rootNT) HandleToken(token *parser.Token) error {
	return nil
}

func (r *rootNT) EndNode() (result interface{}, e error) {
	return r.body, nil
}

type value struct {
	body expr
}

func newValue() *value {
	return &value{}
}

func (v *value) NewNode(node string, token *parser.Token) error {
	return nil
}

func (v *value) HandleNode(node string, result interface{}) error {
	v.body = result.(expr)
	return nil
}

func (v *value) HandleToken(token *parser.Token) error {
	switch token.TypeName() {
	case "name":
		v.body = newVarName(token.Text())
	case "number":
		res, e := strconv.ParseFloat(token.Text(), 64)
		if e == nil {
			v.body = newNumber(res)
		} else {
			return e
		}
	}
	return nil
}

func (v *value) EndNode() (result interface{}, e error) {
	return v.body, nil
}

var hooks = &parser.Hooks{
	Nodes: parser.NodeHooks{
		parser.AnyNode: func(node string, t *parser.Token, pc *parser.ParseContext) (res parser.NodeHookInstance, e error) {
			switch node {
			case "calcGrammar":
				res = newRootNT()
			case "func":
				res = newFuncDef()
			case "assign":
				res = newAssignment()
			case "expr":
				res = newChain('+')
			case "pro":
				res = newChain('*')
			case "pow":
				res = newPower()
			case "value":
				res = newValue()
			case "call":
				res = newFuncCall()
			}
			return
		},
	},
}

var (
	rootContext *context
	calcParser  *parser.Parser
)

func init() {
	rootContext = newContext(nil)
	calcParser = parser.New(calcGrammar)
}

func Compute(text string) (float64, error) {
	input := []byte(text)
	source.NormalizeNls(&input)
	q := source.NewQueue().Append(source.New("input", input))
	x, e := calcParser.Parse(q, hooks)
	if e == nil && !q.IsEmpty() {
		p := q.SourcePos()
		e = unexpectedInputError(p, string(p.Source().Content()[p.Pos():]))
	}

	if e == nil {
		return x.(expr).Compute(rootContext)
	} else {
		return 0.0, e
	}
}
