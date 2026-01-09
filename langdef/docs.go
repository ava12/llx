/*
Package langdef converts textual grammar description to a grammar.Grammar structure.

A grammar is described using a language resembling EBNF. A self-definition of this language is:
*/
//  $$name = /[a-zA-Z_][a-zA-Z_0-9-]*/;
//
//  $space = /[ \r\n\t\f]+/; $comment = /#[^\n]*(?:\n|$)/;
//  $string = /(?:"(?:[^\\"]|\\.)*")|(?:'.*?')/;
//  $name = name;
//  $dir-name = /![a-z]+/;
//  $template-name = /\$\$/ name;
//  $token-name = /\$(?:/ name /)?/;
//  $regexp = /\/(?:[^\\\/]|\\.)+\//;
//  $op = /[(){}\[\]=|,;@]/;
//  $error = /["'!].{0,10}/;
//
//  !side $space $comment; !error $error;
//
//  # the first node is the root one
//  # no token definitions nor directives allowed after the first node
//  langdef = {directive | template-definition | token-definition | layer-definition},
//            node-definition, {node-definition | layer-definition};
//  directive = $dir-name, {$token-name | $string}, ';';
//  template-definition = $template-name, '=', $regexp | $name, {$regexp | name}, ';';
//  token-definition = $token-name, '=', $regexp | $name, {$regexp | name}, ';';
//  layer-definition = '@', $name, {layer-command}, ';';
//  layer-command = $name, '(', [$name | $string, {',', $name | $string}], ')';
//  node-definition = $name, '=', sequence, ';';
//  sequence = item, {',', item};
//  item = variant, {'|', variant}; # NB!: foo | bar, baz is equal to (foo|bar), baz
//  variant = $name | $token-name | $string | group | optional | repeat;
//  group = '(', sequence, ')';
//  optional = '[', sequence, ']'; # match 0 or 1 time
//  repeat = '{', sequence, '}';   # match 0 or more times
/*
A description must be a valid UTF-8 text (no BOM!). Valid space symbols are whitespace (U+0020),
horizontal tabulation (U+0009), line feed (U+000A), form feed (U+000C), and carriage return (U+000D).
Line breaks and indents are insignificant.

A description may contain line comments starting with # and ending with line feed.

Single-quoted string literal is any sequence of symbols except for single quote (')
delimited with single quote signs ('), e.g. 'hello world'.

Double-quoted string literal is any sequence of symbols delimited by double quote signs (").
May contain any symbols, but double quote (") and backslash (\) must be escaped with a backslash,
e.g. "\"Foo\" bar". May also contain special quoted symbols:
  \n          line feed (U+000A)
  \r          carriage return (U+000D)
  \t          horizontal tab (U+0009)
  \x##        (where # is any hexadecimal digit) any byte
  \u####      any rune in range U+0000-U+D7FF, U+E000-U+FFFF
  \U00######  any rune in range U+0000-U+D7FF, U+E000-U+10FFFF

A name is a sequence of latin letters, digits, underscores, and hyphens, starting with a letter or an underscore.
Names are case-sensitive.

A token type name is a name preceded by "$". Token type names are case-sensitive.

A template name is a name preceded by "$$". Template names are case-sensitive.

A regular expression literal is a RE2 regular expression delimited with slashes (/). To use slashes inside regexp
escape them with backslashes (\).

A directive is an exclamation mark followed by small letters.

Operator is one of symbols:
  (){}[]=|,;@

All other symbols not contained in comments or string literals are forbidden.

A grammar description contains five types of records: template definitions, token type definitions,
directives, node definitions, and hook layer definitions.
There must be at least one node definition. All token type definitions and directives
must precede node definitions.

A template definition has a form:
  $$template-name = (/regexp/ | template-name) {, (/regexp/ | template-name)} ;

This defines a template that can be used later in other template or token type definitions.
The resulting regular expression is a concatenation of regexps with delimiters stripped, e.g.
  $$time = /\d\d:\d\d:\d\d/;
  $datetime = /\d{4}-\d\d-\d\d/
                /(?:T/ time /)?/
              /|/ time;
Is equivalent to:
  $datetime = /\d{4}-\d\d-\d\d(?:T\d\d:\d\d:\d\d)?|\d\d:\d\d:\d\d/;

Template definition order is important: all templates must be defined before they can be used, e.g.
  $foo = bar; # bar template is not defined yet
  $$bar = /foo/;
is an error.

A regular expression should not contain capturing groups (e.g. /(foo|bar)+/ will cause incorrect behavior
of lexer), use non-capturing groups instead (e.g. /(?:foo|bar)+/).
By default, token regular expressions use "s" flag (let "." match "\n"), to override it use non-capturing groups
with flags (e.g. /"(?U-s:.*)"/).

A token type definition has a form:
  $type-name = (/regexp/ | template-name) {, (/regexp/ | template-name)} ;

The structure is the same as for a template definition.

There is reserved token type "$" denoting end-of-input token (sent when the source queue is empty).
It is treated as a side token, but it may be used in node definitions, e.g.
  config = {key, "=", value, $new-line | $};

Token definition order is important, a lexer returns the first defined token type it can match.
E.g. a lexer for grammar definition language will match an $error token type only if it sees a quote or exclamation sign,
but cannot match neither a string literal, nor a correct directive name.
Each token type mentioned in a grammar description must be defined exactly once or listed in !extern directive.

A node definition has a form:
  node-name = list ;

A list consists of one or more comma-separated items. An item is one or more variants separated by a pipe ("|") symbol.
A variant is either a node name, a token type, a string literal, or a nested list enclosed in round, square,
or curly braces. Square braces denote optional lists (matched 0 or 1 time), curly braces denote repeated lists
(matched 0 or more times).
NB: foo | bar, baz is the same as (foo | bar), baz.

The first node definition is the root one.
Order of other nodes does not matter, definitions may contain names of nodes that are defined later.
Each node must be defined exactly once, e.g.
  foo = bar, baz; foo = qux; # error: foo already defined
  foo = (bar, baz) | qux; # correct

A hook layer definition has a form:
  @ type {command ( [argument {, argument}] )} ;

type and command are names, argument is a string literal or a name (the latter is a syntactic sugar).
There may be many definitions with the same type, definition may contain many commands with the same name.

A layer definitions contain configuration commands for built-in hook layers.
All tokens and nodes are filtered through those layers, from the first to the last one, before they
are fed to user-provided hooks. Every layer may pass a token "as is", replace or remove it, or add extra tokens.

A directive has a form:
  !name {argument} ;

argument is either a token type name or a string literal.

A directive may contain token types that are defined later. A directive may contain token types, string literals,
or both depending on directive type. A language description may contain several directives of the same type.

  !caseless
!caseless directive lists token types holding case-insensitive strings. String literals matching
case-insensitive token types must be uppercase, e.g.
  $name = /[A-Za-z]+/; !caseless $name;
  block-start = 'begin'; # error: cannot find suitable token type
  block-start = 'Begin'; # same error
  block-start = 'BEGIN'; # correct

  !error
!error directive lists error token types. Lexer returns an error containing the token text when it matches an error token.

  !extern
!extern directive lists token types that are not defined in a grammar description, but may be emitted by hooks.
E.g. $indent and $dedent tokens emitted by hooks when source text indentation level changes.

  !group
!group directive lists token types that must be placed in a separate group. Each token type may be separated
no more than once. Each group effectively defines a separate lexer.
When a parser needs to fetch a token it tries all suitable lexers (based on expected token types)
in order until a token is fetched.
Grouping is useful for fetching a token that can be a prefix of some other (unwanted) token type.
E.g. "-123" depending on context may be parsed as a single negative number or as an "-" operator and a positive number.
In this case "shorter" token type ("-" operator) must be placed in a separate group
leaving "longer" type (number) in default group.
Another case is a "general" type (e.g. raw text) that can be mistaken for a less general type (e.g. a name).
"General" token type must be placed in its own group.

  !literal
!literal directive lists allowed token types for literals and/or string literals allowed in node definitions.
By default, all defined token types and any literals are allowed, i.e. langdef parser accepts any literal
and tries to associate it with all token types that have suitable regular expressions.
If any token type/literal is listed in !literal directive, then all types/literals that are not listed are forbidden.
This can be used to help parser decide which token type should be matched at some point.
E.g. "=" literal by default may be associated both with $operator and $raw-text token types,
and a lexer may return long $raw-text token instead of short $op, which may lead to parsing errors.
Using directive !literal $op; solves this problem.

  !reserved
!reserved directive lists string literals that are treated as reserved words.
If a token text is a reserved word it can be matched as a literal, but not as a token type,
e.g. if a parser expects $name token type and a lexer fetches a "for" reserved word, it is a syntax error.

  !side
!side directive lists token types that do not affect syntax (but may be important for, say, formatters).
Side tokens must not be used in node definitions.

*/
package langdef
