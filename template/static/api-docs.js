(function () {
  "use strict";

  if (window.CMHighlight) {
    window.CMHighlight.highlightAll(document);
  }

  // Code-sample tab switching
  var tabGroups = document.querySelectorAll(".code-card .tabs");
  for (var g = 0; g < tabGroups.length; g++) {
    var tabs = tabGroups[g];
    var card = tabs.closest(".code-card");
    if (!card) continue;
    var btns = tabs.querySelectorAll(".lang-tab");
    for (var b = 0; b < btns.length; b++) {
      (function (btn) {
        if (!btn.getAttribute("data-tab")) return;
        btn.addEventListener("click", function () {
          var target = btn.getAttribute("data-tab");
          var siblings = tabs.querySelectorAll(".lang-tab");
          for (var s = 0; s < siblings.length; s++) {
            if (siblings[s] === btn) {
              siblings[s].classList.add("active");
            } else {
              siblings[s].classList.remove("active");
            }
          }
          var pres = card.querySelectorAll("pre[data-tab-content]");
          for (var p = 0; p < pres.length; p++) {
            pres[p].hidden = (pres[p].getAttribute("data-tab-content") !== target);
          }
        });
      })(btns[b]);
    }
  }

  // Copy buttons
  var copyBtns = document.querySelectorAll(".copy-btn[data-copy]");
  for (var i = 0; i < copyBtns.length; i++) {
    (function (btn) {
      btn.addEventListener("click", function () {
        var card = btn.closest(".code-card");
        var visiblePre = card.querySelector("pre:not([hidden])");
        if (!visiblePre) return;
        var txt = visiblePre.textContent;
        var lbl = btn.querySelector("span");
        try {
          navigator.clipboard.writeText(txt);
        } catch (e) {
          var r = document.createRange();
          r.selectNodeContents(visiblePre);
          var sel = window.getSelection();
          sel.removeAllRanges();
          sel.addRange(r);
          document.execCommand("copy");
          sel.removeAllRanges();
        }
        btn.classList.add("copied");
        if (lbl) {
          var prev = lbl.textContent;
          lbl.textContent = "Copied";
          setTimeout(function () {
            btn.classList.remove("copied");
            lbl.textContent = prev;
          }, 1500);
        }
      });
    })(copyBtns[i]);
  }

  // Scroll-spy TOC
  var tocLinks = document.querySelectorAll(".apidocs-toc a.toc-link");
  var sections = [];
  for (var t = 0; t < tocLinks.length; t++) {
    var href = tocLinks[t].getAttribute("href");
    if (href) {
      var el = document.getElementById(href.slice(1));
      if (el) sections.push(el);
    }
  }

  function setActive(id) {
    for (var a = 0; a < tocLinks.length; a++) {
      if (tocLinks[a].getAttribute("href") === "#" + id) {
        tocLinks[a].classList.add("active");
      } else {
        tocLinks[a].classList.remove("active");
      }
    }
  }

  if (sections.length && "IntersectionObserver" in window) {
    var io = new IntersectionObserver(function (entries) {
      var visible = [];
      for (var e = 0; e < entries.length; e++) {
        if (entries[e].isIntersecting) visible.push(entries[e]);
      }
      visible.sort(function (a, b) {
        return a.target.getBoundingClientRect().top - b.target.getBoundingClientRect().top;
      });
      if (visible[0]) setActive(visible[0].target.id);
    }, { rootMargin: "-80px 0px -65% 0px", threshold: 0 });
    for (var s = 0; s < sections.length; s++) {
      io.observe(sections[s]);
    }
  }

  // Smooth scroll for TOC clicks
  for (var l = 0; l < tocLinks.length; l++) {
    (function (link) {
      link.addEventListener("click", function (e) {
        var id = link.getAttribute("href").slice(1);
        var el = document.getElementById(id);
        if (!el) return;
        e.preventDefault();
        window.scrollTo({ top: el.offsetTop - 76, behavior: "smooth" });
        history.replaceState(null, "", "#" + id);
      });
    })(tocLinks[l]);
  }

  // API key save
  var keyInput = document.getElementById("global-api-key");
  var saveBtn = document.getElementById("save-key");
  if (keyInput && saveBtn) {
    var saved = sessionStorage.getItem("cm-api-docs-key");
    if (saved) keyInput.value = saved;
    saveBtn.addEventListener("click", function (e) {
      e.preventDefault();
      sessionStorage.setItem("cm-api-docs-key", keyInput.value || "");
      var orig = saveBtn.textContent;
      saveBtn.textContent = "Saved!";
      setTimeout(function () { saveBtn.textContent = orig; }, 1400);
    });
  }

  // Try-it form submissions (real API calls)
  function getApiKey() {
    var el = document.getElementById("global-api-key");
    return el ? el.value.trim() : "";
  }

  function escapeHTML(s) {
    var d = document.createElement("div");
    d.appendChild(document.createTextNode(s));
    return d.innerHTML;
  }

  function showTryitResponse(respBox, status, data) {
    var isErr = status >= 400;
    var dotColor = isErr ? "var(--cm-danger)" : (status === 201 ? "var(--cm-info)" : "var(--cm-success)");
    var codeColor = isErr ? "var(--cm-danger)" : (status === 201 ? "var(--cm-info)" : "var(--cm-success)");
    var statusText = status === 200 ? "OK" : (status === 201 ? "Created" : (status === 401 ? "Unauthorized" : (status === 404 ? "Not Found" : (status === 429 ? "Too Many Requests" : "Error"))));
    var json = JSON.stringify(data, null, 2);

    // Build DOM safely
    while (respBox.firstChild) respBox.removeChild(respBox.firstChild);

    var head = document.createElement("div");
    head.className = "resp-head";

    var dot = document.createElement("span");
    dot.className = "dot";
    dot.style.background = dotColor;
    head.appendChild(dot);

    var ok = document.createElement("span");
    ok.className = "ok";
    ok.style.color = codeColor;
    ok.textContent = status + " " + statusText;
    head.appendChild(ok);

    respBox.appendChild(head);

    var pre = document.createElement("pre");
    pre.setAttribute("data-lang", "json");
    pre.textContent = json;
    respBox.appendChild(pre);

    if (window.CMHighlight) {
      window.CMHighlight.highlightAll(respBox);
    }
    respBox.classList.add("open");
  }

  var forms = document.querySelectorAll("form[data-tryit]");
  for (var f = 0; f < forms.length; f++) {
    (function (form) {
      var id = form.getAttribute("data-tryit");
      var panel = form.closest(".tryit");
      var respBox = panel.querySelector(".tryit-response");

      form.addEventListener("submit", function (e) {
        e.preventDefault();
        var sendBtn = form.querySelector(".tryit-send");
        sendBtn.disabled = true;
        var origText = sendBtn.textContent;
        sendBtn.textContent = "Sending...";

        var key = getApiKey();
        var headers = {};
        if (key) headers["X-API-Key"] = key;

        var url, opts;
        if (id === "get-schematics") {
          var query = form.querySelector("[name=query]").value;
          var page = form.querySelector("[name=page]").value || "1";
          url = "/api/schematics?page=" + encodeURIComponent(page);
          if (query) url += "&query=" + encodeURIComponent(query);
          opts = { headers: headers };
        } else if (id === "get-schematic") {
          var name = form.querySelector("[name=name]").value;
          if (!name) { sendBtn.disabled = false; sendBtn.textContent = origText; return; }
          url = "/api/schematics/" + encodeURIComponent(name);
          opts = { headers: headers };
        } else if (id === "post-upload") {
          var fd = new FormData();
          var fileInput = form.querySelector("[name=file]");
          if (fileInput && fileInput.files.length) fd.append("file", fileInput.files[0]);
          var title = form.querySelector("[name=title]");
          if (title) fd.append("title", title.value);
          var tags = form.querySelector("[name=tags]");
          if (tags && tags.value) {
            tags.value.split(",").forEach(function (t) { fd.append("tags[]", t.trim()); });
          }
          url = "/api/schematics/upload";
          opts = { method: "POST", headers: headers, body: fd };
        } else if (id === "get-schematic-stats") {
          var sname = form.querySelector("[name=name]").value;
          if (!sname) { sendBtn.disabled = false; sendBtn.textContent = origText; return; }
          url = "/api/schematics/" + encodeURIComponent(sname) + "/stats";
          opts = { headers: headers };
        } else if (id === "get-user-stats") {
          var upage = form.querySelector("[name=page]").value || "1";
          url = "/api/user/stats?page=" + encodeURIComponent(upage);
          opts = { headers: headers };
        }

        if (!url) { sendBtn.disabled = false; sendBtn.textContent = origText; return; }

        fetch(url, opts)
          .then(function (r) {
            var st = r.status;
            return r.json().then(function (d) { showTryitResponse(respBox, st, d); });
          })
          .catch(function (err) {
            showTryitResponse(respBox, 0, { error: err.message });
          })
          .finally(function () {
            sendBtn.disabled = false;
            sendBtn.textContent = origText;
          });
      });
    })(forms[f]);
  }
})();
