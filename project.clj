(defproject circleci/expr (or (System/getenv "VERSION")
                              "0.1.0-SNAPSHOT")
  :dependencies [[com.google.re2j/re2j "1.8"]]
  :profiles {:dev {:dependencies [[org.clojure/clojure "1.12.6"]
                                  [cheshire "6.2.0"]]
                   :aliases {"generate-corpus" ["run" "-m" "expr.generate-corpus"]}}
             :cljfmt {:plugins [[dev.weavejester/lein-cljfmt "0.16.5"]]
                      :cljfmt {:load-config-file? true}}
             :deploy {:plugins [[circle/s3-wagon-private "1.2.2" :exclusions [commons-codec]]]
                      :repositories [["circle-s3"
                                      {:url "s3p://circle-jars/releases"
                                       :sign-releases false
                                       :username [:gpg :env/circle_jars_username]
                                       :passphrase [:gpg :env/circle_jars_password]
                                       :snapshots false}]]}}
  :resource-paths ["resources" "go/domains/resources"]
  :java-source-paths ["src-java"])
