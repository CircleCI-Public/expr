// Evaluates expressions with expr's Java implementation for the differential
// fuzzer. See differential.ts for the protocol.

import com.circleci.expr.Errors;
import com.circleci.expr.Interpreter;
import com.circleci.expr.Parser;
import com.circleci.expr.Scanner;
import com.circleci.expr.Token;
import com.google.re2j.Pattern;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.io.PrintStream;
import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.HashMap;

public class Oracle {
  public static void main(String[] args) throws Exception {
    var in = new BufferedReader(new InputStreamReader(System.in, StandardCharsets.UTF_8));
    var out = new PrintStream(System.out, true, StandardCharsets.UTF_8);
    String line;
    while ((line = in.readLine()) != null) {
      String response;
      try {
        response = run(line);
      } catch (Throwable e) {
        response = "X\t" + b64(e.toString());
      }
      out.println(response);
    }
  }

  static String b64(String s) {
    return Base64.getEncoder().encodeToString(s.getBytes(StandardCharsets.UTF_8));
  }

  static String unb64(String s) {
    return new String(Base64.getDecoder().decode(s), StandardCharsets.UTF_8);
  }

  static String run(String line) {
    var fields = line.split("\t", -1);
    var expression = unb64(fields[0]);
    var env = new HashMap<String, Object>();
    for (int i = 1; i < fields.length; i++) {
      var parts = fields[i].split(",", -1);
      var name = unb64(parts[0]);
      var value = unb64(parts[2]);
      env.put(name, switch (parts[1]) {
        case "b" -> Boolean.parseBoolean(value);
        case "s" -> value;
        case "n" -> Long.parseLong(value);
        case "p" -> Pattern.compile(value);
        default -> null;
      });
    }

    try {
      var tokens = new Scanner(expression).scan();
      var expr = new Parser(tokens).parse();
      var v = new Interpreter(env).evaluate(expr);
      if (v == null) return "R\tu\t";
      if (v instanceof Boolean) return "R\tb\t" + b64(v.toString());
      if (v instanceof String s) return "R\ts\t" + b64(s);
      if (v instanceof Long) return "R\tn\t" + b64(v.toString());
      if (v instanceof Pattern p) return "R\tp\t" + b64(p.pattern());
      return "X\t" + b64("unexpected value " + v.getClass());
    } catch (Scanner.ScanError e) {
      return error("Scanner/" + e.type, "-", Character.toString(e.errorCodePoint), e.errorPos, e, expression);
    } catch (Parser.ParseError e) {
      return error("Parser/" + e.type, e.token, e, expression);
    } catch (Interpreter.Error e) {
      return error("Interpreter/" + e.type, e.token, e, expression);
    }
  }

  static String error(String type, Token token, Errors.ErrorMessage e, String expression) {
    return error(type, token.type.toString(), token.lexeme, token.charPos, e, expression);
  }

  static String error(String type, String tokenType, String lexeme, int pos, Errors.ErrorMessage e, String expression) {
    String message;
    try {
      message = e.asErrorMessage(expression);
    } catch (RuntimeException ex) {
      message = "!! asErrorMessage threw " + ex;
    }
    return String.join("\t", "E", type, tokenType, b64(lexeme), String.valueOf(pos), b64(message));
  }
}
