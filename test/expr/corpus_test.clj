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

(defn run-test
  [expression environment]
  (try
    {"result" (-> (util/scan expression)
                  (util/parse)
                  (util/interpret environment))}
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

(deftest run-test-corpus
  (doseq [{:strs [input expected]
           :keys [test-name]} (test-files "dev-resources/test-corpus")
          :let [{:strs [expression environment]} input]]
    (is (= expected
           (run-test expression environment))
        test-name)))
