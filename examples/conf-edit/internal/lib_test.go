package internal

import (
	"strings"
	"testing"
)

var cmdHandlers = [4]func(c *Conf, cmd string){
	func(c *Conf, cmd string) {
		c.RemoveSection(cmd)
	},
	func(c *Conf, cmd string) {
		c.AddSection(cmd)
	},
	func(c *Conf, cmd string) {
		c.RemoveEntry(cmd)
	},
	func(c *Conf, cmd string) {
		parts := strings.SplitN(cmd, "=", 2)
		value := ""
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		}
		c.SetEntry(parts[0], value)
	},
}

func diff (a, b string) int {
	l := len(a)
	if len(b) < l {
		l = len(b)
	}
	i := 0
	for i < l && a[i] == b[i] {
		i++
	}
	return i
}

type sample struct {
	src, cmd, expected string
}

func checkSamples (t *testing.T, samples []sample) {
	for i, s := range samples {
		src := []byte(s.src)
		conf, e := Parse("", &src)
		if e != nil {
			t.Errorf("sample #%d: unexpected parsing error: %s", i, e.Error())
			continue
		}

		subs := strings.SplitN(s.cmd, "|", 4)
		for i, s := range subs {
			if s == "" {
				continue
			}

			cs := strings.Split(s, ",")
			for _, s := range cs {
				cmdHandlers[i](conf, strings.TrimSpace(s))
			}
		}

		w := strings.Builder{}
		_, e = Serialize(conf.RootNode, &w)
		if e != nil {
			t.Errorf("sample #%d: unexpected serializing error: %s", i, e.Error())
			continue
		}

		got := w.String()
		dp := diff(s.expected, got)
		if dp < len(s.expected) || dp < len(got) {
			t.Errorf("sample #%d: pos %d: expecting %.20q, got %.20q", i, dp, s.expected[dp :], got[dp :])
		}
	}
}

func TestEmptySource (t *testing.T) {
	samples := []sample{
		{"", "|||foo=bar", "foo=bar\n"},
		{"", "|||a.b=c,name=value,foo.bar=baz,a.b=d,foo.bar=qux", "name=value\n\n[a]\nb=d\n\n[foo]\nbar=qux\n"},
	}
	checkSamples(t, samples)
}

func TestEmptyValue (t *testing.T) {
	samples := []sample{
		{"", "|||foo,bar=baz", "foo=\nbar=baz\n"},
		{"foo=\n", "|||foo", "foo=\n"},
		{"foo=\n", "|||foo=bar", "foo=bar\n"},
		{"foo=bar\n", "|||foo", "foo=\n"},
	}
	checkSamples(t, samples)
}

func TestDataPreserved (t *testing.T) {
	samples := []sample{
		{"\n", "|||foo=bar", "foo=bar\n\n"},
		{"\n#comment\n", "|||foo=bar,user.name=root", "foo=bar\n\n#comment\n[user]\nname=root\n"},
		{"#comment\n", "|||foo=bar,user.name=root", "#comment\nfoo=bar\n\n[user]\nname=root\n"},
		{"#comment\n\n", "|||foo=bar,user.name=root", "#comment\nfoo=bar\n\n[user]\nname=root\n"},
		{"#one\n#two\n", "|||foo=bar,user.name=root", "#one\n#two\nfoo=bar\n\n[user]\nname=root\n"},
		{"foo=bar\n\n[user]\nlogin=root\n", "|||bar=baz,user.name=admin", "foo=bar\nbar=baz\n\n[user]\nlogin=root\nname=admin\n"},
		{"[user]\nname = admin\nlogin = root\n\n", "user.name", "[user]\nname = admin\nlogin = root\n\n"},
		{"[user]\nname = admin\nlogin = root\n\n", "|user.name", "[user]\nname = admin\nlogin = root\n\n[user.name]\n"},
		{"foo = bar #comment\n", "|||foo=baz", "foo = baz#comment\n"},
	}
	checkSamples(t, samples)
}

func TestDataModified (t *testing.T) {
	samples := []sample{
		{"foo=bar\nbar=baz\n\n", "||foo", "bar=baz\n\n"},
		{
			"[user]\nname=admin\n[user.contact]\n#net\naddr=localhost\n[user.contact.phone]\nwork=911\n",
			"user.contact",
			"[user]\nname=admin\n[user.contact.phone]\nwork=911\n",
		},
		{"foo=bar\n[foo]\nbar=qux\n", "|bar", "foo=bar\n[foo]\nbar=qux\n\n[bar]\n"},
		{"[user]\nname = admin\nlogin = root\n\n", "||user.name", "[user]\nlogin = root\n\n"},
		{"foobar=3\nfoo=1\nbar=2\n", "||foo|qux=4", "foobar=3\nbar=2\nqux=4\n"},
	}
	checkSamples(t, samples)
}