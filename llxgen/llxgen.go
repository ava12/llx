package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/langdef"
)

var (
	generateJson, useShortImport bool
	inFileName, outFileName, packageName, varName string
)

func main () {
	flag.Usage = func () {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage is  llxgen ([-j] | [-p <name>] [-v <name>] [-s]) [-o <name>] <file>")
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), "  <file>")
		fmt.Fprintln(flag.CommandLine.Output(), "\tgrammar definition file name")
	}

	flag.BoolVar(&generateJson, "j", false, "output JSON instead of Go")
	flag.StringVar(&outFileName, "o", "", "output file name, default is the name of input file with .go or .json suffix")
	flag.StringVar(&packageName, "p", "", "Go package name, default is dir name of output file")
	flag.BoolVar(&useShortImport, "s", false, "use short path (\"llx/grammar\") for Go import")
	flag.StringVar(&varName, "v", "", "Go variable name, default is the root nonterminal name")
	flag.Parse()
	inFileName = flag.Arg(0)
	if inFileName == "" {
		flag.Usage()
		os.Exit(2)
	}

	if outFileName == "" {
		ext := filepath.Ext(inFileName)
		outFileName = inFileName[: len(inFileName) - len(ext)]
		if generateJson {
			outFileName += ".json"
		} else {
			outFileName += ".go"
		}
	}

	var gr *grammar.Grammar
	src, e := ioutil.ReadFile(inFileName)
	if e == nil {
		gr, e = langdef.ParseBytes(inFileName, src)
	}
	var content []byte
	if e == nil {
		if generateJson {
			content, e = makeJson(gr)
		} else {
			content, e = makeGo(gr)
		}
	}
	if e == nil {
		e = ioutil.WriteFile(outFileName, content, 0o666)
	}

	if e != nil {
		fmt.Println(e.Error())
		os.Exit(3)
	}
}

func makeJson (gr *grammar.Grammar) ([]byte, error) {
	return json.MarshalIndent(gr, "", "  ")
}

func makeGo (gr *grammar.Grammar) ([]byte, error) {
	if packageName == "" {
		dir, e := filepath.Abs(outFileName)
		if e != nil {
			return nil, e
		}

		dir, _ = filepath.Split(dir)
		_, packageName = filepath.Split(dir[: len(dir) - 1])
	}
	if varName == "" {
		varName = gr.Nonterms[0].Name
	}

	re := regexp.MustCompile("^[A-Za-z_][A-Za-z_0-9]*$")
	if !re.MatchString(packageName) {
		return nil, fmt.Errorf("invalid package name: %s", packageName)
	}
	if !re.MatchString(varName) {
		return nil, fmt.Errorf("invalid variable name: %s", packageName)
	}

	var buffer bytes.Buffer
	importString := "llx/grammar"
	if !useShortImport {
		importString = "github.com/ava12/" + importString
	}

	buffer.WriteString("// Code generated with llxgen.\n" +
		"package " + packageName + "\n\n" +
		"import \"" + importString + "\"\n\n" +
		"var " + varName + " = &grammar.Grammar{\n")

	buffer.WriteString("\tTerms: []grammar.Term{\n")
	for _, t := range gr.Terms {
		buffer.WriteString(fmt.Sprintf("\t\t{Name: %q, Re: %q, Groups: %d, Flags: %d},\n", t.Name, t.Re, t.Groups, t.Flags))
	}
	buffer.WriteString("\t},\n")

	buffer.WriteString("\tNonterms: []grammar.Nonterm{\n")
	for _, nt := range gr.Nonterms {
		buffer.WriteString(fmt.Sprintf("\t\t{Name: %q, States: []grammar.State{\n", nt.Name))
		for _, st := range nt.States {
			buffer.WriteString(fmt.Sprintf("\t\t\t{Group: %d, ", st.Group))
			hasRules := (len(st.Rules) != 0)
			hasMulti := len(st.MultiRules) != 0

			if hasRules {
				buffer.WriteString("Rules: map[int]grammar.Rule{\n")
				for k, r := range st.Rules {
					buffer.WriteString(fmt.Sprintf("\t\t\t\t%d: {State: %d, Nonterm: %d},\n", k, r.State, r.Nonterm))
				}
				if hasMulti {
					buffer.WriteString("\t\t\t}, MultiRules: map[int][]grammar.Rule{\n")
				} else {
					buffer.WriteString("\t\t\t}},\n")
				}
			}

			if hasMulti {
				if !hasRules {
					buffer.WriteString("MultiRules: map[int][]grammar.Rule{\n")
				}
				for k, rs := range st.MultiRules {
					buffer.WriteString(fmt.Sprintf("\t\t\t\t%d: {\n", k))
					for _, r := range rs {
						buffer.WriteString(fmt.Sprintf("\t\t\t\t\t{State: %d, Nonterm: %d},\n", r.State, r.Nonterm))
					}
					buffer.WriteString(fmt.Sprintf("\t\t\t\t},\n"))
				}
				buffer.WriteString("\t\t\t}},\n")
			}
		}
		buffer.WriteString("\t\t}},\n")
	}
	buffer.WriteString("\t},\n")

	buffer.WriteString("}\n")
	return buffer.Bytes(), nil
}
