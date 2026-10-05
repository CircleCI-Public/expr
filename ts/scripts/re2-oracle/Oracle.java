// Reports whether re2j accepts patterns, and whether they match inputs in
// full, as the Java implementation does. See update.ts for the protocol.

import com.google.re2j.Pattern;
import com.google.re2j.PatternSyntaxException;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.Base64;

public class Oracle {
  public static void main(String[] args) throws Exception {
    var in = new BufferedReader(new InputStreamReader(System.in, StandardCharsets.UTF_8));
    var out = new StringBuilder();
    Pattern pattern = null;
    String line;
    while ((line = in.readLine()) != null) {
      var parts = line.split("\t", 2);
      var value = new String(Base64.getDecoder().decode(parts.length > 1 ? parts[1] : ""), StandardCharsets.UTF_8);
      if (parts[0].equals("P")) {
        try {
          pattern = Pattern.compile(value);
          out.append("V\t1\n");
        } catch (PatternSyntaxException e) {
          pattern = null;
          var message = Base64.getEncoder().encodeToString(e.getMessage().getBytes(StandardCharsets.UTF_8));
          out.append("V\t0\t").append(message).append("\n");
        }
      } else if (pattern == null) {
        out.append("M\t-\n");
      } else {
        out.append(pattern.matcher(value).matches() ? "M\t1\n" : "M\t0\n");
      }
    }
    System.out.print(out);
  }
}
