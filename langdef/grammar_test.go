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
	node  struct {
		name   string
		states []state
	}
	nodes []node
)

func checkGrammar(g *grammar.Grammar, nti nodes) error {
	ntcnt := len(g.Nodes)
	if ntcnt > len(nti) {
		ntcnt = len(nti)
	}

	for i := 0; i < ntcnt; i++ {
		if g.Nodes[i].Name != nti[i].name {
			return fmt.Errorf("nt #%d: %s expected, got %s", i, nti[i].name, g.Nodes[i].Name)
		}
	}

	if len(g.Nodes) != len(nti) {
		if len(nti) > ntcnt {
			missing := make([]string, 0, len(nti)-ntcnt)
			for i := ntcnt; i < len(nti); i++ {
				missing = append(missing, nti[i].name)
			}
			return errors.New("missing nodes: " + strings.Join(missing, ", "))
		} else {
			missing := make([]string, 0, len(g.Nodes)-ntcnt)
			for i := ntcnt; i < len(g.Nodes); i++ {
				missing = append(missing, g.Nodes[i].Name)
			}
			return errors.New("unexpected nodes: " + strings.Join(missing, ", "))
		}
	}

	for i, nt := range g.Nodes {
		e := checkNode(g, i, nti[i])
		if e != nil {
			return errors.New(nt.Name + ": " + e.Error())
		}
	}

	return nil
}

func checkNode(g *grammar.Grammar, nti int, ent node) error {
	firstState := g.Nodes[nti].FirstState
	var lastState int
	if nti >= len(g.Nodes)-1 {
		lastState = len(g.States)
	} else {
		lastState = g.Nodes[nti+1].FirstState
	}
	states := g.States[firstState:lastState]

	if len(states) != len(ent.states) {
		return fmt.Errorf("state lengths differ: %d (expecting %d)", len(states), len(ent.states))
	}

	for i, s := range states {
		e := checkState(g, s, firstState, ent.states[i])
		if e != nil {
			return errors.New("state " + strconv.Itoa(i) + ": " + e.Error())
		}
	}

	return nil
}

func checkState(g *grammar.Grammar, s grammar.State, firstState int, es state) error {
	l := s.HighRule - s.LowRule + s.HighMultiRule - s.LowMultiRule
	el := len(es)
	rules := make(map[int]grammar.Rule, s.HighRule-s.LowRule)
	for _, r := range g.Rules[s.LowRule:s.HighRule] {
		rules[r.Token] = r
	}
	multiRules := make(map[int][]grammar.Rule)
	for _, mr := range g.MultiRules[s.LowMultiRule:s.HighMultiRule] {
		multiRules[mr.Token] = g.Rules[mr.LowRule:mr.HighRule]
	}

	if el > l {
		for k := range es {
			_, f := rules[k]
			if !f {
				_, f = multiRules[k]
				if !f {
					return fmt.Errorf("missing rule for %d", k)
				}
			}
		}
	}

	for k, r := range rules {
		ers, f := es[k]
		if !f {
			return fmt.Errorf("unexpected rule for %d (%v)", k, r)
		}

		if len(ers) != 1 {
			return fmt.Errorf("only one rule for %d (%v)", k, r)
		}

		er := ers[0]
		if r.State >= 0 {
			er.State += firstState
		}
		if r.State != er.State || r.Node != er.Node {
			return fmt.Errorf("rules for %d differ: %v (expecting %v)", k, r, er)
		}
	}

	for k, rs := range multiRules {
		ers, f := es[k]
		if !f {
			return fmt.Errorf("unexpected rules for %d", k)
		}

		for _, er := range ers {
			f = false
			for _, r := range rs {
				if r.State == er.State && r.Node == er.Node {
					f = true
					break
				}
			}
			if !f {
				return fmt.Errorf("missing rule %d (%v)", k, er)
			}
		}

		for _, r := range rs {
			f = false
			for _, er := range ers {
				if r.State >= 0 {
					er.State += firstState
				}
				if r.State == er.State && r.Node == er.Node {
					f = true
					break
				}
			}
			if !f {
				return fmt.Errorf("unexpected rule %d (%v)", k, r)
			}
		}
	}

	return nil
}

type sample struct {
	src, ntsrc string
}

/*
ntsrc: node;node
node: name:state/state
state: rules&rules
rules: tokenIndex=rule|rule
rule: stateIndex,nodeIndex
*/

func (s sample) nts() nodes {
	ntsrc := strings.ReplaceAll(s.ntsrc, "\n", "")
	ntsrc = strings.ReplaceAll(ntsrc, "\r", "")
	ntsrc = strings.ReplaceAll(ntsrc, "\t", "")
	ntsrc = strings.ReplaceAll(ntsrc, " ", "")

	ntDefs := strings.Split(ntsrc, ";")
	result := make(nodes, 0, len(ntDefs))
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
					indexes = append(indexes, grammar.Rule{atoi(rulePair[0]), atoi(indexPair[0]), atoi(indexPair[1])})
				}
				rules[atoi(rulePair[0])] = indexes
			}
			states = append(states, rules)
		}
		result = append(result, node{ntPair[0], states})
	}
	return result
}

func atoi(a string) int {
	result, e := strconv.Atoi(a)
	if e != nil {
		panic("wrong integer format: " + a)
	}
	return result
}

func TestGrammar(t *testing.T) {
	tokens := "$tok = /\\S+/; "
	dl := "$d = /[0-9]/; $l = /[a-z]/; "
	samples := []sample{
		{tokens + "foo = 'bar';", "foo:1=-1,-1"},
		{tokens + "foo = 'bar' | 'baz';", "foo:1=-1,-1&2=-1,-1"},
		{tokens + "foo = bar|baz; bar='bar'; baz='baz';", "foo:1=-1,1&2=-1,2; bar:1=-1,-1; baz:2=-1,-1"},
		{tokens + "foo = ['bar'], 'baz';", "foo:1=1,-1&2=-1,-1/2=-1,-1"},
		{tokens + "foo = 'bar', ['baz'];", "foo:1=1,-1/-1=-1,-1&2=-1,-1"},

		{tokens + "foo = 'bar', {'baz'};", "foo: 1=1,-1 / 2=1,-1&-1=-1,-1"},
		{tokens + "foo = 'bar', {'baz'}, 'qux';", "foo:1=1,-1 / 2=1,-1&3=-1,-1"},
		{
			"$num=/\\d+/; $op=/[()^*\\/+-]/;" +
				"ari=sum; sum=pro,{('+'|'-'),pro}; pro=pow,{('*'|'/'),pow}; pow=val,{'^',val}; val=$num|('(',sum,')');",
			// $num $op + - *   / ^ ( )
			"ari: 0=-1,1 & 7=-1,1;" +
				"sum: 0=1,2&7=1,2 / -1=-1,-1&2=2,-1&3=2,-1 / 0=1,2&7=1,2;" +
				"pro: 0=1,3&7=1,3 / -1=-1,-1&4=2,-1&5=2,-1 / 0=1,3&7=1,3;" +
				"pow: 0=1,4&7=1,4 / -1=-1,-1&6=2,-1 / 0=1,4&7=1,4;" +
				"val: 0=-1,-1&7=1,-1 / 0=2,1&7=2,1 / 8=-1,-1",
		},
		{dl + "g = {[$d], $l};", "g: 0=1,-1&1=0,-1&-1=-1,-1 / 1=0,-1"},
		{dl + "g = {{$d}, $l};", "g: 0=1,-1&1=0,-1&-1=-1,-1 / 0=1,-1&1=0,-1"},

		{dl + "g = {$d, [$l]};", "g: 0=1,-1&-1=-1,-1 / 1=0,-1&-1=0,-1"},
		{dl + "g = {$d, {$l}};", "g: 0=1,-1&-1=-1,-1 / 1=1,-1&-1=0,-1"},
		{dl + "g = {[$d], [$l]};", "g: 0=1,-1&1=0,-1&-1=-1,-1 / 1=0,-1&-1=0,-1"},
		{dl + "g = [{$d}, $l];", "g: 0=0,-1&1=-1,-1&-1=-1,-1"},
		{dl + "g = [[$d], $l];", "g: 0=1,-1&1=-1,-1&-1=-1,-1 / 1=-1,-1"},

		{dl + "g = [$d, [$l]];", "g: 0=1,-1&-1=-1,-1 / 1=-1,-1&-1=-1,-1"},
		{dl + "g = [$d, {$l}];", "g: 0=1,-1&-1=-1,-1 / 1=1,-1&-1=-1,-1"},
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
