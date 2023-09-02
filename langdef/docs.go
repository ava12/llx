/*
Package langdef converts textual grammar description to grammar.Grammar structure.

Grammar is described using language that resembles EBNF. Self-definition of this language is:
*/
//  $space = /[ \r\n\t\f]+/; $comment = /#[^\n]*/;
//  $string = /(?:".*?")|(?:'.*?')/;
//  $name = /[a-zA-z_][a-zA-Z_0-9-]*/;
//  $type-dir = /!(?:aside|caseless|error|extern|group|shrink)\b/;
//  $literal-dir = /!reserved\b/;
//  $mixed-dir = /!literal\b/;
//  $token-name = /\$[a-zA-z_][a-zA-Z_0-9-]*/;
//  $regexp = /\/(?:[^\\\/]|\\.)+\//;
//  $op = /[(){}\[\]=|,;]/;
//  $error = /["'!].{0,10}/;
//  
//  !aside $space $comment; !error $error;
//  
//  # first node is the root one
//  # no further token definitions or directives allowed after this point
//  langdef = {directive | token-definition}, node-definition, {node-definition};
//  directive = type-directive | literal-directive | mixed-directive;
//  type-directive = $type-dir, {$token-name}, ';';
//  literal-directive = $literal-dir, {$string}, ';';
//  mixed-directive = $mixed-dir, {$token-name | $string}, ';';
//  token-definition = $token-name, '=', $regexp, ';';
//  node-definition = $name, '=', sequence, ';';
//  sequence = item, {',', item};
//  item = variant, {'|', variant}; # NB!: foo | bar, baz is equal to (foo|bar), baz
//  variant = $name | $token-name | $string | group | optional | repeat;
//  group = '(', sequence, ')';
//  optional = '[', sequence, ']'; # match 0 or 1 time
//  repeat = '{', sequence, '}';   # match 0 or more times
/*
Description must be a valid UTF-8 text (no BOM!). Valid space symbols are whitespace (U+0020),
horizontal tabulation (U+0009), line feed (U+000A), form feed (U+000C), and carriage return (U+000D).
Line breaks are insignificant, text may be a one-liner.

Description may contain line comments starting with # and ending with line feed.

String literal is any sequence of symbols (except for delimiter)
delimited with either single (') or double (") quote signs.

Name is a sequence of latin letters, digits, underscores, and hyphens, starting with letter or underscore.
Names are case-sensitive.

Token type name is a name preceded by $. Again, token type names are case-sensitive.

Regular expression literal is a RE2 regular expression delimited with slashes (/). To use slashes inside regexp
escape them with backslashes (\).

Operator is one of symbols:
   (){}[]=|,;

All other symbols not contained in comments or string literals are forbidden.

Grammar description contains three types of records: token type definition, node definition, and directive.
There must be at least one node definition. All token type definitions and directives
must precede node definitions.

Token type definition has a form:
   $type-name = /regexp/ ;

Regular expression should not contain capturing groups (e.g. /(foo|bar)+/ will cause incorrect behaviour
of lexer), use non-capturing groups instead (e.g. /(?:foo|bar)+/).
By default, token regular expressions use s flag (let . match \n), to override use non-capturing group
with flags (e.g. /"(?U-s:.*)"/).

Token definition order is important, lexer returns the first defined token type it can match.
E.g. lexer for grammar definition language will return $error token type only if it sees a quote or exclamation sign,
but cannot match neither string literal, nor correct directive name.
Each token type mentioned in grammar description must be defined exactly once or listed in !extern directive.

Node definition has a form:
   node-name = list ;

A list consists of one or more comma-separated items. An item is one or more variants separated by pipe (|) symbol.
A variant is either a node name, a token type, a string literal, or a nested list enclosed in round, square,
or curly braces. Square braces denote optional lists (matched 0 or 1 time), curly braces denote repeated lists
(matched 0 or more times).
NB: foo | bar, baz is the same as (foo | bar), baz.

The first node definition is the root one.
Order of other nodes does not matter, definitions may contain names of nodes that are defined later.
Each node must be defined exactly once, e.g.
   foo = bar, baz; foo = qux; # error: foo already defined
   foo = (bar, baz) | qux; # correct

Directive has a form:
   !name {$token-name | 'string' | "string"} ;

Directive may contain token types that are defined later. Directive may contain token types, string literals,
or both depending on directive type. Language description may contain several directives of the same type.

!aside directive lists token types that do not affect syntax (but may be important for, say, formatters).
Aside tokens must not be used in node definitions, must not be grouped (they belong to all groups).

!caseless directive lists token types holding case-insensitive strings. String literals matching
case-insensitive token types must be uppercase, e.g.
   $name = /[A-Za-z]+/; !caseless $name;
   block-start = 'begin'; # error: cannot find suitable token type
   block-start = 'Begin'; # same error
   block-start = 'BEGIN'; # correct

!error directive lists error token types. Lexer raises error containing token text when it fetches error token.

!extern directive lists token types that are not defined in grammar description, but may be emitted by hooks.
E.g. $indent and $dedent tokens emitted by hooks when source text indentation level changes.

!group directive lists token types forming a separate group, which effectively defines a separate lexer.
All tokens acceptable at some parsing state must belong to the same group
(or more than one group, in this case langdef parser chooses one).
Grouping allows to distinguish tokens that belong to different types but may have same content,
e.g. HTML parser can use two token groups: one to separate tags from raw text and another one to parse tag contents.
There may be no more than 30 groups. By default, all defined token types form a single group.
When !group directive is used each defined type must be assigned to at least one group.

!literal directive lists allowed token types for literals and/or string literals allowed in node definitions.
By default, all defined token types and any literals are allowed, i.e. langdef parser accepts any literal
and tries to associate it with all token types that have suitable regular expressions.
If any token type/literal is listed in !literal directive all types/literals that are not listed are forbidden.
This can be used to help langdef parser decide which token group should be used at some point.
E.g. "=" literal by default may be associated both with $operator and $raw-text token types that belong
to different groups, and langdef parser may choose the $raw-text group, which leads to parsing errors.
Using directive !literal $op; solves this problem.

!reserved directive lists string literals that are treated as reserved words.
If token text is a reserved word it can be matched as literal, but not as token type,
e.g. if parser expects $name token type and lexer fetches a "for" reserved word, a syntax error is raised.

!shrink directive lists shrinkable token types.
When a token of shrinkable type causes a syntax error lexer steps back and tries to re-fetch a shorter token.
E.g. ">>" token may be split into two ">" tokens.

*/
package langdef
