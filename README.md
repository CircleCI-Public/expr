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

## Variables
The interpreter looks in the environment mapping identifiers to values when it
encounters an identifier.

If the identifier is found in the environment the value is used. It is an error
to refer to a variable that isn't found in the environment.

If the variable is found but the value is `null` then the variable is treated
as being declared but not defined and is given the special `undefined` value.

This has the following behaviour:
* `undefined` isn't a value, so it is neither equal, nor not-equal with
  `undefined`.
  * `foo == foo` is false
  * `foo != foo` is false
* If an equality or comparison operator has one or more `undefined` operands
  then the result of the comparison is `undefined`. All of the following
  (non-exhaustive) list will evaluate `undefined` when `foo` isn't defined:
  * `1 == foo`
  * `foo != 1`
  * `foo == foo`
  * `foo != foo`
  * `foo > 3`
  * `"hello" starts-with foo`
* `undefined` is treated as `false` in a truthiness context, such as logical
  operators or when the interpreter needs to determine the final truthiness of
  an expression:
  * If a logical operator has an `undefined` operand then the `undefined` operand
    will be treated as though it had the value `false`, but the `undefined` value
    will be preserved. This allows you to deal with possibly undefined variables
    and provide alternatives:
    * `foo or 12` evaluates to `12`
    * `12 or foo` evaluates to `12`
    * `foo and "hello"` evaluates to `undefined`
    * `"hello" and foo` evaluates to `undefined`
    * `not foo` evaluates to `true`
  * If an expression ultimately evaluates to `undefined` the interpreter will
    return `false`.

## Building
Build the Java sources with `lein javac`

## Tests
Run tests with `lein test`
