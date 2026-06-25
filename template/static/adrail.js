// Desktop right-rail ads + A/B test.
//
// Each page declares its rail with a container element:
//   <div class="ad-rail d-none d-xl-block" data-cm-adrail
//        data-prefix="mods" data-kw="minecraft,..." data-page="mods"></div>
// (search / mod_detail use .search-ad-rail-wide; the data-cm-adrail attribute
// is what matters.)
//
// On each page view we randomly (~50/50) pick a variant:
//   Variant A  -> video ad on top + ONE NitroPay sticky-stack ad.
//   Variant B  -> video ad on top + TWO static display ads (300x600 / 300x250
//                 / 160x600) that stick together as one unit.
// The ad-unit ids encode the variant (e.g. mods_a_sticky, mods_b_display_1) so
// NitroPay reporting can compare revenue/RPM between the two setups.
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

  // Builds one page's rail and runs the A/B test. Exposed for completeness;
  // normally invoked by the initializer below via the data attributes.
  window.cmAdRail = function (rail, prefix, keywords, pageType) {
    if (!rail || !nitro() || !nitro().createAd || !prefix) return;

    var common = {
      keywords: keywords || "",
      targeting: { pageType: pageType || prefix },
      report: { enabled: true, icon: true, wording: "Report Ad", position: "top-right" },
      mediaQuery: "(min-width: 1200px)"
    };
    var refresh = {
      refreshLimit: 10, refreshTime: 45,
      refreshVisibleOnly: true, renderVisibleOnly: true, visibleMargin: 300
    };

    // Video ad on top in both variants — scrolls away with the page.
    rail.appendChild(slot(prefix + "_video"));
    nitro().createAd(prefix + "_video", Object.assign({ format: "video-nc" }, common));

    if (Math.random() < 0.5) {
      // Variant A: a single NitroPay sticky-stack ad.
      rail.appendChild(slot(prefix + "_a_sticky"));
      nitro().createAd(prefix + "_a_sticky", Object.assign({
        format: "sticky-stack",
        stickyStackLimit: 15,
        stickyStackSpace: 2.5,
        stickyStackOffset: 8,
        stickyStackResizable: false
      }, refresh, common));
    } else {
      // Variant B: two static display ads stacked in one sticky wrapper, so
      // they pin together while scrolling and never overlap.
      var wrap = document.createElement("div");
      wrap.className = "ad-sticky-stack";
      wrap.appendChild(slot(prefix + "_b_display_1"));
      wrap.appendChild(slot(prefix + "_b_display_2"));
      rail.appendChild(wrap);
      var display = Object.assign({
        sizes: [["300", "600"], ["300", "250"], ["160", "600"]]
      }, refresh, common);
      nitro().createAd(prefix + "_b_display_1", display);
      nitro().createAd(prefix + "_b_display_2", display);
    }
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
