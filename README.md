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

Matching: `matches`
* regular expression string match. The left-hand side operand must be a string,
  the right-hand side operand must be a pattern value. The `matches` operator
  evaluates true if—and only if—the entire string matches the pattern.

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

Regular expression patterns are enclosed with forward-slashes `/`. The `\`
character is used to escape an embedded `/` or `\`. The pattern language is
[re2](https://github.com/google/re2/wiki/syntax). While the pattern syntax
supports capturing groups, there is no way to make use of the captures in an
Expr expression.
* E.g. `/hello\s+world/`

The boolean literals are `true`, and `false`.

## Grammar

```
expression -> logic_or
logic_or -> logic_and ( "or" logic_and )*;
logic_and -> equality ( "and" equality )*;
equality -> comparison ( ( "==" | "!=" | "starts-with" | "matches" ) comparison )*;
comparison -> unary ( ( ">=" | ">" | "<=" | "<" ) unary)*;
unary -> "not" unary | primary;
primary -> "true" | "false" | NUMBER | STRING | IDENTIFIER | PATTERN | "(" expression ")"

NUMBER: /\d+/
STRING: "\"" [\"]* "\""
PATTERN: "/" [\/]* "/"
IDENTIFIER: /[a-zA-Z][\w\-]*(?:\.[\w\-]+)*/
```

### Note on strings
The `\` character can be used to escape embedded `"` or `\` characters.

E.g.  `"foo == \"ma\\in\"` is the literal string `foo == "ma\in"`

### Note on patterns
The `\` character can be used to escape embedded `/` or `\` characters.

The pattern syntax is [re2](https://github.com/google/re2/wiki/syntax) with a
restriction that only ASCII and Latin-1 characters are supported in the
pattern. Metacharacters (such as `.`) can still match any Unicode character.

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
| matches     |               |
+-------------+---------------+
| >= > <= <   | left          |
+-------------+---------------+
| not !       |               |
+-------------+---------------+
```

## Variables
The interpreter looks in the environment (a mapping of identifiers to values)
when it encounters an identifier.

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

## Test corpus
There is a corpus of test cases in `dev-resources/test-corpus` which can be
used to assert that implementations in different languages have the same
behaviours as the original Java implementation.

Each individual JSON file is a single test case.
Keys:
* `input` - the test inputs, an `expression`, and an `environment`
    * `expression` is a string containing the expression to interpret.
    * `environment` is a JSON object with string keys, and string, boolean,
      integer, or `null` values.
* `expected` - the expected outcome, the content differs if the test case is
  expected to successfully interpret the expression and return a result, or if
  there should be an error reported.
    * If the expression is expected to be interpreted successfully this is a
      JSON object with a single key `result` that maps to the outcome.
    * If the expression is expected to cause an error, this is a JSON object
      with a single key `error` which maps to a JSON object describing the
      error.

### Successful interpretation
A test case expecting the interpreter to return a result:
```json
{
  "input" : {
    "expression" : "1 == 1",
    "environment" : { }
  },
  "expected" : {
    "result" : true
  }
}
```

Since the expression should evaluate `true`, the expected `result` is `true`.
If the expression is expected to evaluatee `false` then the expected `result`
is `false`.

### Error test cases
A test case expecting that the interpreter returns an error:
```json
{
  "input" : {
    "expression" : "42 <= \"hello\"",
    "environment" : { }
  },
  "expected" : {
    "error" : {
      "tokenType" : "LESS_EQUAL",
      "errorType" : "Interpreter/EXPECTED_NUMERIC_OPERAND",
      "lexeme" : "<=",
      "charPos" : 3
    }
  }
}
```
The `error` object will not have a `tokenType` key if the error occured in the scanner.

#### Scan errors
Errors with scanning have the following keys:
* `errorType` - The type of error, values can be:
    * `"Scanner/INCOMPLETE_EQUALS"` - Malformed equality operator.
    * `"Scanner/UNTERMINATED_STRING"` - The scanner is scanning a string but
      reached EOF before reading the closing double-quote character. The
      reported error character is the opening double-quote.
    * `"Scanner/UNEXPECTED_CHARACTER"` - Encountered an invalid character in
      the input.
* `lexeme` - The character where the error was encountered. This isn't strictly
  speaking a `lexeme` at this stage, but it makes dealing with the error cases
  easier if they have consistent key names.
* `charPos` - The position of `errorChar` in the input, 0-indexed.

E.g. for the expression `"text \"an unterminated string"` the response should be:
```json
{
  "errorType": "Scanner/UNTERMINATED_STRING",
  "lexeme": "\"",
  "charPos": 5
}
```

#### Parse errors
Errors with parsing have the following keys:
* `errorType` - The type of error, values can be:
    * `"Parser/EXPECTED_EXPRESSION"` - The input doesn't match the expression grammar.
    * `"Parser/UNEXPECTED_ADDITIONAL_INPUT"` - There is additional input
      remaining after a full expression has been matched. The parser expects to
      consume all of the input expression.
    * `"Parser/EXPECTED_RIGHT_PAREN"` - There are unbalanced parentheses in the
      expression, a `(` was seen without a matching `)`.
* `tokenType` - The type of token being parsed when the error was encountered.
  See `TokenType.java` for a full list.
* `lexeme` - The lexeme for the token being parsed.
* `charPos` - The start position of the token in the input, 0-indexed.

E.g. for the expression `"and"` the response should be:
```json
{
  "errorType": "Parser/EXPECTED_EXPRESSION",
  "tokenType": "AND",
  "lexeme": "and",
  "charPos": 0
}
```

#### Interpreter errors
The interpreter operates on an Abstract Syntax Tree (AST). Each node in the
tree is associated with a parsed token. This token is reported in the
interpreter errors since the token identifies the problematic part of the
input.

Errors with interpreting have the following keys:
* `errorType` - The type of error, values can be:
    * `"Interpreter/EXPECTED_NUMERIC_OPERAND"` - The operator being interpreted
      needs numeric operands, but at least one of the operands is non-numeric.
      For these errors the operator token is reported.
    * `"Interpreter/EXPECTED_STRING_OPERAND"` - The operator being interpreted
      needs string operands, but at least one of the operands is not a string.
      For these errors the operator token is reported.
    * `"Interpreter/UNKNOWN_VARIABLE"` - A variable is referenced in the
      expression which is not available in the environment.
* `tokenType` - The type of the token associated with the AST node where the
  error was encountered.
  See `TokenType.java` for a full list.
* `lexeme` - The lexeme for the token.
* `charPos` - The start position of the token in the input, 0-indexed.

E.g. for the expression `"foo starts-with \"hello\""` with environment `{"foo": 42}`
```json
{
  "errorType": "Interpreter/EXPECTED_STRING_OPERAND",
  "tokenType": "STARTS_WITH",
  "lexeme": "starts-with",
  "charPos": 4
}
```
