(function () {
  var esc = function (s) {
    return s.replace(/[&<>]/g, function (c) {
      return {"&": "&amp;", "<": "&lt;", ">": "&gt;"}[c];
    });
  };

  function highlightJSON(src) {
    var out = "";
    var i = 0;
    var n = src.length;
    while (i < n) {
      var ch = src[i];
      if (ch === " " || ch === "\t" || ch === "\n" || ch === "\r") {
        out += ch; i++; continue;
      }
      if (ch === '"') {
        var j = i + 1, escape = false;
        while (j < n) {
          var c = src[j];
          if (escape) { escape = false; j++; continue; }
          if (c === "\\") { escape = true; j++; continue; }
          if (c === '"') { j++; break; }
          j++;
        }
        var str = src.slice(i, j);
        var k = j;
        while (k < n && (src[k] === " " || src[k] === "\t")) k++;
        var isKey = src[k] === ":";
        out += '<span class="' + (isKey ? "tok-k" : "tok-s") + '">' + esc(str) + "</span>";
        i = j;
        continue;
      }
      if (/[0-9]/.test(ch) || (ch === "-" && /[0-9]/.test(src[i + 1] || ""))) {
        var m = src.slice(i).match(/^-?\d+(\.\d+)?([eE][+-]?\d+)?/);
        if (m) {
          out += '<span class="tok-n">' + esc(m[0]) + "</span>";
          i += m[0].length; continue;
        }
      }
      if (src.slice(i, i + 4) === "true" || src.slice(i, i + 4) === "null") {
        out += '<span class="tok-b">' + src.slice(i, i + 4) + "</span>";
        i += 4; continue;
      }
      if (src.slice(i, i + 5) === "false") {
        out += '<span class="tok-b">false</span>';
        i += 5; continue;
      }
      if (src.slice(i, i + 3) === "...") {
        out += '<span class="tok-c">...</span>';
        i += 3; continue;
      }
      if ("{}[]:,".indexOf(ch) !== -1) {
        out += '<span class="tok-p">' + ch + "</span>";
        i++; continue;
      }
      out += esc(ch);
      i++;
    }
    return out;
  }

  function highlightBash(src) {
    var patterns = [
      [/^#[^\n]*/, "tok-c"],
      [/^"(?:\\.|[^"\\])*"/, "tok-s"],
      [/^'(?:\\.|[^'\\])*'/, "tok-s"],
      [/^https?:\/\/[^\s"'\\)]+/, "tok-url"],
      [/^-{1,2}[A-Za-z][\w-]*/, "tok-fl"],
      [/^\b(curl|wget|export|echo|cd|cat|ls|sudo|pip|npm|yarn|go|brew|apt|set)\b/, "tok-kw"],
      [/^\\\n/, "tok-op"],
      [/^\b\d+\b/, "tok-n"]
    ];
    return tokenize(src, patterns);
  }

  var JS_KW = ["const","let","var","function","class","extends","return","if","else","for","while","do","switch","case","break","continue","new","delete","typeof","instanceof","in","of","try","catch","finally","throw","async","await","import","export","from","default","yield","this","super","void","null","true","false","undefined","static"];
  function highlightJS(src) {
    var patterns = [
      [/^\/\/[^\n]*/, "tok-c"],
      [/^\/\*[\s\S]*?\*\//, "tok-c"],
      [/^"(?:\\.|[^"\\])*"/, "tok-s"],
      [/^'(?:\\.|[^'\\])*'/, "tok-s"],
      [/^`(?:\\.|[^`\\])*`/, "tok-s"],
      [/^\b\d+(?:\.\d+)?\b/, "tok-n"],
      [/^\b[A-Za-z_$][\w$]*\b(?=\s*\()/, "tok-fn"],
      [/^\b[A-Za-z_$][\w$]*\b/, function (w) { return JS_KW.indexOf(w) !== -1 ? "tok-kw" : null; }],
      [/^[+\-*/%=&|<>!?]+/, "tok-op"]
    ];
    return tokenize(src, patterns);
  }

  var PY_KW = ["import","from","as","def","class","return","if","elif","else","for","while","in","not","and","or","is","lambda","with","try","except","finally","raise","pass","break","continue","yield","global","nonlocal","async","await","True","False","None","print"];
  var PY_BOOL = ["True","False","None"];
  function highlightPython(src) {
    var patterns = [
      [/^#[^\n]*/, "tok-c"],
      [/^"""[\s\S]*?"""/, "tok-s"],
      [/^'''[\s\S]*?'''/, "tok-s"],
      [/^"(?:\\.|[^"\\])*"/, "tok-s"],
      [/^'(?:\\.|[^'\\])*'/, "tok-s"],
      [/^\b\d+(?:\.\d+)?\b/, "tok-n"],
      [/^\b[A-Za-z_][\w]*\b(?=\s*\()/, "tok-fn"],
      [/^\b[A-Za-z_][\w]*\b/, function (w) { return PY_KW.indexOf(w) !== -1 ? (PY_BOOL.indexOf(w) !== -1 ? "tok-b" : "tok-kw") : null; }],
      [/^[+\-*/%=&|<>!]+/, "tok-op"]
    ];
    return tokenize(src, patterns);
  }

  var GO_KW = ["package","import","func","var","const","type","struct","interface","return","if","else","for","range","switch","case","default","break","continue","go","defer","chan","map","select","nil","true","false","fallthrough","goto"];
  var GO_BOOL = ["true","false","nil"];
  function highlightGo(src) {
    var patterns = [
      [/^\/\/[^\n]*/, "tok-c"],
      [/^\/\*[\s\S]*?\*\//, "tok-c"],
      [/^"(?:\\.|[^"\\])*"/, "tok-s"],
      [/^`[^`]*`/, "tok-s"],
      [/^\b\d+(?:\.\d+)?\b/, "tok-n"],
      [/^\b[A-Za-z_][\w]*\b(?=\s*\()/, "tok-fn"],
      [/^\b[A-Za-z_][\w]*\b/, function (w) { return GO_KW.indexOf(w) !== -1 ? (GO_BOOL.indexOf(w) !== -1 ? "tok-b" : "tok-kw") : null; }],
      [/^[+\-*/%=&|<>!]+/, "tok-op"]
    ];
    return tokenize(src, patterns);
  }

  function highlightHTTP(src) {
    var patterns = [
      [/^#[^\n]*/, "tok-c"],
      [/^\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\b/, "tok-kw"],
      [/^HTTP\/[\d.]+/, "tok-kw"],
      [/^\b\d{3}\b/, "tok-n"],
      [/^https?:\/\/[^\s"'\\)]+/, "tok-url"],
      [/^[A-Z][A-Za-z-]+(?=:)/, "tok-k"],
      [/^\b\d+\b/, "tok-n"]
    ];
    return tokenize(src, patterns);
  }

  function tokenize(src, patterns) {
    var out = "";
    var i = 0;
    var n = src.length;
    while (i < n) {
      var slice = src.slice(i);
      var matched = false;
      for (var p = 0; p < patterns.length; p++) {
        var rx = patterns[p][0];
        var cls = patterns[p][1];
        var m = slice.match(rx);
        if (m) {
          var tok = m[0];
          var klass = typeof cls === "function" ? cls(tok) : cls;
          if (klass) {
            out += '<span class="' + klass + '">' + esc(tok) + "</span>";
          } else {
            out += esc(tok);
          }
          i += tok.length;
          matched = true;
          break;
        }
      }
      if (!matched) {
        out += esc(src[i]);
        i++;
      }
    }
    return out;
  }

  var langs = {
    json: highlightJSON,
    js: highlightJS,
    py: highlightPython,
    go: highlightGo,
    bash: highlightBash,
    sh: highlightBash,
    http: highlightHTTP
  };

  function highlightAll(root) {
    root = root || document;
    var blocks = root.querySelectorAll("pre[data-lang]");
    for (var i = 0; i < blocks.length; i++) {
      var pre = blocks[i];
      if (pre.getAttribute("data-highlighted") === "1") continue;
      var lang = pre.getAttribute("data-lang");
      var fn = langs[lang];
      if (!fn) continue;
      // Safe: esc() sanitizes all content before insertion into pre elements
      // that contain only static code samples, not user input
      var original = pre.innerHTML;
      if (/<span class="tok-/.test(original)) {
        var parts = original.split(/(<span class="tok-[^"]+">[^<]*<\/span>)/g);
        var result = [];
        for (var p = 0; p < parts.length; p++) {
          if (parts[p].indexOf('<span class="tok-') === 0) {
            result.push(parts[p]);
          } else {
            result.push(fn(parts[p].replace(/&amp;/g, "&").replace(/&lt;/g, "<").replace(/&gt;/g, ">")));
          }
        }
        pre.innerHTML = result.join("");
      } else {
        pre.innerHTML = fn(pre.textContent);
      }
      pre.setAttribute("data-highlighted", "1");
    }
  }

  window.CMHighlight = { highlightAll: highlightAll };
})();
