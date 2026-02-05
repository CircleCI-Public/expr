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
(ns expr.core-test
  (:require [clojure.test :refer (deftest is testing)]
            [clojure.string :as string]
            [expr.util :as util]
            [expr.core :as expr])
  (:import (com.circleci.expr Errors
                              Interpreter$Error
                              Parser$ParseError
                              Scanner$ScanError)
           com.google.re2j.Pattern))

(deftest build-error-message
  (testing "Pinpoints errors in single-line expressions"
    (is (= (string/join \newline ["Preamble message:"
                                  "some.number > baz or true"
                                  "              ^^^"])
           (Errors/errorMessage "Preamble message:"
                                "some.number > baz or true"
                                14
                                3))))

  (testing "Pinpoints errors in multi-line expressions"
    (is (= (string/join \newline ["Preamble message:"
                                  "some.number > baz or true"
                                  "              ^^^"])
           (Errors/errorMessage "Preamble message:"
                                (string/join \newline ["foo.bar == \"main\" and"
                                                       "some.number > baz or true"])
                                36
                                3))))

  (testing "Error is on the end of a line"
    (is (= (string/join \newline ["Preamble message:"
                                  "some.number > baz or t"
                                  "                     ^"])
           (Errors/errorMessage "Preamble message:"
                                (string/join \newline ["foo.bar == \"main\" and"
                                                       "some.number > baz or t"
                                                       "1 <= 15"])
                                43
                                1))))

  (testing "Error is after the end of a line"
    (is (= (string/join \newline ["Preamble message:"
                                  "foo.bar.baz == (1 > 5"
                                  "                     ^"])
           (Errors/errorMessage "Preamble message:"
                                "foo.bar.baz == (1 > 5"
                                21
                                1)))))

(deftest pretty-scan-error
  (testing "Unexpected characters"
    (let [expression "2 > 5 && false"
          e (is (thrown-with-msg? Scanner$ScanError #"Unexpected character\."
                  (util/scan expression)))]
      (is (= (string/join \newline ["Unexpected character '&':"
                                    "2 > 5 && false"
                                    "      ^"])
             (.asErrorMessage e expression)))))

  (testing "Incomplete tokens"
    (let [expression "foo = 58"
          e (is (thrown-with-msg? Scanner$ScanError #"Incomplete token, expected \"==\"\."
                  (util/scan expression)))]
      (is (= (string/join \newline ["Incomplete token, expected \"==\", found ' ':"
                                    "foo = 58"
                                    "     ^"])
             (.asErrorMessage e expression)))))

  (testing "Invalid numeric literal"
    (let [expression "foo < 9223372036854775808"
          e (is (thrown-with-msg? Scanner$ScanError #"Invalid numeric literal\."
                  (util/scan expression)))]
      (is (= (string/join \newline ["Invalid numeric literal, numbers can range from 0 to 2^63 - 1:"
                                    "foo < 9223372036854775808"
                                    "      ^"])
             (.asErrorMessage e expression)))))

  (testing "Unterminated strings"
    (let [expression "foo == \"an unterminated string"
          e (is (thrown-with-msg? Scanner$ScanError #"Unterminated string\."
                  (util/scan expression)))]
      (is (= (string/join \newline ["Unterminated string starting here:"
                                    "foo == \"an unterminated string"
                                    "       ^"])
             (.asErrorMessage e expression)))))

  (testing "Unterminated patterns"
    (let [expression "foo matches /an unterminated pattern"
          e (is (thrown-with-msg? Scanner$ScanError #"Unterminated pattern\."
                  (util/scan expression)))]
      (is (= (string/join \newline ["Unterminated pattern starting here:"
                                    "foo matches /an unterminated pattern"
                                    "            ^"])
             (.asErrorMessage e expression)))))

  (testing "Invalid pattern character"
    (let [expression "foo matches /hello \u23f0/"
          e (is (thrown-with-msg? Scanner$ScanError #"Invalid pattern character\."
                  (util/scan expression)))]
      (is (= (string/join \newline ["Invalid pattern character, only ASCII and Latin-1 are allowed in patterns:"
                                    "foo matches /hello \u23f0/"
                                    "                   ^"])
             (.asErrorMessage e expression)))))

  (testing "Bad pattern syntax"
    (let [expression "foo matches /hello (world/"
          e (is (thrown-with-msg? Scanner$ScanError #"Invalid pattern\."
                  (util/scan expression)))]
      (is (= (string/join \newline ["Syntax error in pattern:"
                                    "foo matches /hello (world/"
                                    "            ^"])
             (.asErrorMessage e expression)))))

  (testing "Overly long pattern"
    (let [expression "foo matches /xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/"
          e (is (thrown-with-msg? Scanner$ScanError #"Pattern too long\."
                  (util/scan expression)))]
      (is (= (string/join \newline ["Pattern length exceeded, limit is 128 characters:"
                                    "foo matches /xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/"
                                    "            ^"])
             (.asErrorMessage e expression))))))

(deftest pretty-parse-error
  (testing "Unexpected additional input"
    (let [expression "5 > 4 foo"
          e (is (thrown-with-msg? Parser$ParseError #"Unexpected additional input\."
                  (util/parse (util/scan expression))))]
      (is (= (string/join \newline ["Unexpected additional input, found \"foo\", expected EOF:"
                                    "5 > 4 foo"
                                    "      ^^^"])
             (.asErrorMessage e expression)))))

  (testing "Expected an expression"
    (let [expression "foo and ) bar"
          e (is (thrown-with-msg? Parser$ParseError #"Expected expression\."
                  (util/parse (util/scan expression))))]
      (is (= (string/join \newline ["Expected expression, found \")\":"
                                    "foo and ) bar"
                                    "        ^"])
             (.asErrorMessage e expression)))))

  (testing "Expected a right parenthesis"
    (let [expression "foo and (bar > 3"
          e (is (thrown-with-msg? Parser$ParseError #"Expected '\)' after expression\."
                  (util/parse (util/scan expression))))]
      (is (= (string/join \newline ["Expected ')' after expression:"
                                    "foo and (bar > 3"
                                    "                ^"])
             (.asErrorMessage e expression))))))

(deftest pretty-interpreter-error
  (testing "Expected a numeric operand"
    (let [expression "foo <= true or false"
          e (is (thrown-with-msg? Interpreter$Error #"Expected numeric value\."
                  (-> (util/scan expression)
                      (util/parse)
                      (util/interpret {"foo" 5}))))]
      (is (= (string/join \newline ["Expected numeric operands to \"<=\" operator:"
                                    "foo <= true or false"
                                    "    ^^"])
             (.asErrorMessage e expression)))))

  (testing "Expected a string operand"
    (let [expression "foo starts-with \"api\""
          e (is (thrown-with-msg? Interpreter$Error #"Expected string value\."
                  (-> (util/scan expression)
                      (util/parse)
                      (util/interpret {"foo" 5}))))]
      (is (= (string/join \newline ["Expected string operands to \"starts-with\" operator:"
                                    "foo starts-with \"api\""
                                    "    ^^^^^^^^^^^"])
             (.asErrorMessage e expression)))))

  (testing "Expected a pattern operand"
    (let [expression "\"hello\" matches 5"
          e (is (thrown-with-msg? Interpreter$Error #"Expected regular expression value\."
                  (-> (util/scan expression)
                      (util/parse)
                      (util/interpret {}))))]
      (is (= (string/join \newline ["Expected the right operand to \"matches\" operator to be a pattern:"
                                    "\"hello\" matches 5"
                                    "        ^^^^^^^"])
             (.asErrorMessage e expression)))))

  (testing "Unknown variable"
    (let [expression "1 > 1 or \"main\" != foo and false"
          e (is (thrown-with-msg? Interpreter$Error #"Referred to a variable that is not set\."
                  (-> (util/scan expression)
                      (util/parse)
                      (util/interpret {}))))]
      ;; The escaped quotes in the expression is why the error indicator
      ;; appears to be offset, it's really in the correct place.
      (is (= (string/join \newline ["Referred to a variable \"foo\" that does not exist:"
                                    "1 > 1 or \"main\" != foo and false"
                                    "                   ^^^"])
             (.asErrorMessage e expression))))))

(deftest parse-can-parse
  (testing "no errors"
    (is (= {:result true}
           (expr/parse "foo >= bar"))))

  (testing "scanner error"
    (is (= {:errors [(string/join \newline ["Unexpected character '&':"
                                            "2 > 5 && false"
                                            "      ^"])]}
           (expr/parse "2 > 5 && false"))))

  (testing "parser error"
    (is (= {:errors [(string/join \newline ["Expected ')' after expression:"
                                            "foo and (bar > 3"
                                            "                ^"])]}
           (expr/parse "foo and (bar > 3")))))

(deftest analyse-can-analyse
  (testing "no errors"
    (is (= {:result true
            :variables []}
           (expr/analyse "2 > 5 or true")))

    (is (= {:result true
            :variables [{:name "foo" :pos 0}
                        {:name "bar" :pos 6}
                        {:name "baz" :pos 13}]}
           (expr/analyse "foo > bar or baz")))

    (testing "every use of a variable is reported"
      (is (= {:result true
              :variables [{:name "foo" :pos 0}
                          {:name "foo" :pos 6}
                          {:name "foo" :pos 13}]}
             (expr/analyse "foo > foo or foo")))))

  (testing "scanner error"
    (is (= {:errors [(string/join \newline ["Unexpected character '&':"
                                            "2 > 5 && false"
                                            "      ^"])]}
           (expr/analyse "2 > 5 && false"))))

  (testing "parser error"
    (is (= {:errors [(string/join \newline ["Expected ')' after expression:"
                                            "foo and (bar > 3"
                                            "                ^"])]}
           (expr/analyse "foo and (bar > 3")))))

(deftest interpret-can-interpret
  (testing "no errors"
    (is (= {:result true}
           (expr/interpret "15 < 3 or not (true == false)" {}))))

  (testing "scanner error"
    (is (= {:errors [(string/join \newline ["Unexpected character '&':"
                                            "2 > 5 && false"
                                            "      ^"])]}
           (expr/interpret "2 > 5 && false" {}))))

  (testing "parser error"
    (is (= {:errors [(string/join \newline ["Expected ')' after expression:"
                                            "foo and (bar > 3"
                                            "                ^"])]}
           (expr/interpret "foo and (bar > 3" {}))))

  (testing "interpreter error"
    (is (= {:errors [(string/join \newline ["Referred to a variable \"bar\" that does not exist:"
                                            "foo >= bar"
                                            "       ^^^"])]}
           (expr/interpret "foo >= bar" {"foo" 3})))))

(deftest evaluate-can-evaluate
  (testing "no errors"
    (is (= {:result "hello"}
           (expr/evaluate "15 < 3 or \"hello\"" {}))))

  (testing "scanner error"
    (is (= {:errors [(string/join \newline ["Unexpected character '&':"
                                            "2 > 5 && false"
                                            "      ^"])]}
           (expr/evaluate "2 > 5 && false" {}))))

  (testing "parser error"
    (is (= {:errors [(string/join \newline ["Expected ')' after expression:"
                                            "foo and (bar > 3"
                                            "                ^"])]}
           (expr/evaluate "foo and (bar > 3" {}))))

  (testing "evaluater error"
    (is (= {:errors [(string/join \newline ["Referred to a variable \"bar\" that does not exist:"
                                            "foo >= bar"
                                            "       ^^^"])]}
           (expr/evaluate "foo >= bar" {"foo" 3})))))

(deftest pushing-boundaries
  (testing "many nots"
    (let [expression (str (string/join (repeat 1000 "not "))
                          "true")
          start (System/nanoTime)
          result (expr/interpret expression {})
          end (System/nanoTime)]
      (is (= {:result true} result))
      (is (<= (- end start) 100e6)))))

;; If this test fails then re2j may have fixed the issue that can cause
;; infinite loops with case-folding in expressions.
;; If the fix is a permanent one that ensures the JVM's supported version of
;; Unicode is used for computing case folding tables at run time then we can
;; remove the restriction on expr patterns which are only allowed to contain a
;; subset of Unicode characters
(deftest re2j-issue-168-canary
  (let [f (future (doto (Pattern/compile "(?i)\u1c80")
                    (.matches "c")))
        v (deref f 200 ::timedout)]
    (is (= v ::timedout) "re2j may have fixed https://github.com/google/re2j/issues/168")))
