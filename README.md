# Universal LL(*) parser library

The goal is to create a library suitable for making different parsing tools (translators, linters, style 
checkers/formatters, etc.). It is not meant to be super-fast or to be able to process gigabyte files. Grammars are 
described in EBNF-like language, but unlike most parser generators this library does not generate _parsers_, only 
_data_ used by the built-in parser.

This README assumes that the reader is familiar with the terms like «parser», «lexer», «grammar», «token», «syntax 
tree».

Parser uses finite-state machine with a stack for syntax tree nodes. Parser itself does not keep the whole syntax 
tree, only the node being processed and its ancestors are stored in the stack, a node is dropped as soon as parser 
consumes all tokens forming that node and its descendants.

Parser allows using two types of hooks:

  - token hooks are triggered when a new token is fetched from lexer;
  - node hooks are triggered when a syntax tree node is pushed on stack or dropped, a nested node is processed or a 
    token is consumed by parser.

Usage is:

  1. Describe grammar using EBNF-like language.
  2. Convert this description to state machine data. You can do it «on the fly» using `langdef.Parse()` function
     or run `llxgen` utility to generate .go source file.
  3. Create a new `parser.Parser` with this data.
  4. Prepare source file(s) and hooks.
  5. Run `Parser.Parse()` with those sources and hooks and get a result or an error.

The simplest program could look like this:

```go
package main

import (
    "context"
    "fmt"
    "github.com/ava12/llx/langdef"
    "github.com/ava12/llx/parser"
)

func main () {
    input := "foo = hello\nbar = world\n[sec]\nbaz =\n[sec.subsec]\nqux = !\n"

    grammar := `
$space = /[ \t\r]+/; $nl = /\n/;
$op = /[=\[\]]/;
$name = /[a-z]+/; $sec-name = /[a-z]+(?:\.[a-z]+)*/;
$value = /[^\n]+/;

!side $space;
!group $sec-name; !group $value $nl; !group $op $name $nl;

config = {section | value | $nl};
section = '[', $sec-name, ']', $nl;
value = $name, '=', [$value], $nl;
`
    configGrammar, e := langdef.ParseString("grammar", grammar)
    if e != nil {
        fmt.Println(e)
        return
    }

    configParser := parser.New(configGrammar)
    result := make(map[string]string)
    prefix, name, value := "", "", ""
    hooks := parser.Hooks{Tokens: parser.TokenHooks{
        parser.AnyToken: func (_ context.Context, t *parser.Token, pc *parser.ParseContext) (emit bool, e error) {
            switch t.TypeName() {
            case "sec-name":
                prefix = t.Text() + "."
            case "name":
                name = prefix + t.Text()
            case "value":
                value = t.Text()
            case "nl":
                if name != "" {
                    result[name] = value
                    name, value = "", ""
                }
            }
            return true, nil
        },
    }}
    _, e = configParser.ParseString(context.Background(), "input", input, hooks)
    if e == nil {
        fmt.Println(result)
    } else {
        fmt.Println(e)
    }
}
```

The parser is LL(*), no left recursion allowed. It assumes that in most cases one token lookahead is enough. When 
parser needs a deeper lookahead (i.e. there are conflicting parsing rules), a separate parsing branch is created for 
each possible variant, all branches are traced simultaneously (node hooks are not used at this stage), fetched tokens 
and applied rules for each branch are recorded. If another conflict is found while tracing a branch that branch is 
split again. Branch is discarded when a syntax error encountered or when the node caused initial branching is dropped 
from the stack (thus the variant consuming the longest run of tokens will be preferred). This process is stopped 
when there is only one (or none) branch left, and then captured tokens and rules are «replayed» using node hooks.

## Features

### Multiple sources for the same parsing process

Parser uses source queues that may contain more than one source (prologues, epilogues, included files). All sources 
in a queue yield unified stream of tokens, there is only one limit: a token cannot span across source boundaries. 
Every token fetched from source file contains a reference to its source. Hooks may add, drop, or seek sources freely.

### Multiple parsers for the same source queue

Source queue allows using multiple parsers. E.g. an HTML parser hook can process embedded style sheet with a CSS 
parser and then continue with HTML using the same queue.

### Multiple lexers for the same grammar

Grammar definitions allow splitting tokens into groups, a separate lexer is constructed for each group. This is 
useful in case lexer output is context-sensitive. E.g. HTML grammar needs two lexers: one to separate tags from raw 
text and another one for tag internal parts.

### Side tokens

A source may contain tokens which are not used in grammar rules, e.g. spaces, line breaks, comments. Such tokens 
still can be hooked, e.g. to autogenerate semicolons at line breaks.

### External tokens

Grammar definitions may contain tokens which are not present in sources, but may be emitted by token hooks. E.g. 
fake `INDENT` and `DEDENT` tokens emitted when source indentation level changes.

### Token hooks

A token hook is a function which is called by the parser when specific (or any) token is fetched from source file. A 
hook can instruct the parser whether to keep that token or to drop it. It can also insert new tokens, manipulate source 
queue, use another parser for the next part of source, etc.

### Node hooks

Node hooks are functions which are called by the parser when one of four events occur during processing 
specific (or any) nodes. The events are:

  - new node is pushed on the stack;
  - node is dropped from the stack; handler returns some value (can be anything) which will be passed to 
    parent node (or returned as parsing result if it is the root node);
  - new token is consumed;
  - nested node is dropped from the stack; handler receives the value returned by nested node handler.

By default only non-side tokens are sent to node hooks. Parse option `WithSides()` can be used to send all tokens, even side ones, e.g. to keep comments for documentation generator.