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
(ns expr.parser-test
  (:require [clojure.test :refer (are deftest is testing)]
            [expr.util :as util])
  (:import (com.circleci.expr Parser$ParseError
                              Parser$ParseError$Type
                              TokenType)))

(defn- parse
  [expression]
  (-> (util/scan expression)
      (util/parse)
      (util/->s-expression)))

(deftest literals
  (are [expression expected] (= expected (parse expression))
       "foo" '(identifier "foo")
       "14" '(literal 14)
       "true" '(literal true)
       "false" '(literal false)
       "\"a string\"" '(literal "a string")
       "\"a \\\"str\\\\ing\"" '(literal "a \"str\\ing")))

(deftest logical-expressions
  (are [expression expected] (= expected (parse expression))
       "true or false" '(or (literal true)
                            (literal false))
       "false and false" '(and (literal false)
                               (literal false))
       "14 and true or \"hello\"" '(or (and (literal 14)
                                            (literal true))
                                       (literal "hello"))))

(deftest binary-expressions
  (are [expression expected] (= expected (parse expression))
       "\"str\" == 14" '(== (literal "str")
                            (literal 14))
       "3 != 14" '(!= (literal 3)
                      (literal 14))
       "4 > 3 < 2" '(< (> (literal 4)
                          (literal 3))
                       (literal 2))
       "18 >= 10 <= 16" '(<= (>= (literal 18)
                                 (literal 10))
                             (literal 16))))

(deftest grouping
  (is (= '(>= (literal 18)
              (grouping (<= (literal 10)
                            (literal 16))))
         (parse "18 >= (10 <= 16)"))))

(deftest can-parse-expressions
  (is (= '(or (and (== (identifier "foo.bar")
                       (literal "main"))
                   (<= (literal 1)
                       (literal 4)))
              (< (literal 5)
                 (identifier "baz")))
         (parse "foo.bar == \"main\" and 1 <= 4 or 5 < baz")))

  (is (= '(and (grouping
                 (or (< (literal 5)
                        (literal 4))
                     (not (identifier "foo"))))
               (!= (literal "s")
                   (literal 42)))
         (parse "(5 < 4 or not foo) and \"s\" != 42"))))

(deftest parse-errors
  (testing "requires expressions"
    (let [e (is (thrown-with-msg? Parser$ParseError #"Expected expression"
                                  (parse "and")))]
      (is (= {:type TokenType/AND
              :lexeme "and"
              :pos 0
              :literal "and"}
             (util/token->map (.-token e))))
      (is (= Parser$ParseError$Type/EXPECTED_EXPRESSION
             (.-type e)))))

  (testing "Additional input is an error"
    (let [e (is (thrown-with-msg? Parser$ParseError #"Unexpected additional input\."
                                  (parse "foo and bar baz")))]
      (is (= {:type TokenType/IDENTIFIER
              :lexeme "baz"
              :pos 12
              :literal "baz"}
             (util/token->map (.-token e))))
      (is (= Parser$ParseError$Type/UNEXPECTED_ADDITIONAL_INPUT
             (.-type e)))))

  (testing "Grouping parens must be balanced"
    (let [e (is (thrown-with-msg? Parser$ParseError #"Expected '\)' after expression\."
                                  (parse "(1 > 2 3")))]
      (is (= {:type TokenType/NUMBER
              :lexeme "3"
              :pos 7
              :literal 3}
             (util/token->map (.-token e))))
      (is (= Parser$ParseError$Type/EXPECTED_RIGHT_PAREN
             (.-type e))))

    (let [e (is (thrown-with-msg? Parser$ParseError #"Unexpected additional input\."
                                  (parse "1 > 2) < 3")))]
      (is (= {:type TokenType/RIGHT_PAREN
              :lexeme ")"
              :pos 5}
             (util/token->map (.-token e))))
      (is (= Parser$ParseError$Type/UNEXPECTED_ADDITIONAL_INPUT
             (.-type e))))

    (are [expression] (thrown-with-msg? Parser$ParseError #"Expected '\)' after expression\."
                                        (parse expression))
         "((foo and bar)"
         "0 <= (1 > (2 < 3) >= 5"))

  (testing "Expressions must be well-formed"
    (are [input] (thrown-with-msg? Parser$ParseError #"Expected expression\."
                                   (parse input))
         ""
         "and"
         "foo or"
         "and foo"
         "!"
         "5 <"
         ">= 4"
         "("
         "foo and (")))
