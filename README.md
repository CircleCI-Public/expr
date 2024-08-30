# Expr - a boolean expression evaluator

Expr is a scanner, recursive descent parser, and interpreter for
interpreting boolean expressions of the form:
```
foo and not bar or baz == "qux" or (12 > 14 or not true)
```

An environment mapping identifiers to values can be supplied to the
interpreter for variable lookup.

## Operators
Logical: `and`, `or`
* These are short-circuiting boolean operators

Equality: `==`, `!=`
* string, numeric, and boolean equality

Equality: `starts-with`
* string prefix equality

Comparison: `>=`, `>`, `<=`, `<`
* Numeric comparisons

Negation: `not`
* Boolean negation

Sub-expressions can be grouped with parentheses `(`, `)`

## Literals
Numeric literals are whole integers (longs).
* E.g. `1`, `768`

String literals are enclosed within double-quotes `"`. The `\`
character is used to escape an embedded quote, or to escape an
embedded `\`.
* E.g. `"the quick brown fox"`, `"You can embed \" and \\ characters"`

The boolean literals are `true`, and `false`.

## Grammar

```
expression -> logic_or
logic_or -> logic_and ( "or" logic_and )*;
logic_and -> equality ( "and" equality )*;
equality -> comparison ( ( "==" | "!=" | "starts-with" ) comparison )*;
comparison -> unary ( ( ">=" | ">" | "<=" | "<" ) unary)*;
unary -> "not" unary | primary;
primary -> "true" | "false" | NUMBER | STRING | IDENTIFIER | "(" expression ")"

NUMBER: /\d+/
STRING: "\"" [\"]* "\""
IDENTIFIER: /[a-zA-Z][\w\-]*(?:\.[\w\-]+)*/
```

### Note on strings
The `\` character can be used to escape embedded `"` or `\` characters.

E.g.  `"foo == \"ma\\in\"` is the literal string `foo == "ma\in"`

### Note on identifiers
Identifiers start with an alphabetic character and can be followed by
alphanumeric, `-`, and `_`. Identifiers can be split into segments with `.`,
but `.` may not appear multiple times in a row.

Valid identifiers:
* `foo`
* `foo.bar`
* `foo.bar.baz`
* `fo19-bar.baz_12`

Invalid identifiers:
* `14bar` - must start with an alphabetic character
* `_bar` - must start with an alphabetic character
* `-bar` - must start with an alphabetic character
* `.bar` - may not start with `.`
* `foo..bar` - `.` may not appear multiple times in a row

### Precedence and associativity
Precedence is encoded in the grammar, associativity is determined by
the parser.

The precedence table, from weakest to strongest binding:
```
+-------------+---------------+
| Operator    | Associativity |
+-------------+---------------+
| or          | left          |
+-------------+---------------+
| and         | left          |
+-------------+---------------+
| == !=       | left          |
| starts-with |               |
+-------------+---------------+
| >= > <= <   | left          |
+-------------+---------------+
| not !       |               |
+-------------+---------------+
```

## Building
Build the Java sources with `lein javac`

## Tests
Run tests with `lein test`
