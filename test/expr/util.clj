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
(ns expr.util
  (:import (com.circleci.expr Expr
                              Expr$Binary
                              Expr$Grouping
                              Expr$Identifier
                              Expr$Literal
                              Expr$Logical
                              Expr$Unary
                              Expr$Visitor
                              Interpreter
                              Parser
                              Scanner
                              VariableAnalyser)))

(defn- s-expression-visitor
  []
  (reify Expr$Visitor
    (visitLogicalExpr [this ^Expr$Logical logical]
      (list (symbol (.-lexeme (.-operator logical)))
            (.accept (.-left logical) this)
            (.accept (.-right logical) this)))

    (visitBinaryExpr [this ^Expr$Binary binary]
      (list (symbol (.-lexeme (.-operator binary)))
            (.accept (.-left binary) this)
            (.accept (.-right binary) this)))

    (visitUnaryExpr [this ^Expr$Unary unary]
      (list (symbol (.-lexeme (.-operator unary)))
            (.accept (.-right unary) this)))

    (visitLiteralExpr [_this ^Expr$Literal literal]
      (list 'literal (.-value literal)))

    (visitIdentifierExpr [_this ^Expr$Identifier identifier]
      (let [ident (.-name identifier)]
        (list 'identifier (.-lexeme ident))))

    (visitGroupingExpr [this ^Expr$Grouping grouping]
      (list 'grouping (.accept (.-expression grouping) this)))))

(defn ->s-expression
  [^Expr expr]
  (.accept expr (s-expression-visitor)))

(defn parse
  [tokens]
  (.parse (Parser. tokens)))

(defn token->map
  [token]
  (merge {:type (.-type token)
          :lexeme (.-lexeme token)
          :pos (.-charPos token)}
         (when-let [literal (.-literal token)]
           {:literal literal})))

(defn scan
  [^String expression]
  (.scan (Scanner. expression)))

(defn analyse
  [^Expr expr]
  (.gatherVariables (VariableAnalyser.) expr))

(defn interpret
  [^Expr expr environment]
  (.interpret (Interpreter. environment) expr))
