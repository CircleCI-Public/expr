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
(ns expr.scanner-test
  (:require [clojure.test :refer (are deftest is testing)]
            [expr.util :as util])
  (:import (com.circleci.expr Scanner$ScanError
                              Scanner$ScanError$Type
                              TokenType)))

(deftest scans-keywords
  (are [expression expected] (= expected (map util/token->map (util/scan expression)))
       "and" [{:type TokenType/AND :lexeme "and" :pos 0 :literal "and"}
              {:type TokenType/EOF :lexeme "" :pos 3}]
       "AND" [{:type TokenType/AND :lexeme "AND" :pos 0 :literal "AND"}
              {:type TokenType/EOF :lexeme "" :pos 3}]

       "or" [{:type TokenType/OR :lexeme "or" :pos 0 :literal "or"}
             {:type TokenType/EOF :lexeme "" :pos 2}]
       "OR" [{:type TokenType/OR :lexeme "OR" :pos 0 :literal "OR"}
             {:type TokenType/EOF :lexeme "" :pos 2}]

       "not"[{:type TokenType/NOT :lexeme "not" :pos 0 :literal "not"} {:type TokenType/EOF :lexeme "" :pos 3}]
       "NOT" [{:type TokenType/NOT :lexeme "NOT" :pos 0 :literal "NOT"}
              {:type TokenType/EOF :lexeme "" :pos 3}]

       "true" [{:type TokenType/TRUE :lexeme "true" :pos 0 :literal "true"}
               {:type TokenType/EOF :lexeme "" :pos 4}]
       "TRUE" [{:type TokenType/TRUE :lexeme "TRUE" :pos 0 :literal "TRUE"}
               {:type TokenType/EOF :lexeme "" :pos 4}]

       "false" [{:type TokenType/FALSE :lexeme "false" :pos 0 :literal "false"}
                {:type TokenType/EOF :lexeme "" :pos 5}]
       "FALSE" [{:type TokenType/FALSE :lexeme "FALSE" :pos 0 :literal "FALSE"}
                {:type TokenType/EOF :lexeme "" :pos 5}]

       "starts-with" [{:type TokenType/STARTS_WITH :lexeme "starts-with" :pos 0 :literal "starts-with"}
                      {:type TokenType/EOF :lexeme "" :pos 11}]
       "STARTS-WITH" [{:type TokenType/STARTS_WITH :lexeme "STARTS-WITH" :pos 0 :literal "STARTS-WITH"}
                      {:type TokenType/EOF :lexeme "" :pos 11}]))

(deftest scans-operators
  (testing "well-formed operators"
    (are [expression expected] (= expected (map util/token->map (util/scan expression)))
         "()" [{:type TokenType/LEFT_PAREN :lexeme "(" :pos 0}
               {:type TokenType/RIGHT_PAREN :lexeme ")" :pos 1}
               {:type TokenType/EOF :lexeme "" :pos 2}]

         "!" [{:type TokenType/NOT :lexeme "!" :pos 0}
              {:type TokenType/EOF :lexeme "" :pos 1}]

         "!= ==" [{:type TokenType/NOT_EQUAL :lexeme "!=" :pos 0}
                  {:type TokenType/EQUAL :lexeme "==" :pos 3}
                  {:type TokenType/EOF :lexeme "" :pos 5}]

         "> >=" [{:type TokenType/GREATER :lexeme ">" :pos 0}
                 {:type TokenType/GREATER_EQUAL :lexeme ">=" :pos 2}
                 {:type TokenType/EOF :lexeme "" :pos 4}]

         "< <=" [{:type TokenType/LESS :lexeme "<" :pos 0}
                 {:type TokenType/LESS_EQUAL :lexeme "<=" :pos 2}
                 {:type TokenType/EOF :lexeme "" :pos 4}]))

  (testing "improperly-formed operators"
    (let [e (is (thrown-with-msg? Scanner$ScanError #"Incomplete token, expected \"==\"\."
                  (util/scan "=!")))]
      (is (= Scanner$ScanError$Type/INCOMPLETE_EQUALS
             (.-type e)))
      (is (= \! (.-errorChar e)))
      (is (= 1 (.-errorPos e))))))

(deftest scans-strings
  (testing "well-formed strings"
    (are [expression expected] (= expected (map util/token->map (util/scan expression)))
         "\"\"" [{:type TokenType/STRING :lexeme "\"\"" :pos 0 :literal ""}
                 {:type TokenType/EOF :lexeme "" :pos 2}]

         "\"a string\"" [{:type TokenType/STRING :lexeme "\"a string\"" :pos 0 :literal "a string"}
                         {:type TokenType/EOF :lexeme "" :pos 10}]

         "\"an \\\"escaped\\\" string\"" [{:type TokenType/STRING :lexeme "\"an \\\"escaped\\\" string\"" :pos 0 :literal "an \"escaped\" string"}
                                          {:type TokenType/EOF :lexeme "" :pos 23}]

         "\"backslash \\\\escapes\"" [{:type TokenType/STRING :lexeme "\"backslash \\\\escapes\"" :pos 0 :literal "backslash \\escapes"}
                                      {:type TokenType/EOF :lexeme "" :pos 21}]))

  (testing "unterminated string"
    (let [e (is (thrown-with-msg? Scanner$ScanError #"Unterminated string\."
                  (util/scan "text \"an unterminated string")))]
      (is (= Scanner$ScanError$Type/UNTERMINATED_STRING
             (.-type e)))
      (is (= \" (.-errorChar e)))
      (is (= 5 (.-errorPos e))))))

(deftest scans-digits
  (doseq [i (range 200)]
    (is (= [{:type TokenType/NUMBER :lexeme (str i) :pos 0 :literal i}
            {:type TokenType/EOF :lexeme "" :pos (count (str i))}]
           (map util/token->map (util/scan (str i)))))))

(deftest scans-identifiers
  (are [expression expected] (= expected (map util/token->map (util/scan expression)))
    "foo" [{:type TokenType/IDENTIFIER :lexeme "foo" :pos 0 :literal "foo"}
           {:type TokenType/EOF :lexeme "" :pos 3}]
    "foo.bar" [{:type TokenType/IDENTIFIER :lexeme "foo.bar" :pos 0 :literal "foo.bar"}
               {:type TokenType/EOF :lexeme "" :pos 7}]
    "foo-bar" [{:type TokenType/IDENTIFIER :lexeme "foo-bar" :pos 0 :literal "foo-bar"}
               {:type TokenType/EOF :lexeme "" :pos 7}]

    "foo_bar" [{:type TokenType/IDENTIFIER :lexeme "foo_bar" :pos 0 :literal "foo_bar"}
               {:type TokenType/EOF :lexeme "" :pos 7}])

  (testing "identifiers cannot contain strings of '.'s"
    (let [e (is (thrown-with-msg? Scanner$ScanError #"Unexpected character\."
                                  (util/scan "foo..bar")))]
      (is (= Scanner$ScanError$Type/UNEXPECTED_CHARACTER
             (.-type e)))
      (is (= \. (.-errorChar e)))
      (is (= 3 (.-errorPos e)))))

  (testing "identifiers cannot contain start with '.'"
    (let [e (is (thrown-with-msg? Scanner$ScanError #"Unexpected character\."
                                  (util/scan ".foo")))]
      (is (= Scanner$ScanError$Type/UNEXPECTED_CHARACTER
             (.-type e)))
      (is (= \. (.-errorChar e)))
      (is (= 0 (.-errorPos e)))))

  (testing "identifiers cannot contain start with '-'"
    (let [e (is (thrown-with-msg? Scanner$ScanError #"Unexpected character\."
                                  (util/scan "-foo")))]
      (is (= Scanner$ScanError$Type/UNEXPECTED_CHARACTER
             (.-type e)))
      (is (= \- (.-errorChar e)))
      (is (= 0 (.-errorPos e)))))

  (testing "identifiers cannot contain start with '_'"
    (let [e (is (thrown-with-msg? Scanner$ScanError #"Unexpected character\."
                                  (util/scan "_foo")))]
      (is (= Scanner$ScanError$Type/UNEXPECTED_CHARACTER
             (.-type e)))
      (is (= \_ (.-errorChar e)))
      (is (= 0 (.-errorPos e))))))

(deftest throws-for-unexpected-characters
  (doseq [c "&*^%$£?/#~:;@'"]
    (let [e (is (thrown-with-msg? Scanner$ScanError #"Unexpected character\."
                                  (util/scan (str c))))]
      (is (= Scanner$ScanError$Type/UNEXPECTED_CHARACTER
             (.-type e)))
      (is (= c (.-errorChar e)))
      (is (= 0 (.-errorPos e))))))
