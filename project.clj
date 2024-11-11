(defproject circleci/expr (or (System/getenv "VERSION")
                              "0.1.0-SNAPSHOT")
  :profiles {:dev {:dependencies [[org.clojure/clojure "1.12.0"]
                                  [cheshire "5.13.0"]]}
             :cljfmt {:plugins [[dev.weavejester/lein-cljfmt "0.13.0"]]
                      :cljfmt {:load-config-file? true}}
             :deploy {:plugins [[circle/s3-wagon-private "1.2.2" :exclusions [commons-codec]]]
                      :repositories [["circle-s3"
                                      {:url "s3p://circle-jars/releases"
                                       :sign-releases false
                                       :username [:gpg :env/circle_jars_username]
                                       :passphrase [:gpg :env/circle_jars_password]
                                       :snapshots false}]]}}
  :java-source-paths ["src-java"])
