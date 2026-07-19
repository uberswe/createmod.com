/* Shared split-button download control.
 *
 * Markup contract (see template/include/download_split.html):
 *   <div class="dl-split" data-dl-split>
 *     <a class="btn dl-split-primary" href="...">Download</a>
 *     <button class="btn dl-split-toggle" aria-haspopup="menu" aria-expanded="false">▾</button>
 *     <div class="dl-split-menu" role="menu" hidden>
 *       <a role="menuitem" href="..." data-lossy="..." ...>.schem</a>
 *       ...
 *     </div>
 *   </div>
 *
 * Behavior: primary link downloads immediately; the arrow toggles the menu;
 * Escape/click-outside close it; arrow keys move between items. Multiple
 * instances per page are supported. Idempotent under hx-boost re-swaps.
 */
(function () {
  'use strict';

  function closeAll(except) {
    document.querySelectorAll('[data-dl-split]').forEach(function (root) {
      if (root === except) return;
      var menu = root._dlSplitMenu || root.querySelector('.dl-split-menu');
      var toggle = root.querySelector('.dl-split-toggle');
      if (menu && !menu.hidden) {
        menu.hidden = true;
        root.appendChild(menu); // restore from the body portal
        if (toggle) toggle.setAttribute('aria-expanded', 'false');
      }
    });
  }

  function initOne(root) {
    if (root._dlSplitInit) return;
    root._dlSplitInit = true;
    var toggle = root.querySelector('.dl-split-toggle');
    var menu = root.querySelector('.dl-split-menu');
    if (!toggle || !menu) return;
    root._dlSplitMenu = menu;

    function open() {
      closeAll(root);
      // Portal the menu to <body> while open: fixed positioning alone is
      // not enough, because ancestors with backdrop-filter or transform
      // (e.g. the generator controls overlay) become the containing block
      // for fixed descendants and their overflow clips the menu.
      document.body.appendChild(menu);
      menu.hidden = false;
      // Position with fixed viewport coordinates so ancestors with
      // overflow:hidden (cards, description containers) cannot clip the
      // menu. Falls back upward when there is no room below.
      var r = toggle.getBoundingClientRect();
      var mw = menu.offsetWidth;
      var mh = menu.offsetHeight;
      var left = Math.max(8, Math.min(r.right - mw, window.innerWidth - mw - 8));
      var top = r.bottom + 4;
      if (top + mh > window.innerHeight - 8) {
        top = Math.max(8, r.top - mh - 4);
      }
      menu.style.position = 'fixed';
      menu.style.left = left + 'px';
      menu.style.top = top + 'px';
      menu.style.right = 'auto';
      toggle.setAttribute('aria-expanded', 'true');
      var first = menu.querySelector('[role="menuitem"]:not([aria-disabled="true"])');
      if (first) { try { first.focus({ preventScroll: true }); } catch (e) { first.focus(); } }
    }
    function close(refocus) {
      menu.hidden = true;
      root.appendChild(menu); // restore from the body portal
      toggle.setAttribute('aria-expanded', 'false');
      if (refocus) toggle.focus();
    }

    toggle.addEventListener('click', function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      if (menu.hidden) open();
      else close(false);
    });

    menu.addEventListener('keydown', function (ev) {
      var items = Array.prototype.filter.call(
        menu.querySelectorAll('[role="menuitem"]'),
        function (el) { return el.getAttribute('aria-disabled') !== 'true'; }
      );
      var idx = items.indexOf(document.activeElement);
      if (ev.key === 'ArrowDown') {
        ev.preventDefault();
        items[(idx + 1) % items.length].focus();
      } else if (ev.key === 'ArrowUp') {
        ev.preventDefault();
        items[(idx - 1 + items.length) % items.length].focus();
      } else if (ev.key === 'Escape') {
        ev.preventDefault();
        close(true);
      } else if (ev.key === 'Tab') {
        close(false);
      }
    });

    menu.addEventListener('click', function (ev) {
      var item = ev.target.closest('[role="menuitem"]');
      if (!item) return;
      if (item.getAttribute('aria-disabled') === 'true') {
        ev.preventDefault();
        return;
      }
      close(false);
    });
  }

  function initAll() {
    document.querySelectorAll('[data-dl-split]').forEach(initOne);
  }

  document.addEventListener('click', function (ev) {
    // The open menu lives in <body> (portal), so check it separately.
    if (!ev.target.closest('[data-dl-split]') && !ev.target.closest('.dl-split-menu')) closeAll(null);
  });
  document.addEventListener('keydown', function (ev) {
    if (ev.key === 'Escape') closeAll(null);
  });
  // Fixed-position menus don't follow their anchor; close instead of drifting.
  window.addEventListener('scroll', function () { closeAll(null); }, true);
  window.addEventListener('resize', function () { closeAll(null); });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initAll);
  } else {
    initAll();
  }
  // Re-init after HTMX swaps (hx-boost full-page navigations)
  document.addEventListener('htmx:afterSwap', initAll);
})();
