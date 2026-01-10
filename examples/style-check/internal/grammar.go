// Code generated with llxgen.

package internal

import "github.com/ava12/llx/grammar"

var cDataGrammar = &grammar.Grammar{
	Tokens: []grammar.Token{
		{Name: "indent", Re: "\\n[ \\t]*", Group: 0, Flags: 4},
		{Name: "space", Re: "[ \\t]+", Group: 0, Flags: 4},
		{Name: "comment", Re: "\\/\\*.*?\\*\\/", Group: 0, Flags: 4},
		{Name: "number", Re: "\\d+", Group: 0, Flags: 0},
		{Name: "name", Re: "[A-Za-z_][A-Za-z_0-9]*", Group: 0, Flags: 0},
		{Name: "op", Re: "[{}\\[\\],;]", Group: 0, Flags: 0},
		{Name: ",", Re: "", Group: 0, Flags: 1},
		{Name: ";", Re: "", Group: 0, Flags: 1},
		{Name: "typedef", Re: "", Group: 0, Flags: 1},
		{Name: "struct", Re: "", Group: 0, Flags: 1},
		{Name: "{", Re: "", Group: 0, Flags: 1},
		{Name: "}", Re: "", Group: 0, Flags: 1},
		{Name: "[", Re: "", Group: 0, Flags: 1},
		{Name: "]", Re: "", Group: 0, Flags: 1},
	},
	Nodes: []grammar.Node{
		{Name: "cDataGrammar", FirstState: 0},
		{Name: "var-def", FirstState: 2},
		{Name: "type-def", FirstState: 8},
		{Name: "type", FirstState: 12},
		{Name: "name", FirstState: 13},
		{Name: "size-def", FirstState: 14},
		{Name: "simple-type", FirstState: 17},
		{Name: "struct-type", FirstState: 18},
	},
	States: []grammar.State{
		{23, 0, 0, 0, 3}, // cDataGrammar(0)
		{23, 0, 0, 3, 7},
		{23, 0, 0, 7, 9}, // var-def(2)
		{23, 0, 0, 9, 10},
		{39, 0, 0, 10, 13},
		{39, 0, 0, 13, 15},
		{23, 0, 0, 15, 16},
		{39, 0, 0, 16, 18},
		{23, 0, 0, 18, 19}, // type-def(8)
		{23, 0, 0, 19, 21},
		{23, 0, 0, 21, 22},
		{39, 0, 0, 22, 24},
		{23, 0, 0, 24, 26}, // type(12)
		{23, 0, 0, 26, 27}, // name(13)
		{39, 0, 0, 27, 28}, // size-def(14)
		{15, 0, 0, 28, 29},
		{39, 0, 0, 29, 30},
		{23, 0, 0, 30, 31}, // simple-type(17)
		{23, 0, 0, 31, 32}, // struct-type(18)
		{39, 0, 0, 32, 33},
		{23, 0, 0, 33, 35},
		{55, 0, 0, 35, 38},
	},
	MultiRules: []grammar.MultiRule{},
	Rules: []grammar.Rule{
		{4, 1, 1}, // 0(0)
		{8, 1, 2},
		{9, 1, 1},
		{-1, -1, -1}, // 1(3)
		{4, 1, 1},
		{8, 1, 2},
		{9, 1, 1},
		{4, 3, 3}, // 2(7)
		{9, 3, 3},
		{4, 4, 4}, // 3(9)
		{6, 6, -1}, // 4(10)
		{7, -1, -1},
		{12, 4, 5},
		{6, 6, -1}, // 5(13)
		{7, -1, -1},
		{4, 7, 4}, // 6(15)
		{-1, 5, -1}, // 7(16)
		{12, 7, 5},
		{8, 9, -1}, // 8(18)
		{4, 10, 3}, // 9(19)
		{9, 10, 3},
		{4, 11, 4}, // 10(21)
		{7, -1, -1}, // 11(22)
		{12, 11, 5},
		{4, -1, 6}, // 12(24)
		{9, -1, 7},
		{4, -1, -1}, // 13(26)
		{12, 15, -1}, // 14(27)
		{3, 16, -1}, // 15(28)
		{13, -1, -1}, // 16(29)
		{4, -1, -1}, // 17(30)
		{9, 19, -1}, // 18(31)
		{10, 20, -1}, // 19(32)
		{4, 21, 1}, // 20(33)
		{9, 21, 1},
		{4, 21, 1}, // 21(35)
		{9, 21, 1},
		{11, -1, -1},
	},
}
