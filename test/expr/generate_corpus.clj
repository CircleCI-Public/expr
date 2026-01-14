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
(ns expr.generate-corpus
  (:require [clojure.edn :as edn]
            [clojure.pprint]
            [clojure.java.io :as io]
            [cheshire.core :as json])
  (:import java.io.PushbackReader))

(defn write-test-file
  [dirs n expression environment expected]
  (let [file (apply io/file (concat ["dev-resources"]
                                    dirs
                                    [(format "%03d.json" n)]))]
    (.mkdirs (.getParentFile file))
    (spit file (str (json/generate-string {:input {:expression expression
                                                   :environment environment}
                                           :expected expected}
                                          {:pretty true})
                    \newline))))

(defn- read-test
  [rdr]
  (let [eof (Object.)
        v (edn/read {:eof eof} rdr)]
    (if (= v eof)
      nil
      v)))

(def ^:private test-corpus-data
  [{:file "test-corpus.edn"
    :dir "test-corpus"}
   {:file "evaluator-corpus.edn"
    :dir "evaluator-test-corpus"}])

(defn -main
  []
  (doseq [{:keys [file dir]} test-corpus-data]
    (try
      (with-open [r (io/reader (io/resource file))
                  rdr (PushbackReader. r)]
        (loop [data (read-test rdr)]
          (cond
            (nil? data)
            true

            :else
            (let [cases (partition-all 3 (:tests data))
                  testname (if (vector? (:name data))
                             (:name data)
                             [(:name data)])]
              (doseq [[n [expression environment expected]] (map-indexed vector cases)]
                (write-test-file (concat [dir] testname)
                                 (inc n)
                                 expression
                                 environment
                                 (if (map? expected)
                                   {:error expected}
                                   {:result expected})))
              (recur (read-test rdr))))))
      (catch Exception e
        (println (format "unable to generate test corpus from resource %s" file))
        (.printStackTrace e)
        (System/exit 1)))))
