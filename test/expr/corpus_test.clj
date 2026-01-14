(ns expr.corpus-test
  (:require [clojure.test :refer (deftest is)]
            [clojure.java.io :as io]
            [clojure.string :as string]
            [cheshire.core :as json]
            [expr.util :as util])
  (:import (com.circleci.expr Interpreter$Error
                              Parser$ParseError
                              Scanner$ScanError)))

(defn- test-files
  [dir]
  (->> (file-seq (io/file dir))
       (filter #(.isFile %))
       (remove (fn [f]
                 (string/starts-with? (.getName f) ".")))
       (map (fn [f]
              (assoc (json/parse-string (slurp f))
                     :test-name (.getPath f))))))

(defn- run-test
  [process expression environment]
  (try
    {"result" (process expression environment)}
    (catch Scanner$ScanError e
      {"error" {"errorType" (str "Scanner/" (.-type e))
                "lexeme" (str (.-errorChar e))
                "charPos" (.-errorPos e)}})
    (catch Parser$ParseError e
      (let [token (.-token e)]
        {"error" {"errorType" (str "Parser/" (.-type e))
                  "tokenType" (str (.-type token))
                  "lexeme" (.-lexeme token)
                  "charPos" (.-charPos token)}}))
    (catch Interpreter$Error e
      (let [token (.-token e)]
        {"error" {"errorType" (str "Interpreter/" (.-type e))
                  "tokenType" (str (.-type token))
                  "lexeme" (.-lexeme token)
                  "charPos" (.-charPos token)}}))))

(deftest run-interpreter-test-corpus
  (let [run-interpreter-test (partial run-test (fn [expression environment]
                                                 (-> (util/scan expression)
                                                     (util/parse)
                                                     (util/interpret environment))))]
    (doseq [{:strs [input expected]
             :keys [test-name]} (test-files "dev-resources/test-corpus")
            :let [{:strs [expression environment]} input]]
      (is (= expected
             (run-interpreter-test expression environment))
          (format "%s: %s" test-name expression)))))

(deftest run-evaluator-test-corpus
  (let [run-evaluator-test (partial run-test (fn [expression environment]
                                               (-> (util/scan expression)
                                                   (util/parse)
                                                   (util/evaluate environment))))]
    (doseq [{:strs [input expected]
             :keys [test-name]} (test-files "dev-resources/evaluator-test-corpus")
            :let [{:strs [expression environment]} input]]
      (is (= expected
             (run-evaluator-test expression environment))
          (format "%s: %s" test-name expression)))))
