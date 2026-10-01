(defproject circleci/expr (or (System/getenv "VERSION")
                              "0.1.0-SNAPSHOT")
  :description "A boolean expression evaluator for CircleCI pipeline expressions"
  :url "https://github.com/CircleCI-Public/expr"
  :license {:name "MIT License"
            :url "https://opensource.org/licenses/MIT"}
  :dependencies [[com.google.re2j/re2j "1.8"]]
  :profiles {:dev {:dependencies [[org.clojure/clojure "1.12.6"]
                                  [cheshire "6.2.0"]]
                   :aliases {"generate-corpus" ["run" "-m" "expr.generate-corpus"]}}
             :cljfmt {:plugins [[dev.weavejester/lein-cljfmt "0.16.5"]]
                      :cljfmt {:load-config-file? true}}}
  :deploy-repositories [["releases" {:url "https://repo.clojars.org"
                                     :username :env/clojars_username
                                     :password :env/clojars_token
                                     :sign-releases false}]]
  :resource-paths ["resources" "../domains/resources"]
  :java-source-paths ["src-java"])
