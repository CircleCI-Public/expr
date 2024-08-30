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
(ns expr.analyser-test
  (:require [clojure.test :refer (are deftest)]
            [expr.util :as util]))

(defn- analyse
  [expression]
  (->> (util/scan expression)
       (util/parse)
       (util/analyse)))

(deftest gathers-variables
  (are [expression expected] (= expected (map (fn [t] (.-lexeme t))
                                              (analyse expression)))
       ;; literals
       "1" []
       "true" []
       "\"a string\"" []
       "foo" ["foo"]

       ;; binary expressions
       "foo and bar" ["foo" "bar"]
       "foo or bar" ["foo" "bar"]
       "1 == foo" ["foo"]
       "foo != 1" ["foo"]
       "foo > bar >= baz" ["foo" "bar" "baz"]

       ;; unary
       "not foo" ["foo"]
       "!bar" ["bar"]

       ;; grouping
       "true and (false or foo)" ["foo"]
       "true and (false and not foo)" ["foo"]))
