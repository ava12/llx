package grammar

const (
	RootNonterm = 0
	InitialState = 0
)


type TermFlags int

type Term struct {
	Name, Re string
	Groups int
	Flags TermFlags
}

const (
	LiteralTerm TermFlags = 1 << iota
	ExternalTerm
	AsideTerm
	ErrorTerm
	ShrinkableTerm
)


type Rule struct {
	State, Nonterm int
}

const (
	SameNonterm = -1
	FinalState = -1
)


type State struct {
	Rules map[int]Rule   `json:",omitempty"`
	MultiRules map[int][]Rule `json:",omitempty"`
}

const AnyTerm = -1


type Nonterm struct {
	Name string
	Group int
	States []State
}

type Grammar struct {
	Terms []Term
	Nonterms []Nonterm
}
