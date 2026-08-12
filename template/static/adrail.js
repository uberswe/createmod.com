// Desktop right-rail ads.
//
// Each page declares its rail with a container element:
//   <div class="cm-side-rail d-none d-xl-block" data-cm-adrail
//        data-prefix="mods" data-kw="minecraft,..." data-page="mods"></div>
// (search / mod_detail use .search-ad-rail-wide; the data-cm-adrail attribute
// is what matters.)
//
// The rail carries a NitroPay sticky-stack unit (id "<prefix>_sticky") that
// spans the full column height (see the [id$="_sticky"] rule in app.css):
// NitroPay places ads down the column that pin near the top of the viewport as
// the user scrolls past them, and stickyStackResizable lets it add extra units
// when the viewport is tall enough.
//
// Video normally runs site-wide through the floating outstream player in
// foot.html. But on smaller laptop screens (1200-1440px) the viewport is too
// short for a 300px rail plus a bottom-right corner video, so the floating
// player overlaps the rail. In that band we instead run the video inline at the
// TOP of the rail (id "<prefix>_railvideo"), above the sticky stack, and
// suppress the floating player (its mediaQuery excludes this band). NitroPay's
// mediaQuery gates the in-rail unit, so the slot stays empty at other widths.
(function () {
  "use strict";

  // Smaller-laptop band: rail present (>=1200) but viewport too short for a
  // clean floating corner video. Kept in sync with the outstream player's
  // exclusion mediaQuery in foot.html.
  var LAPTOP_RAIL_VIDEO_MQ = "(min-width: 1200px) and (max-width: 1439.98px)";

  function slot(id, cls) {
    var d = document.createElement("div");
    d.id = id;
    d.className = cls === undefined ? "mb-3" : cls;
    return d;
  }

  function nitro() {
    return window.nitroAds;
  }

  // Builds one page's rail. Exposed for completeness; normally invoked by the
  // initializer below via the data attributes.
  window.cmAdRail = function (rail, prefix, keywords, pageType) {
    if (!rail || !nitro() || !nitro().createAd || !prefix) return;

    // In-rail video on top (smaller-laptop band only; empty slot elsewhere).
    // No default margin: spacing below it is added via CSS only in-band, so the
    // empty slot adds no top gap to the rail at other widths.
    rail.appendChild(slot(prefix + "_railvideo", ""));
    nitro().createAd(prefix + "_railvideo", {
      format: "video-nc",
      renderVisibleOnly: true,
      onNavigateMin: 4000,
      keywords: keywords || "",
      targeting: { pageType: pageType || prefix },
      report: { enabled: true, icon: true, wording: "Report Ad", position: "top-right" },
      mediaQuery: LAPTOP_RAIL_VIDEO_MQ
    });

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
