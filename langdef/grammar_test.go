package langdef

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/ava12/llx/grammar"
)

type (
	state map[int][]grammar.Rule
	nonTerm struct {
		name   string
		states []state
	}
	nonTerms []nonTerm
)

func checkGrammar (g *grammar.Grammar, nti nonTerms) error {
	ntcnt := len(g.NonTerms)
	if ntcnt > len(nti) {
		ntcnt = len(nti)
	}

	for i := 0; i < ntcnt; i++ {
		if g.NonTerms[i].Name != nti[i].name {
			return errors.New(fmt.Sprintf("nt #%d: %s expected, got %s", i, nti[i].name, g.NonTerms[i].Name))
		}
	}

	if len(g.NonTerms) != len(nti) {
		if len(nti) > ntcnt {
			missing := make([]string, 0, len(nti) - ntcnt)
			for i := ntcnt; i < len(nti); i++ {
				missing = append(missing, nti[i].name)
			}
			return errors.New("missing non-terminals: " + strings.Join(missing, ", "))
		} else {
			missing := make([]string, 0, len(g.NonTerms) - ntcnt)
			for i := ntcnt; i < len(g.NonTerms); i++ {
				missing = append(missing, g.NonTerms[i].Name)
			}
			return errors.New("unexpected non-terminals: " + strings.Join(missing, ", "))
		}
	}

	for i, nt := range g.NonTerms {
		e := checkNonTerm(g, i, nti[i])
		if e != nil {
			return errors.New(nt.Name + ": " + e.Error())
		}
	}

	return nil
}

func checkNonTerm (g *grammar.Grammar, nti int, ent nonTerm) error {
	firstState := g.NonTerms[nti].FirstState
	var lastState int
	if nti >= len(g.NonTerms) - 1 {
		lastState = len(g.States)
	} else {
		lastState = g.NonTerms[nti + 1].FirstState
	}
	states := g.States[firstState : lastState]

	if len(states) != len(ent.states) {
		return errors.New(fmt.Sprintf("state lengths differ: %d(%d)", len(states), len(ent.states)))
	}

	for i, s := range states {
		e := checkState(s, firstState, ent.states[i])
		if e != nil {
			return errors.New("state " + strconv.Itoa(i) + ": " + e.Error())
		}
	}

	return nil
}

func checkState (s grammar.State, firstState int, es state) error {
	l := len(s.Rules) + len(s.MultiRules)
	el := len(es)
	if el > l {
		for k := range es {
			_, f := s.Rules[k]
			if !f {
				_, f = s.MultiRules[k]
				if !f {
					return errors.New(fmt.Sprintf("missing rule for %d", k))
				}
			}
		}
	}

	for k, r := range s.Rules {
		ers, f := es[k]
		if !f {
			return errors.New(fmt.Sprintf("unexpected rule for %d", k))
		}

		if len(ers) != 1 {
			return errors.New(fmt.Sprintf("only one rule for %d (%v)", k, r))
		}

		er := ers[0]
		if r.State >= 0 {
			er.State += firstState
		}
		if r.State != er.State || r.NonTerm != er.NonTerm {
			return errors.New(fmt.Sprintf("rules for %d differ: %v (%v)", k, r, er))
		}
	}

	for k, rs := range s.MultiRules {
		ers, f := es[k]
		if !f {
			return errors.New(fmt.Sprintf("unexpected rules for %d", k))
		}

		for _, er := range ers {
			f = false
			for _, r := range rs {
				if r.State == er.State && r.NonTerm == er.NonTerm {
					f = true
					break
				}
			}
			if !f {
				return errors.New(fmt.Sprintf("missing rule %d (%v)", k, er))
			}
		}

		for _, r := range rs {
			f = false
			for _, er := range ers {
				if r.State >= 0 {
					er.State += firstState
				}
				if r.State == er.State && r.NonTerm == er.NonTerm {
					f = true
					break
				}
			}
			if !f {
				return errors.New(fmt.Sprintf("unexpected rule %d (%v)", k, r))
			}
		}
	}

	return nil
}


type sample struct {
	src, ntsrc string
}

/*
ntsrc: nonTerm;nonTerm
nonTerm: name:state/state
state: rules&rules
rules: termIndex=rule|rule
rule: stateIndex,nonTermIndex
*/

func (s sample) nts () nonTerms {
	ntsrc := strings.ReplaceAll(s.ntsrc, "\n", "")
	ntsrc = strings.ReplaceAll(ntsrc, "\r", "")
	ntsrc = strings.ReplaceAll(ntsrc, "\t", "")
	ntsrc = strings.ReplaceAll(ntsrc, " ", "")

	ntDefs := strings.Split(ntsrc, ";")
	result := make(nonTerms, 0, len(ntDefs))
	for _, ntDef := range ntDefs {
		ntPair := strings.Split(ntDef, ":")
		stateDefs := strings.Split(ntPair[1], "/")
		states := make([]state, 0, len(stateDefs))
		for _, stateDef := range stateDefs {
			ruleDefs := strings.Split(stateDef, "&")
			rules := make(state, len(ruleDefs))
			for _, ruleDef := range ruleDefs {
				rulePair := strings.Split(ruleDef, "=")
				indexDefs := strings.Split(rulePair[1], "|")
				indexes := make([]grammar.Rule, 0, len(indexDefs))
				for _, indexDef := range indexDefs {
					indexPair := strings.Split(indexDef, ",")
					indexes = append(indexes, grammar.Rule{atoi(indexPair[0]), atoi(indexPair[1])})
				}
				rules[atoi(rulePair[0])] = indexes
			}
			states = append(states, rules)
		}
		result = append(result, nonTerm{ntPair[0], states})
	}
	return result
}

func atoi (a string) int {
	result, e := strconv.Atoi(a)
	if e != nil {
		panic("wrong integer format: " + a)
	}
	return result
}

func TestGrammar (t *testing.T) {
	terms := "$term = /\\S+/; "
	samples := []sample{
		{terms + "foo = 'bar';", "foo:1=-1,-1"},
		{terms + "foo = 'bar' | 'baz';", "foo:1=-1,-1&2=-1,-1"},
		{terms + "foo = bar|baz; bar='bar'; baz='baz';", "foo:1=-1,1&2=-1,2; bar:1=-1,-1; baz:2=-1,-1"},
		{terms + "foo = ['bar'], 'baz';", "foo:-1=1,-1&1=1,-1/2=-1,-1"},
		{terms + "foo = 'bar', ['baz'];", "foo:1=1,-1/-1=-1,-1&2=-1,-1"},
		{terms + "foo = 'bar', {'baz'};", "foo:1=1,-1/2=1,-1&-1=-1,-1"},
		{terms + "foo = 'bar', {'baz'}, 'qux';", "foo:1=1,-1/2=1,-1&-1=2,-1/3=-1,-1"},
		{
			"$num=/\\d+/; $op=/[()^*\\/+-]/;" +
				"ari=sum; sum=pro,{('+'|'-'),pro}; pro=pow,{('*'|'/'),pow}; pow=val,{'^',val}; val=$num|('(',sum,')');",
			// $num $op + - *   / ^ ( )
			"ari: 0=-1,1 & 7=-1,1;" +
				"sum: 0=1,2&7=1,2 / -1=-1,-1&2=2,-1&3=2,-1 / 0=1,2&7=1,2;" +
				"pro: 0=1,3&7=1,3 / -1=-1,-1&4=2,-1&5=2,-1 / 0=1,3&7=1,3;" +
				"pow: 0=1,4&7=1,4 / -1=-1,-1&6=2,-1 / 0=1,4&7=1,4;" +
				"val: 0=-1,-1&7=1,-1 / 0=2,1 & 7=2,1 / 8=-1,-1",
		},
	}

	for i, s := range samples {
		g, e := ParseString("", s.src)
		if e != nil {
			t.Errorf("sample #%d: unexpected error: %s", i, e.Error())
			continue
		}

		e = checkGrammar(g, s.nts())
		if e != nil {
			t.Errorf("sample #%d : error: %s", i, e.Error())
		}
	}
}
