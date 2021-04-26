//go:generate llxgen grammar.llx
package lib

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/source"
)

type function struct {
	argNames []string
	body expr
}

func newFunc (body expr, argNames ... string) *function {
	return &function{argNames, body}
}

func (f *function) call (c *context, args []float64) (float64, error) {
	argc := len(args)
	if argc != len(f.argNames) {
		return 0.0, fmt.Errorf("wrong number of arguments: expecting %d, got %d", len(f.argNames), argc)
	}

	newc := newContext(c)
	for i, name := range f.argNames {
		newc.vars[name] = args[i]
	}
	return f.body.Compute(newc)
}

type context struct {
	parent *context
	vars map[string]float64
	functions map[string]*function
}

func newContext (parent *context) *context {
	return &context{parent, make(map[string]float64), make(map[string]*function)}
}

func (c *context) variable (name string) (res float64, e error) {
	var f bool
	res, f = c.vars[name]
	if !f {
		if c.parent != nil {
			res, e = c.parent.variable(name)
		} else {
			e = errors.New("unknown variable: " + name)
		}
	}
	return
}

func (c *context) function (name string) (res *function, e error) {
	var f bool
	res, f = c.functions[name]
	if !f {
		if c.parent != nil {
			res, e = c.parent.function(name)
		} else {
			e = errors.New("unknown function: " + name)
		}
	}
	return
}


type expr interface {
	IsNumber () bool
	Compute (*context) (float64, error)
}

type number struct {
	value float64
}

func newNumber (value float64) *number {
	return &number{value}
}

func (n number) IsNumber () bool {
	return true
}

func (n number) Compute (*context) (float64, error) {
	return n.value, nil
}


type opVal struct {
	op rune
	value expr
}

type chain struct {
	defaultOp, lastOp rune
	value float64
	opVals []opVal
}

func newChain (defaultOp rune) *chain {
	res := &chain{defaultOp, defaultOp, 0.0, make([]opVal, 0)}
	if defaultOp == '*' {
		res.value = 1.0
	}
	return res
}

func (ch *chain) IsNumber () bool {
	return (len(ch.opVals) == 0)
}

func (ch *chain) Compute (c *context) (res float64, e error) {
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

func (ch *chain) update (res, val float64, op rune) float64 {
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

func (ch *chain) HandleNonTerm (nonTerm string, result interface{}) error {
	exp := result.(expr)
	if exp.IsNumber() {
		val, _ := exp.Compute(nil)
		ch.value = ch.update(ch.value, val, ch.lastOp)
	} else {
		ch.opVals = append(ch.opVals, opVal{ch.lastOp, exp})
	}
	return nil
}

func (ch *chain) HandleToken (token *lexer.Token) error {
	ch.lastOp = rune(token.Text()[0])
	return nil
}

func (ch *chain) EndNonTerm () (result interface{}, e error) {
	if ch.IsNumber() {
		return newNumber(ch.value), nil
	} else {
		return ch, nil
	}
}


type power struct {
	base, exp expr
}

func newPower () *power {
	return &power{exp: newNumber(1.0)}
}

func (p *power) IsNumber () bool {
	return (p.base.IsNumber() && p.exp.IsNumber())
}

func (p *power) Compute (c *context) (res float64, e error) {
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

func (p *power) HandleNonTerm (nonTerm string, result interface{}) error {
	x := result.(expr)
	if p.base == nil {
		p.base = x
	} else {
		p.exp = x
	}
	return nil
}

func (p *power) HandleToken (token *lexer.Token) error {
	return nil
}

func (p *power) EndNonTerm () (result interface{}, e error) {
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

func newVarName (name string) *varName {
	return &varName{name}
}

func (v *varName) IsNumber () bool {
	return false
}

func (v *varName) Compute (c *context) (float64, error) {
	return c.variable(v.name)
}


type assignment struct {
	name string
	value expr
}

func newAssignment () *assignment {
	return &assignment{}
}

func (a *assignment) IsNumber () bool {
	return false
}

func (a *assignment) Compute (c *context) (res float64, e error) {
	res, e = a.value.Compute(c)
	if e == nil {
		c.vars[a.name] = res
	}
	return
}

func (a *assignment) HandleNonTerm (nonTerm string, result interface{}) error {
	if nonTerm == "expr" {
		a.value = result.(expr)
	}
	return nil
}

func (a *assignment) HandleToken (token *lexer.Token) error {
	if token.TypeName() == "name" {
		a.name = token.Text()
	}
	return nil
}

func (a *assignment) EndNonTerm () (result interface{}, e error) {
	return a, nil
}


type funcDef struct {
	name string
	argNames []string
	body expr
	nameIndex map[string]bool
}

func newFuncDef () *funcDef {
	return &funcDef{argNames: make([]string, 0), nameIndex: make(map[string]bool)}
}

func (fd *funcDef) IsNumber () bool {
	return false
}

func (fd *funcDef) Compute (c *context) (res float64, e error) {
	c.functions[fd.name] = newFunc(fd.body, fd.argNames...)
	return 0.0, nil
}

func (fd *funcDef) HandleNonTerm (nonTerm string, result interface{}) error {
	if nonTerm == "expr" {
		fd.body = result.(expr)
	}
	return nil
}

func (fd *funcDef) HandleToken (token *lexer.Token) (e error) {
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
		return fmt.Errorf("argument %q already defined", name)
	}

	fd.argNames = append(fd.argNames, token.Text())
	fd.nameIndex[name] = true
	return
}

func (fd *funcDef) EndNonTerm () (result interface{}, e error) {
	return fd, nil
}


type funcCall struct {
	name string
	args []expr
}

func newFuncCall () *funcCall {
	return &funcCall{}
}

func (fc *funcCall) IsNumber () bool {
	return false
}

func (fc *funcCall) Compute (c *context) (float64, error) {
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

func (fc *funcCall) HandleNonTerm (nonTerm string, result interface{}) error {
	if nonTerm == "expr" {
		fc.args = append(fc.args, result.(expr))
	}
	return nil
}

func (fc *funcCall) HandleToken (token *lexer.Token) error {
	if token.TypeName() == "name" {
		fc.name = token.Text()
	}
	return nil
}

func (fc *funcCall) EndNonTerm () (result interface{}, e error) {
	return fc, nil
}


type rootNT struct {
	body expr
}

func newRootNT () *rootNT {
	return &rootNT{}
}

func (r *rootNT) HandleNonTerm (nonTerm string, result interface{}) error {
	r.body = result.(expr)
	return nil
}

func (r *rootNT) HandleToken (token *lexer.Token) error {
	return nil
}

func (r *rootNT) EndNonTerm () (result interface{}, e error) {
	return r.body, nil
}


type value struct {
	body expr
}

func newValue () *value {
	return &value{}
}

func (v *value) HandleNonTerm (nonTerm string, result interface{}) error {
	v.body = result.(expr)
	return nil
}

func (v *value) HandleToken (token *lexer.Token) error {
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

func (v *value) EndNonTerm () (result interface{}, e error) {
	return v.body, nil
}


var hooks = &parser.Hooks{
	NonTerms: parser.NonTermHooks{
		parser.AnyNonTerm: func (nonTerm string, pc *parser.ParseContext) (res parser.NonTermHookInstance, e error) {
			switch nonTerm {
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
	calcParser *parser.Parser
)

func init () {
	rootContext = newContext(nil)
	calcParser = parser.New(calcGrammar)
}

func Compute (text string) (float64, error) {
	q := source.NewQueue().Append(source.New("input", []byte(text)))
	x, e := calcParser.Parse(q, hooks)
	if e == nil {
		return x.(expr).Compute(rootContext)
	} else {
		return 0.0, e
	}
}
