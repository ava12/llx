package grammar

const (
	RootNonTerm  = 0
)

type BitSet = int
type TermFlags int

type Term struct {
	Name   string
	Re     string
	Groups BitSet
	Flags  TermFlags
}

const (
	LiteralTerm TermFlags = 1 << iota
	ExternalTerm
	AsideTerm
	ErrorTerm
	ShrinkableTerm
)


type NonTerm struct {
	Name       string
	FirstState int
}

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


type Grammar struct {
	Terms    []Term
	NonTerms []NonTerm
	States   []State
}
