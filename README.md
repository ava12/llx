# Universal LL(*) parser library

The goal is to create a library suitable for making different parsing tools (translators, linters, style 
checkers/formatters, etc.). It is not meant to be super-fast or to be able to process gigabyte files. Grammars are 
described in EBNF-like language, but unlike most parser generators this library does not generate _parsers_, only 
_data_ used by the built-in parser.

This README assumes that the reader is familiar with the terms like «parser», «lexer», «grammar», «token», «terminal», 
«non-terminal».

Parser uses finite-state machine with a stack for non-terminals. The bottom of the stack is the root non-terminal 
(representing the whole grammar), and the top is the current one. Non-terminal is pushed on the stack when it is 
expanded and dropped from the stack when all tokens it was expanded to are consumed by the parser.

Parser does not build parse tree by default, it allows using token and non-terminal hooks instead. A token hook is 
triggered when a new token is read from source file. A non-terminal hook is triggered when parser starts or finishes 
processing a non-terminal, or when a token is consumed.

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

!aside $space;
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
        parser.AnyToken: func (t *parser.Token, pc *parser.ParseContext) (emit bool, e error) {
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
    _, e = configParser.ParseString("input", input, &hooks)
    if e == nil {
        fmt.Println(result)
    } else {
        fmt.Println(e)
    }
}
```

The parser is LL(*), no left recursion allowed. It assumes that in most cases one token lookahead is enough. When 
parser needs a deeper lookahead, a separate parsing branch is created for each possible variant, all branches are 
traced simultaneously (non-terminal hooks are not used), fetched tokens and applied rules for each branch are 
memoized. When deep lookahead is needed during this process, branches are further split. Branch is discarded when 
an error found or when the non-terminal caused initial branching is dropped from the stack (thus the variant consuming 
the longest run of tokens will be preferred). This process is stopped when there is only one (or none) branch left, and 
then captured tokens and rules are «replayed».

## Goal feature list

  - [x] multiple sources for the same parsing process
  - [x] multiple parsers for the same source queue
  - [x] multiple lexers for the same grammar
  - [x] aside tokens
  - [x] external tokens
  - [x] token hooks
  - [x] non-terminal hooks
  - [x] parse tree generation and manipulation
  - [ ] error recovery

## Implemented features

### Multiple sources for the same parsing process

Parser uses source queues that may contain more than one source (prologues, epilogues, included files). All sources 
in a queue yield unified stream of tokens, there is only one limit: a token cannot span across source boundaries. 
Every token contains a pointer to the source it originates from.

### Multiple parsers for the same source queue

Source queue allows using multiple parsers. E.g. an HTML parser hook can process embedded style sheet with a CSS 
parser and then continue with HTML.

### Multiple lexers for the same grammar

Grammar definitions allow splitting tokens into groups, a separate lexer is constructed for each group. This is 
useful in case lexer output is context-sensitive. E.g. HTML grammar needs two lexers: one to separate tags from raw 
text and another one for tag internal parts.

### Aside tokens

A source may contain tokens which are not used in grammar rules, e.g. spaces, line breaks, comments. Such tokens 
still can be hooked, e.g. to autogenerate semicolons at line breaks. Normally aside tokens are dropped, but token 
hooks can instruct the parser to keep them, e.g. keep comments for documentation generator.

### External tokens

Grammar definitions may contain tokens which are not present in sources, but may be emitted by token hooks. E.g. 
fake `INDENT` and `DEDENT` tokens emitted when source indentation level changes.

### Token hooks

A token hook is a function which is called by the parser when specific (or any) token is fetched from source file. A 
hook can instruct the parser whether to keep that token or to drop it. It can also insert new tokens, manipulate source 
queue, use another parser for the next part of source, etc.

### Non-terminal hooks

Non-terminal hooks are functions which are called by the parser when one of four events occur during processing 
specific (or any) non-terminals. The events are:

  - new non-terminal is pushed on the stack;
  - non-terminal is dropped from the stack; handler returns some value (can be anything) which will be passed to 
    parent non-terminal;
  - new token is consumed;
  - nested non-terminal is dropped from the stack; handler receives the value returned by nested non-terminal handler.
