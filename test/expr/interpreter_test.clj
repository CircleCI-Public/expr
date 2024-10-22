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
(ns expr.interpreter-test
  (:require [clojure.test :refer (are deftest is testing)]
            [expr.util :as util])
  (:import (com.circleci.expr Interpreter$Error
                              Interpreter$Error$Type
                              TokenType)))

(deftest interprets-literals
  (are [expression expected] (= expected
                                (-> (util/scan expression)
                                    (util/parse)
                                    (util/interpret {"ident" 5})))
    ;; All literals are true
    "0" true
    "1" true
    "\"string\"" true
    "ident" true
    "true" true
    ;; Except for 'false'
    "false" false))

(deftest logical-expressions
  (are [expression environment expected] (= expected
                                            (-> (util/scan expression)
                                                (util/parse)
                                                (util/interpret environment)))
    ;; ===
    ;; and
    ;; ===
    "true and true" {} true
    "true and false" {} false
    "false and true" {} false
    "false and false" {} false

    ;; 'and' short-circuits, shown here by not needing to reference the
    ;; 'foo' variable that doesn't exist in the environment
    "false and foo" {} false

    ;; Alias is accepted
    "true AND false" {} false

    ;; Environment lookups work
    "true and foo" {"foo" true} true
    "foo and false" {"foo" true} false
    "true and foo" {"foo" false} false

    ;; ==
    ;; or
    ;; ==
    "true or true" {} true
    "true or false" {} true
    "false or true" {} true
    "false or false" {} false

    ;; 'or' short-circuits, shown here by not needing to reference the 'foo'
    ;; variable that doesn't exist in the environment
    "true or foo" {} true

    ;; Alias is accepted
    "false OR true" {} true

    ;; Environment lookups work
    "false or foo" {"foo" true} true
    "foo or foo" {"foo" false} false

    ;; ===
    ;; not
    ;; ===
    "!true" {} false
    "!false" {} true
    ;; not is an alias for !
    "not true" {} false
    "not false" {} true

    ;; Environment works
    "not foo" {"foo" true} false
    "!foo" {"foo" true} false))

(deftest binary-expressions
  (are [expression environment expected] (= expected
                                            (-> (util/scan expression)
                                                (util/parse)
                                                (util/interpret environment)))
    ;; ========
    ;; equality
    ;; ========
    "1 == 1" {} true
    "1 == 1" {} true
    "1 == 2" {} false
    "\"hi\" == \"hi\"" {} true
    "\"hi\" == \"bye\"" {} false
    "true == true" {} true
    "false == false" {} true
    "foo == foo" {"foo" "\"hello\""} true
    "foo == bar" {"foo" 1, "bar" 1} true
    "foo == bar" {"foo" 5, "bar" 1} false

    ;; Mixed types are never equal
    "true == 12" {} false
    "\"string\" == 4" {} false
    "5 == false" {} false
    "foo == bar" {"foo" "5", "bar" true} false

    ;; ==========
    ;; inequality
    ;; ==========
    "1 != 1" {} false
    "1 != 2" {} true
    "\"hi\" != \"hi\"" {} false
    "\"hi\" != \"bye\"" {} true
    "true != true" {} false
    "false != false" {} false
    "foo != foo" {"foo" "\"hello\""} false
    "foo != bar" {"foo" 1, "bar" 1} false
    "foo != bar" {"foo" 5, "bar" 1} true

    ;; Mixed types are never equal
    "true != 12" {} true
    "\"string\" != 4" {} true
    "5 != false" {} true
    "foo != bar" {"foo" "5", "bar" true} true

    ;; ============
    ;; greater than
    ;; ============
    ;; NB operands must be numbers
    "1 > 3" {} false
    "5 > 3" {} true
    "5 > 5" {} false
    "5 > foo" {"foo" 4} true
    "foo > 5" {"foo" 7} true

    ;; ==================
    ;; greater than equal
    ;; ==================
    ;; NB operands must be numbers
    "1 >= 3" {} false
    "5 >= 3" {} true
    "5 >= 5" {} true
    "5 >= foo" {"foo" 4} true
    "foo >= 5" {"foo" 7} true

    ;; =========
    ;; less than
    ;; =========
    ;; NB operands must be numbers
    "1 < 3" {} true
    "5 < 3" {} false
    "5 < 5" {} false
    "5 < foo" {"foo" 4} false
    "foo < 5" {"foo" 1} true

    ;; ===============
    ;; less than equal
    ;; ===============
    ;; NB operands must be numbers
    "1 <= 3" {} true
    "5 <= 3" {} false
    "5 <= 5" {} true
    "5 <= foo" {"foo" 4} false
    "foo <= 5" {"foo" 1} true

    ;; ===========
    ;; starts-with
    ;; ===========
    "\"hello world\" starts-with \"hello\"" {} true
    "\"hello world\" starts-with \"api\"" {} false
    "\"hello world\" starts-with foo" {"foo" "he"} true
    "foo starts-with \"hello\"" {"foo" "hello world"} true
    "foo starts-with \"api\"" {"foo" "hello world"} false))

(deftest comparison-expressions-require-numeric-operands
  (are [expression environment] (thrown-with-msg? Interpreter$Error #"Expected numeric value\."
                                  (-> (util/scan expression)
                                      (util/parse)
                                      (util/interpret environment)))
    "5 > true" {}
    "false > 3" {}
    "\"hi\" > \"hi\"" {}
    "false > false" {}
    "foo > 10" {"foo" "\"string\""}

    "5 >= true" {}
    "false >= 3" {}
    "\"hi\" >= \"hi\"" {}
    "false >= false" {}
    "foo >= 10" {"foo" "\"string\""}

    "5 < true" {}
    "false < 3" {}
    "\"hi\" < \"hi\"" {}
    "false < false" {}
    "foo < 10" {"foo" "\"string\""}

    "5 <= true" {}
    "false <= 3" {}
    "\"hi\" <= \"hi\"" {}
    "false <= false" {}
    "foo <= 10" {"foo" "\"string\""})

  (testing "exception highlights the error"
    (let [e (is (thrown-with-msg? Interpreter$Error #"Expected numeric value\."
                  (-> (util/scan "42 <= \"hello\"")
                      (util/parse)
                      (util/interpret {}))))]
      (is (= {:type TokenType/LESS_EQUAL
              :lexeme "<="
              :pos 3}
             (util/token->map (.-token e))))
      (is (= Interpreter$Error$Type/EXPECTED_NUMERIC_OPERAND
             (.-type e))))))

(deftest starts-with-requires-string-operands
  (are [expression environment] (thrown-with-msg? Interpreter$Error #"Expected string value\."
                                  (-> (util/scan expression)
                                      (util/parse)
                                      (util/interpret environment)))
    "\"string\" starts-with 5" {}
    "false starts-with \"string\"" {}
    "55 starts-with 5" {}
    "false starts-with false" {}
    "foo starts-with \"f\"" {"foo" 55})

  (testing "exception highlights the error"
    (let [e (is (thrown-with-msg? Interpreter$Error #"Expected string value\."
                  (-> (util/scan "42 starts-with \"hello\"")
                      (util/parse)
                      (util/interpret {}))))]
      (is (= {:type TokenType/STARTS_WITH
              :lexeme "starts-with"
              :literal "starts-with"
              :pos 3}
             (util/token->map (.-token e))))
      (is (= Interpreter$Error$Type/EXPECTED_STRING_OPERAND
             (.-type e))))))

(deftest throws-when-variable-lookup-fails
  (let [e (is (thrown-with-msg? Interpreter$Error #"Referred to a variable that is not set\."
                (-> (util/scan "foo > 5")
                    (util/parse)
                    (util/interpret {}))))]
    (is (= {:type TokenType/IDENTIFIER
            :lexeme "foo"
            :pos 0
            :literal "foo"}
           (util/token->map (.-token e))))
    (is (= Interpreter$Error$Type/UNKNOWN_VARIABLE
           (.-type e)))))

(deftest interprets-complex-expressions
  (are [expression environment expected] (= expected
                                            (-> (util/scan expression)
                                                (util/parse)
                                                (util/interpret environment)))
    "branch == \"main\" and project == \"my-project\" or always_run"
    {"branch" "main"
     "project" "my-project"
     "always_run" false}
    true

    "branch == \"main\" and project == \"my-project\" or always_run"
    {"branch" "main"
     "project" "my-project"
     "always_run" true}
    true

    "branch == \"main\" and project == \"my-project\" or always_run"
    {"branch" "develop"
     "project" "my-project"
     "always_run" true}
    true

    "branch == \"main\" and project == \"my-project\" or always_run"
    {"branch" "develop"
     "project" "my-project"
     "always_run" false}
    false

    "branch == \"main\" and not (number == 5 and project == \"my-project\")"
    {"branch" "main"
     "number" 5
     "project" "another-project"}
    true

    "branch starts-with \"main\" and not (number == 5 and project == \"my-project\")"
    {"branch" "mainly-on-the-plain"
     "number" 5
     "project" "another-project"}
    true))

(deftest undefined-variable-handling
  (are [expression environment expected] (= expected
                                            (-> (util/scan expression)
                                                (util/parse)
                                                (util/interpret environment)))
       ;; ========
       ;; equality
       ;; ========
    "1 == foo" {"foo" nil} false
    "foo == 2" {"foo" nil} false
    "\"hi\" == foo" {"foo" nil} false
    "foo == \"bye\"" {"foo" nil} false
    "foo == foo" {"foo" nil} false
    "foo == foo" {"foo" nil, "bar" "\"hello\""} false
    "foo == bar" {"foo" 1, "bar" nil} false
    "foo == bar" {"foo" nil, "bar" 1} false

    "1 != foo" {"foo" nil} false
    "foo != 2" {"foo" nil} false
    "\"hi\" != foo" {"foo" nil} false
    "foo != \"bye\"" {"foo" nil} false
    "foo != true" {"foo" nil} false
    "false != foo" {"foo" nil} false
    "foo != foo" {"foo" nil} false
    "foo != foo" {"foo" nil, "bar" "\"hello\""} false
    "foo != bar" {"foo" 1, "bar" nil} false
    "foo != bar" {"foo" nil, "bar" 5} false

       ;; ============
       ;; greater than
       ;; ============
    "1 > foo" {"foo" nil} false
    "foo > 3" {"foo" nil} false
    "foo > foo" {"foo" nil} false

       ;; ==================
       ;; greater than equal
       ;; ==================
    "1 >= foo" {"foo" nil} false
    "foo >= 3" {"foo" nil} false
    "foo >= foo" {"foo" nil} false

       ;; =========
       ;; less than
       ;; =========
    "1 < foo" {"foo" nil} false
    "foo < 3" {"foo" nil} false
    "foo < foo" {"foo" nil} false

       ;; ===============
       ;; less than equal
       ;; ===============
    "1 <= foo" {"foo" nil} false
    "foo <= 3" {"foo" nil} false
    "foo <= foo" {"foo" nil} false

       ;; ===========
       ;; starts-with
       ;; ===========
    "\"hello world\" starts-with foo" {"foo" nil} false
    "foo starts-with \"hello\"" {"foo" nil} false

       ;; ===
       ;; and
       ;; ===
    "true and foo" {"foo" nil} false
    "foo and true" {"foo" nil} false
    "12 and foo" {"foo" nil} false
    "foo and 12" {"foo" nil} false
    "\"hello world\" and foo" {"foo" nil} false
    "foo and \"hello world\"" {"foo" nil} false

    "false and foo" {"foo" nil} false
    "foo and false" {"foo" nil} false
    "foo and foo" {"foo" nil} false

       ;; ==
       ;; or
       ;; ==
    "true or foo" {"foo" nil} true
    "foo or true" {"foo" nil} true
    "12 or foo" {"foo" nil} true
    "foo or 12" {"foo" nil} true
    "\"hello world\" or foo" {"foo" nil} true
    "foo or \"hello world\"" {"foo" nil} true

    "false or foo" {"foo" nil} false
    "foo or false" {"foo" nil} false
    "foo or foo" {"foo" nil} false

       ;; ===
       ;; not
       ;; ===
    "!foo" {"foo" nil} true
       ;; not is an alias for !
    "not foo" {"foo" nil} true

       ;; complex expressions
    "branch == \"main\" and project == \"my-project\" or always_run"
    {"branch" nil
     "project" "my-project"
     "always_run" true}
    true

    "branch == \"main\" and project == \"my-project\" or always_run"
    {"branch" "main"
     "project" nil
     "always_run" true}
    true

    "branch == \"main\" and project == \"my-project\" or always_run"
    {"branch" "main"
     "project" "my-project"
     "always_run" nil}
    true

    "branch == \"main\" or project == \"my-project\" and always_run"
    {"branch" nil
     "project" "my-project"
     "always_run" true}
    true

    "branch == \"main\" or project == \"my-project\" and always_run"
    {"branch" "main"
     "project" nil
     "always_run" false}
    true

    "branch == \"main\" or project == \"my-project\" and always_run"
    {"branch" "main"
     "project" "my-project"
     "always_run" nil}
    true))
