/*
llxgen is a console utility translating grammar description to Go or JSON file.
Usage is

	llxgen ([-j] | [-p <name>] [-v <name>]) [-o <name>] <file>

-j flag instructs llxgen to output JSON file instead of Go source;

-o <name> defines output file name, default is the name of input file with .go or .json suffix;

-p <name> defines Go package name, default is directory name of input file;

-v <name> defines generated Go variable name of type *grammar.Grammar, default is the name of root node;

<file> defines grammar definition file parsable by langdef.Parse().
*/
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/langdef"
)

var (
	generateJson                                  bool
	inFileName, outFileName, packageName, varName string
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage is  llxgen ([-j] | [-p <name>] [-v <name>]) [-o <name>] <file>")
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), "  <file>")
		fmt.Fprintln(flag.CommandLine.Output(), "\tgrammar definition file name")
	}

	flag.BoolVar(&generateJson, "j", false, "output JSON instead of Go")
	flag.StringVar(&outFileName, "o", "", "output file name, default is the name of input file with .go or .json suffix")
	flag.StringVar(&packageName, "p", "", "Go package name, default is dir name of output file")
	flag.StringVar(&varName, "v", "", "Go variable name, default is the root node name")
	flag.Parse()
	inFileName = flag.Arg(0)
	if inFileName == "" {
		flag.Usage()
		os.Exit(2)
	}

	if outFileName == "" {
		ext := filepath.Ext(inFileName)
		outFileName = inFileName[:len(inFileName)-len(ext)]
		if generateJson {
			outFileName += ".json"
		} else {
			outFileName += ".go"
		}
	}

	var gr *grammar.Grammar
	src, e := os.ReadFile(inFileName)
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
		e = os.WriteFile(outFileName, content, 0o666)
	}

	if e != nil {
		fmt.Println(e.Error())
		os.Exit(3)
	}
}

func makeJson(gr *grammar.Grammar) ([]byte, error) {
	return json.MarshalIndent(gr, "", "  ")
}

func makeGo(gr *grammar.Grammar) ([]byte, error) {
	if packageName == "" {
		dir, e := filepath.Abs(outFileName)
		if e != nil {
			return nil, e
		}

		dir, _ = filepath.Split(dir)
		_, packageName = filepath.Split(dir[:len(dir)-1])
	}
	if varName == "" {
		varName = gr.Nodes[0].Name
	}

	re := regexp.MustCompile("^[A-Za-z_][A-Za-z_0-9]*$")
	if !re.MatchString(packageName) {
		return nil, fmt.Errorf("invalid package name: %s", packageName)
	}
	if !re.MatchString(varName) {
		return nil, fmt.Errorf("invalid variable name: %s", packageName)
	}

	var buffer bytes.Buffer

	buffer.WriteString("// Code generated with llxgen.\n\n" +
		"package " + packageName + "\n\n" +
		"import \"github.com/ava12/llx/grammar\"\n\n" +
		"var " + varName + " = &grammar.Grammar{\n")

	buffer.WriteString("\tTokens: []grammar.Token{\n")
	for _, t := range gr.Tokens {
		buffer.WriteString(fmt.Sprintf("\t\t{Name: %q, Re: %q, Group: %d, Flags: %d},\n", t.Name, t.Re, t.Group, t.Flags))
	}
	buffer.WriteString("\t},\n")

	buffer.WriteString("\tNodes: []grammar.Node{\n")
	for _, nt := range gr.Nodes {
		buffer.WriteString(fmt.Sprintf("\t\t{Name: %q, FirstState: %d},\n", nt.Name, nt.FirstState))
	}
	buffer.WriteString("\t},\n")

	buffer.WriteString("\tStates: []grammar.State{\n")
	for _, st := range gr.States {
		buffer.WriteString(fmt.Sprintf("\t\t{%d, %d, %d, %d, %d},\n", st.TokenTypes, st.LowMultiRule, st.HighMultiRule, st.LowRule, st.HighRule))
	}
	buffer.WriteString("\t},\n")

	buffer.WriteString("\tMultiRules: []grammar.MultiRule{")
	if len(gr.MultiRules) == 0 {
		buffer.WriteString("},\n")
	} else {
		for _, mr := range gr.MultiRules {
			buffer.WriteString(fmt.Sprintf("\n\t\t{%d, %d, %d},", mr.Token, mr.LowRule, mr.HighRule))
		}
		buffer.WriteString("\n\t},\n")
	}

	buffer.WriteString("\tRules: []grammar.Rule{")
	if len(gr.Rules) == 0 {
		buffer.WriteString("},\n")
	} else {
		for _, r := range gr.Rules {
			buffer.WriteString(fmt.Sprintf("\n\t\t{%d, %d, %d},", r.Token, r.State, r.Node))
		}
		buffer.WriteString("\n\t},\n")
	}

	buffer.WriteString("}\n")
	return buffer.Bytes(), nil
}
