package grammar

const (
	RootNonTerm  = 0
	InitialState = 0
)


type TermFlags int

type Term struct {
	Name   string
	Re     string
	Groups int
	Flags  TermFlags
}

const (
	LiteralTerm TermFlags = 1 << iota
	ExternalTerm
	AsideTerm
	ErrorTerm
	ShrinkableTerm
)


type Rule struct {
	State   int
	NonTerm int
}

const (
	SameNonTerm = -1
	FinalState  = -1
)


type State struct {
	Group      int            `json:",omitempty"`
	Rules      map[int]Rule   `json:",omitempty"`
	MultiRules map[int][]Rule `json:",omitempty"`
}

const AnyTerm = -1


type NonTerm struct {
	Name   string
	States []State
}

type Grammar struct {
	Terms    []Term
	NonTerms []NonTerm
}
