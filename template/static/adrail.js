// Desktop right-rail ads.
//
// Each page declares its rail with a container element:
//   <div class="ad-rail d-none d-xl-block" data-cm-adrail
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

  if (document.readyState !== "loading") {
    initAdRails();
  }
  document.addEventListener("DOMContentLoaded", initAdRails);
  // hx-boost swaps the body on navigation; rebuild rails for the new page.
  document.addEventListener("htmx:afterSettle", initAdRails);
})();
