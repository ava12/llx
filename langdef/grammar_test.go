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
	nonterm struct {
		name string
		states []state
	}
	nonterms []nonterm
)

func checkGrammar (g *grammar.Grammar, nti nonterms) error {
	ntcnt := len(g.Nonterms)
	if ntcnt > len(nti) {
		ntcnt = len(nti)
	}

	for i := 0; i < ntcnt; i++ {
		if g.Nonterms[i].Name != nti[i].name {
			return errors.New(fmt.Sprintf("nt #%d: %s expected, got %s", i, nti[i].name, g.Nonterms[i].Name))
		}
	}

	if len(g.Nonterms) != len(nti) {
		if len(nti) > ntcnt {
			missing := make([]string, 0, len(nti) - ntcnt)
			for i := ntcnt; i < len(nti); i++ {
				missing = append(missing, nti[i].name)
			}
			return errors.New("missing nonterminals: " + strings.Join(missing, ", "))
		} else {
			missing := make([]string, 0, len(g.Nonterms) - ntcnt)
			for i := ntcnt; i < len(g.Nonterms); i++ {
				missing = append(missing, g.Nonterms[i].Name)
			}
			return errors.New("unexpected nonterminals: " + strings.Join(missing, ", "))
		}
	}

	for i, nt := range g.Nonterms {
		e := checkNonterm(nt, nti[i])
		if e != nil {
			return errors.New(nt.Name + ": " + e.Error())
		}
	}

	return nil
}

func checkNonterm (nt grammar.Nonterm, ent nonterm) error {
	if len(nt.States) != len(ent.states) {
		return errors.New(fmt.Sprintf("state counts differ: %d(%d)", len(nt.States), len(ent.states)))
	}

	for i, s := range nt.States {
		e := checkState(s, ent.states[i])
		if e != nil {
			return errors.New("state " + strconv.Itoa(i) + ": " + e.Error())
		}
	}

	return nil
}

func checkState (s grammar.State, es state) error {
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
		if r.State != er.State || r.Nonterm != er.Nonterm {
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
				if r.State == er.State && r.Nonterm == er.Nonterm {
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
				if r.State == er.State && r.Nonterm == er.Nonterm {
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
ntsrc: nonterm;nonterm
nonterm: name:state/state
state: rules&rules
rules: termIndex=rule|rule
rule: stateIndex,nontermIndex
*/

func (s sample) nts () nonterms {
	ntsrc := strings.ReplaceAll(s.ntsrc, "\n", "")
	ntsrc = strings.ReplaceAll(ntsrc, "\r", "")
	ntsrc = strings.ReplaceAll(ntsrc, "\t", "")
	ntsrc = strings.ReplaceAll(ntsrc, " ", "")

	ntDefs := strings.Split(ntsrc, ";")
	result := make(nonterms, 0, len(ntDefs))
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
		result = append(result, nonterm{ntPair[0], states})
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
	samples := []sample{
		{"foo = 'bar';", "foo:-0=-1,-1"},
		{"foo = 'bar' | 'baz';", "foo:0=-1,-1&1=-1,-1"},
		{"foo = bar|baz; bar='bar'; baz='baz';", "foo:-0=-1,1&1=-1,2; bar:0=-1,-1; baz:1=-1,-1"},
		{"foo = ['bar'], 'baz';", "foo:-1=1,-1&0=1,-1/1=-1,-1"},
		{"foo = 'bar', ['baz'];", "foo:0=1,-1/-1=-1,-1&1=-1,-1"},
		{"foo = 'bar', {'baz'};", "foo:0=1,-1/1=1,-1&-1=-1,-1"},
		{"foo = 'bar', {'baz'}, 'qux';", "foo:0=1,-1/1=1,-1&-1=2,-1/2=-1,-1"},
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
