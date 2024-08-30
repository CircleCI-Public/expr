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
  (:require [clojure.string :as string])
  (:import (com.circleci.expr Scanner
                              Scanner$ScanError
                              Parser
                              Parser$ParseError
                              Interpreter
                              Interpreter$Error
                              VariableAnalyser)))

(defn- build-error-message
  [preamble expression error-pos error-length]
  (let [;; This `inc` needs a little bit of explaining.
        ;; `String.lastIndexOf` returns the index of the last occurrence of a
        ;; newline working backwards from the error position. If there isn't a
        ;; newline preceding the error position then `String.lastIndexOf`
        ;; returns -1
        ;;
        ;; In either case, incrementing the result of `String.lastIndexOf` gets
        ;; the start index of the line the error occurred on:
        ;; 1. A newline was found: `.lastIndexOf` is the index of the newline,
        ;;    the next index is the start of the line.
        ;; 2. A newline was not found: `.lastIndexOf` is -1, the next index (0)
        ;;    is the start of the line.
        line-start (inc (.lastIndexOf expression (int \newline) error-pos))
        line-end (.indexOf expression (int \newline) error-pos)
        error-line (subs expression line-start (if (pos? line-end)
                                                 line-end
                                                 (count expression)))]
    (string/join \newline [preamble
                           error-line
                           (str (.repeat " " (- error-pos line-start))
                                (.repeat "^" error-length))])))

(defn- pretty-scan-error
  [^Scanner$ScanError error ^String expression]
  (let [error-pos (.-errorPos error)
        error-char (.-errorChar error)
        error-type (.name (.-type error))]
    (case error-type
      "UNEXPECTED_CHARACTER"
      (build-error-message (format "Unexpected character '%s':" error-char)
                           expression
                           error-pos
                           1)

      "INCOMPLETE_EQUALS"
      (build-error-message (format "Incomplete token, expected \"==\", found '%s':" error-char)
                           expression
                           error-pos
                           1)

      "UNTERMINATED_STRING"
      (build-error-message (format "Unterminated string starting here:")
                           expression
                           error-pos
                           1)

      (format "Unknown error scanning expression: '%s'" expression))))

(defn- pretty-parse-error
  [^Parser$ParseError error ^String expression]
  (let [token (.-token error)
        error-pos (.-charPos token)
        error-string (.-lexeme token)
        error-type (.name (.-type error))]
    (case error-type
      "UNEXPECTED_ADDITIONAL_INPUT"
      (build-error-message (format "Unexpected additional input, found \"%s\", expected EOF:" error-string)
                           expression
                           error-pos
                           (count error-string))

      "EXPECTED_EXPRESSION"
      (build-error-message (format "Expected expression, found \"%s\":" error-string)
                           expression
                           error-pos
                           (count error-string))

      "EXPECTED_RIGHT_PAREN"
      (build-error-message "Expected ')' after expression:"
                           expression
                           error-pos
                           1)

      (format "Unknown error parsing expression: '%s'" expression))))

(defn- pretty-interpreter-error
  [^Interpreter$Error error ^String expression]
  (let [token (.-token error)
        error-pos (.-charPos token)
        error-string (.-lexeme token)
        error-type (.name (.-type error))]
    (case error-type
      "EXPECTED_NUMERIC_OPERAND"
      (build-error-message (format "Expected numeric operands to \"%s\" operator:" error-string)
                           expression
                           error-pos
                           (count error-string))

      "EXPECTED_STRING_OPERAND"
      (build-error-message (format "Expected string operands to \"%s\" operator:" error-string)
                           expression
                           error-pos
                           (count error-string))

      "UNKNOWN_VARIABLE"
      (build-error-message (format "Referred to a variable \"%s\" that does not exist:" error-string)
                           expression
                           error-pos
                           (count error-string))

      (format "Unknown error interpreting expression: '%s'" expression))))

(defn parse
  [^String expression]
  (try
    (let [tokens (.scan (Scanner. expression))]
      (.parse (Parser. tokens))
      {:result true})
    (catch Scanner$ScanError e
      {:errors [(pretty-scan-error e expression)]})
    (catch Parser$ParseError e
      {:errors [(pretty-parse-error e expression)]})))

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
    {:errors <collection of error message strings>}"
  [^String expression]
  (try
    (let [tokens (.scan (Scanner. expression))
          expr (.parse (Parser. tokens))]
      {:result true
       :variables (map variable-token->map (.gatherVariables (VariableAnalyser.) expr))})
    (catch Scanner$ScanError e
      {:errors [(pretty-scan-error e expression)]})
    (catch Parser$ParseError e
      {:errors [(pretty-parse-error e expression)]})))

(defn interpret
  [^String expression pipeline-values]
  (try
    (let [tokens (.scan (Scanner. expression))
          expr (.parse (Parser. tokens))]
      {:result (.interpret (Interpreter. pipeline-values) expr)})
    (catch Scanner$ScanError e
      {:errors [(pretty-scan-error e expression)]})
    (catch Parser$ParseError e
      {:errors [(pretty-parse-error e expression)]})
    (catch Interpreter$Error e
      {:errors [(pretty-interpreter-error e expression)]})))
