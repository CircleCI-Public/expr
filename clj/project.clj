(defproject com.circleci/expr (or (System/getenv "VERSION")
                              "0.1.0-SNAPSHOT")
  :description "A boolean expression evaluator for CircleCI pipeline expressions"
  :url "https://github.com/CircleCI-Public/expr"
  :license {:name "MIT License"
            :url "https://opensource.org/licenses/MIT"}
  :dependencies [[com.google.re2j/re2j "1.8"]]
  :profiles {:dev {:dependencies [[org.clojure/clojure "1.12.6"]
                                  [cheshire "6.2.0"]]
                   :aliases {"generate-corpus" ["run" "-m" "expr.generate-corpus"]}}
             ;; lein with-profile +fuzz fuzz [libFuzzer options]
             :fuzz {:dependencies [[com.code-intelligence/jazzer "0.30.0"]]
                    :java-source-paths ["fuzz"]
                    :aliases {"fuzz" ["run" "-m" "com.code_intelligence.jazzer.Jazzer"
                                      "--target_class=com.circleci.expr.ExprFuzzer"
                                      "--reproducer_path=target/fuzz-findings"
                                      "-artifact_prefix=target/fuzz-findings/"
                                      "target/fuzz-corpus"]}}
             :cljfmt {:plugins [[dev.weavejester/lein-cljfmt "0.16.5"]]
                      :cljfmt {:load-config-file? true}}}
  :deploy-repositories [["releases" {:url "https://repo.clojars.org"
                                     :username :env/clojars_username
                                     :password :env/clojars_token
                                     :sign-releases false}]]
  :resource-paths ["resources" "../domains/resources"]
  :java-source-paths ["src-java"])
