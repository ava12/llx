// Code generated with llxgen.
package lib

import "github.com/ava12/llx/grammar"

var calcGrammar = &grammar.Grammar{
	Tokens: []grammar.Token{
		{Name: "space", Re: "\\s+", Groups: 1, Flags: 4},
		{Name: "number", Re: "-?\\d+(?:\\.\\d+)?(?:[Ee]-?\\d+)?", Groups: 1, Flags: 0},
		{Name: "name", Re: "[A-Za-z][A-Za-z0-9_]*", Groups: 1, Flags: 0},
		{Name: "op", Re: "[(),=*\\/^+-]", Groups: 1, Flags: 0},
		{Name: "func", Re: "", Groups: 1, Flags: 1},
		{Name: "(", Re: "", Groups: 1, Flags: 1},
		{Name: ",", Re: "", Groups: 1, Flags: 1},
		{Name: ")", Re: "", Groups: 1, Flags: 1},
		{Name: "=", Re: "", Groups: 1, Flags: 1},
		{Name: "-", Re: "", Groups: 1, Flags: 1},
		{Name: "+", Re: "", Groups: 1, Flags: 1},
		{Name: "*", Re: "", Groups: 1, Flags: 1},
		{Name: "/", Re: "", Groups: 1, Flags: 1},
		{Name: "^", Re: "", Groups: 1, Flags: 1},
	},
	NonTerms: []grammar.NonTerm{
		{Name: "calcGrammar", FirstState: 0},
		{Name: "expr", FirstState: 1},
		{Name: "assign", FirstState: 5},
		{Name: "func", FirstState: 8},
		{Name: "pro", FirstState: 16},
		{Name: "pow", FirstState: 19},
		{Name: "value", FirstState: 22},
		{Name: "call", FirstState: 25},
	},
	States: []grammar.State{
		{Group: 0, Rules: map[int]grammar.Rule{
			9: {State: -1, NonTerm: 1},
			1: {State: -1, NonTerm: 1},
			4: {State: -1, NonTerm: 3},
			5: {State: -1, NonTerm: 1},
		}, MultiRules: map[int][]grammar.Rule{
			2: {
				{State: -1, NonTerm: 1},
				{State: -1, NonTerm: 2},
			},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			-1: {State: 2, NonTerm: -1},
			9: {State: 2, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: 3, NonTerm: 4},
			2: {State: 3, NonTerm: 4},
			5: {State: 3, NonTerm: 4},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			9: {State: 4, NonTerm: -1},
			-1: {State: -1, NonTerm: -1},
			10: {State: 4, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: 3, NonTerm: 4},
			2: {State: 3, NonTerm: 4},
			5: {State: 3, NonTerm: 4},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			2: {State: 6, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			8: {State: 7, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			5: {State: -1, NonTerm: 1},
			9: {State: -1, NonTerm: 1},
			1: {State: -1, NonTerm: 1},
			2: {State: -1, NonTerm: 1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			4: {State: 9, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			2: {State: 10, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			5: {State: 11, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			-1: {State: 12, NonTerm: -1},
			2: {State: 14, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			7: {State: 13, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: -1, NonTerm: 1},
			2: {State: -1, NonTerm: 1},
			5: {State: -1, NonTerm: 1},
			9: {State: -1, NonTerm: 1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			-1: {State: 12, NonTerm: -1},
			6: {State: 15, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			2: {State: 14, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: 17, NonTerm: 5},
			2: {State: 17, NonTerm: 5},
			5: {State: 17, NonTerm: 5},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			-1: {State: -1, NonTerm: -1},
			11: {State: 18, NonTerm: -1},
			12: {State: 18, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: 17, NonTerm: 5},
			2: {State: 17, NonTerm: 5},
			5: {State: 17, NonTerm: 5},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			5: {State: 20, NonTerm: 6},
			1: {State: 20, NonTerm: 6},
			2: {State: 20, NonTerm: 6},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			-1: {State: -1, NonTerm: -1},
			13: {State: 21, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: -1, NonTerm: 5},
			2: {State: -1, NonTerm: 5},
			5: {State: -1, NonTerm: 5},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: -1, NonTerm: -1},
			5: {State: 23, NonTerm: -1},
		}, MultiRules: map[int][]grammar.Rule{
			2: {
				{State: -1, NonTerm: -1},
				{State: -1, NonTerm: 7},
			},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: 24, NonTerm: 1},
			2: {State: 24, NonTerm: 1},
			5: {State: 24, NonTerm: 1},
			9: {State: 24, NonTerm: 1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			7: {State: -1, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			2: {State: 26, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			5: {State: 27, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			-1: {State: 28, NonTerm: -1},
			1: {State: 29, NonTerm: 1},
			2: {State: 29, NonTerm: 1},
			5: {State: 29, NonTerm: 1},
			9: {State: 29, NonTerm: 1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			7: {State: -1, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			-1: {State: 28, NonTerm: -1},
			6: {State: 30, NonTerm: -1},
		}},
		{Group: 0, Rules: map[int]grammar.Rule{
			1: {State: 29, NonTerm: 1},
			2: {State: 29, NonTerm: 1},
			5: {State: 29, NonTerm: 1},
			9: {State: 29, NonTerm: 1},
		}},
	},
}
