(defproject expr-parent "0.0.1-SNAPSHOT"
  :plugins [[lein-modules "0.3.11"]
            [lein-parent "0.3.9"]]

  :pedantic? :abort
  :min-lein-version "2.8.0"
  :modules {:dirs ["clj"]
            :subprocess nil})
