; Permission is hereby granted, free of charge, to any person obtaining a copy
; of this software and associated documentation files (the "Software"), to
; deal in the Software without restriction, including without limitation the
; rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
; sell copies of the Software, and to permit persons to whom the Software is
; furnished to do so, subject to the following conditions:
;
; The above copyright notice and this permission notice shall be included in
; all copies or substantial portions of the Software.
;
; THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
; IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
; FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
; AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
; LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
; FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
; IN THE SOFTWARE.
(ns expr.core
  (:import (com.circleci.expr Scanner
                              Scanner$ScanError
                              Parser
                              Parser$ParseError
                              Interpreter
                              Interpreter$Error
                              Options
                              VariableAnalyser)))

(def ^:private too-deeply-nested
  ;; Parsing, analysing and interpreting recurse for each level of nesting,
  ;; so a deeply nested expression overflows the stack.
  {:errors ["Expression is too deeply nested"]})

(defn- ->options
  ^Options [{:keys [lists?]}]
  (Options. (boolean lists?)))

(defn parse
  "Parse an expression.

  `options` is a map. `{:lists? true}` enables list literals and the `in`
  operator."
  ([^String expression]
   (parse expression {}))
  ([^String expression options]
   (let [options (->options options)]
     (try
       (let [tokens (.scan (Scanner. expression options))]
         (.parse (Parser. tokens options))
         {:result true})
       (catch Scanner$ScanError e
         {:errors [(.asErrorMessage e expression)]})
       (catch Parser$ParseError e
         {:errors [(.asErrorMessage e expression)]})
       (catch StackOverflowError _
         too-deeply-nested)))))

(defn- variable-token->map
  [token]
  {:name (.-lexeme token)
   :pos (.-charPos token)})

(defn analyse
  "Analyse an expression.
  On success Returns a map of the form:
    {:result true, :variables <variables>}

  where <variables> is a collection of maps of the form
    {:name \"variable name\"
     :pos 6}

  On error returns a map of the form:
    {:errors <collection of error message strings>}

  `options` is a map. `{:lists? true}` enables list literals and the `in`
  operator."
  ([^String expression]
   (analyse expression {}))
  ([^String expression options]
   (let [options (->options options)]
     (try
       (let [tokens (.scan (Scanner. expression options))
             expr (.parse (Parser. tokens options))]
         {:result true
          :variables (map variable-token->map (.gatherVariables (VariableAnalyser.) expr))})
       (catch Scanner$ScanError e
         {:errors [(.asErrorMessage e expression)]})
       (catch Parser$ParseError e
         {:errors [(.asErrorMessage e expression)]})
       (catch StackOverflowError _
         too-deeply-nested)))))

(defn interpret
  "Interpret an expression.

  Interpret returns the truthiness of the eventual value of the expression.

  On success returns a map of the form:
    {:result <boolean result of interpreting expression>}

  On error returns a map of the form:
    {:errors <collection of error message strings>}

  `options` is a map. `{:lists? true}` enables list literals and the `in`
  operator."
  ([^String expression pipeline-values]
   (interpret expression pipeline-values {}))
  ([^String expression pipeline-values options]
   (let [options (->options options)]
     (try
       (let [tokens (.scan (Scanner. expression options))
             expr (.parse (Parser. tokens options))]
         {:result (.interpret (Interpreter. pipeline-values) expr)})
       (catch Scanner$ScanError e
         {:errors [(.asErrorMessage e expression)]})
       (catch Parser$ParseError e
         {:errors [(.asErrorMessage e expression)]})
       (catch Interpreter$Error e
         {:errors [(.asErrorMessage e expression)]})
       (catch StackOverflowError _
         too-deeply-nested)))))

(defn evaluate
  "Evaluate an expression.

  Evaluate returns the eventual value of the expression, which may be any
  scalar type supported by the language.

  On success returns a map of the form:
    {:result <result of evaluating expression>}

  On error returns a map of the form:
    {:errors <collection of error message strings>}

  `options` is a map. `{:lists? true}` enables list literals and the `in`
  operator."
  ([^String expression pipeline-values]
   (evaluate expression pipeline-values {}))
  ([^String expression pipeline-values options]
   (let [options (->options options)]
     (try
       (let [tokens (.scan (Scanner. expression options))
             expr (.parse (Parser. tokens options))]
         {:result (.evaluate (Interpreter. pipeline-values) expr)})
       (catch Scanner$ScanError e
         {:errors [(.asErrorMessage e expression)]})
       (catch Parser$ParseError e
         {:errors [(.asErrorMessage e expression)]})
       (catch Interpreter$Error e
         {:errors [(.asErrorMessage e expression)]})
       (catch StackOverflowError _
         too-deeply-nested)))))
