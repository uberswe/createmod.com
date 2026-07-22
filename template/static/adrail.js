// Desktop right-rail ads.
//
// Each page declares its rail with a container element:
//   <div class="cm-side-rail d-none d-xl-block" data-cm-adrail
//        data-prefix="mods" data-kw="minecraft,..." data-page="mods"></div>
// (search / mod_detail use .search-ad-rail-wide; the data-cm-adrail attribute
// is what matters.)
//
// The rail is a single NitroPay sticky-stack unit (id "<prefix>_sticky") that
// spans the full column height (see the [id$="_sticky"] rule in app.css):
// NitroPay places ads down the column that pin near the top of the viewport as
// the user scrolls past them, and stickyStackResizable lets it add extra units
// when the viewport is tall enough. This replaced an earlier A/B setup (video
// on top + either a sticky-stack or two fixed display ads) — video now runs
// site-wide through the floating outstream player in foot.html, which only
// appears when there is an ad to play, so the rail carries display demand only.
(function () {
  "use strict";

  function slot(id) {
    var d = document.createElement("div");
    d.id = id;
    d.className = "mb-3";
    return d;
  }

  function nitro() {
    return window.nitroAds;
  }

  // Builds one page's rail. Exposed for completeness; normally invoked by the
  // initializer below via the data attributes.
  window.cmAdRail = function (rail, prefix, keywords, pageType) {
    if (!rail || !nitro() || !nitro().createAd || !prefix) return;

    rail.appendChild(slot(prefix + "_sticky"));
    nitro().createAd(prefix + "_sticky", {
      format: "sticky-stack",
      stickyStackLimit: 15,
      stickyStackSpace: 2,
      stickyStackOffset: 8,
      stickyStackResizable: true,
      refreshLimit: 30,
      refreshTime: 30,
      refreshVisibleOnly: true,
      renderVisibleOnly: true,
      visibleMargin: 300,
      onNavigateMin: 4000,
      keywords: keywords || "",
      targeting: { pageType: pageType || prefix },
      report: { enabled: true, icon: true, wording: "Report Ad", position: "top-right" },
      mediaQuery: "(min-width: 1200px)"
    });
  };

  function initAdRails() {
    var rails = document.querySelectorAll("[data-cm-adrail]");
    for (var i = 0; i < rails.length; i++) {
      var el = rails[i];
      if (el.getAttribute("data-cm-adrail-done")) continue;
      el.setAttribute("data-cm-adrail-done", "1");
      window.cmAdRail(
        el,
        el.getAttribute("data-prefix"),
        el.getAttribute("data-kw"),
        el.getAttribute("data-page")
      );
    }
  }

  // ---- Blocked-ads fallback ----
  // Most blockers stop the NitroPay loader at the network level, which would
  // leave the rail as a blank gutter. Instead of a popup or blocking content,
  // fill it with a quiet first-party support note. Signals: the loader's
  // onerror flag (_nitroBlocked, set in head.html), NitroPay's own abp flag,
  // and the np.detect event dispatched by the detection script in foot.html.
  function renderSupportNote(rail) {
    if (rail.querySelector(".cm-support-note")) return;
    // The empty unit holder spans the full column; hide it so the note
    // isn't pushed below the fold.
    var holders = rail.querySelectorAll('[id$="_sticky"]');
    for (var i = 0; i < holders.length; i++) holders[i].style.display = "none";

    var note = document.createElement("div");
    note.className = "cm-support-note";
    var head = document.createElement("strong");
    head.textContent = "Enjoying CreateMod?";
    var body = document.createElement("p");
    body.textContent =
      "CreateMod.com is a free community project funded by the ads that " +
      "would normally appear here. If the site is useful to you, please " +
      "consider allowlisting createmod.com in your ad blocker.";
    note.appendChild(head);
    note.appendChild(body);
    rail.appendChild(note);
  }

  function blockedFallback() {
    var rails = document.querySelectorAll("[data-cm-adrail]");
    for (var i = 0; i < rails.length; i++) renderSupportNote(rails[i]);
  }

  function checkBlocked() {
    if (window._nitroBlocked || (window.nitroAds && window.nitroAds.abp === true)) {
      blockedFallback();
    }
  }

  document.addEventListener("np.detect", function (e) {
    if (e && e.detail && e.detail.blocking) blockedFallback();
  });
  setTimeout(checkBlocked, 6000);
  document.addEventListener("htmx:afterSettle", function () {
    setTimeout(checkBlocked, 1500);
  });

  if (document.readyState !== "loading") {
    initAdRails();
  }
  document.addEventListener("DOMContentLoaded", initAdRails);
  // hx-boost swaps the body on navigation; rebuild rails for the new page.
  document.addEventListener("htmx:afterSettle", initAdRails);
})();
