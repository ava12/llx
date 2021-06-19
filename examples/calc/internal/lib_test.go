package internal

import (
	"fmt"
	"math"
	"testing"

	"github.com/ava12/llx"
	"github.com/ava12/llx/parser"
)

const eps = 1e-9

func sign (f float64) int {
	if math.Signbit(f) {
		return -1
	} else {
		return 1
	}
}

func equal (a, b float64) bool {
	if math.IsNaN(a) {
		return math.IsNaN(b)
	}

	s := sign(a)
	if math.IsInf(a, s) {
		return math.IsInf(b, s)
	}

	return (math.Abs(a - b) < eps)
}

type sample struct {
	input  string
	result float64
	err    int
}

func testSamples (t *testing.T, samples []sample) {
	var i int
	var s sample

	report := func (msg string, params ... interface{}) {
		prefix := fmt.Sprintf("sample #%d (%q): ", i, s.input)
		t.Fatalf(prefix + msg, params...)
	}

	for i, s = range samples {
		r, e := Compute(s.input)
		if e == nil {
			if s.err != 0 {
				report("expecting error code %d, got success", s.err)
			}

			if !equal(r, s.result) {
				report("expecting %f, got %f", s.result, r)
			}
		} else {
			ee, f := e.(*llx.Error)
			if !f {
				report("unexpected error: %s", e)
			}

			if s.err == 0 || ee.Code != s.err {
				report("unexpected error code %d (%s)", ee.Code, ee)
			}
		}
	}
}

func TestSyntaxErrors (t *testing.T) {
	samples := []sample {
		{" ", 0, parser.UnexpectedEofError},
		{"2 + ", 0, parser.UnexpectedEofError},
		{"(3 * 4", 0, parser.UnexpectedEofError},
		{"2 * -x", 0, parser.UnexpectedTokenError},
	}

	testSamples(t, samples)
}

func TestCalc (t *testing.T) {
	samples := []sample{
		{"2", 2, 0},
		{"2 + 2 ^ 3 * 2", 18, 0},
		{"2 / 6", 1.0/3, 0},
		{"1/0", math.Inf(1), 0},
		{"-10/0", math.Inf(-1), 0},
		{"-2^0.5", math.NaN(), 0},
		{"(2+2)*2^-2", 1, 0},
		{"2^2^3", 256, 0},
		{"x", 0, UnknownVarError},
		{"x = 2 + y", 0, UnknownVarError},
		{"x", 0, UnknownVarError},
		{"x = 3", 3, 0},
		{"x", 3, 0},
		{"x(4)", 0, UnknownFuncError},
		{"func x (a) a + 1", 0, 0},
		{"x(4)", 5, 0},
		{"-x + (-x) / x(-x)", -1.5, 0},
		{"x = 7", 7, 0},
		{"x", 7, 0},
		{"func x (a, b) a * b", 0, 0},
		{"x = x(x, 3)", 21, 0},
		{"func y(a,b,a) a + b", 0, ArgDefinedError},
		{"y()", 0, UnknownFuncError},
		{"func y (x, y) x + y", 0, 0},
		{"y(11, 22)", 33, 0},
		{"2 + 3 4", 0, UnexpectedInputError},
	}

	rootContext = newContext(nil)
	testSamples(t, samples)
}
