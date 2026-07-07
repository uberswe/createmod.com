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
      var menu = root.querySelector('.dl-split-menu');
      var toggle = root.querySelector('.dl-split-toggle');
      if (menu && !menu.hidden) {
        menu.hidden = true;
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

    function open() {
      closeAll(root);
      menu.hidden = false;
      toggle.setAttribute('aria-expanded', 'true');
      var first = menu.querySelector('[role="menuitem"]:not([aria-disabled="true"])');
      if (first) first.focus();
    }
    function close(refocus) {
      menu.hidden = true;
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
    if (!ev.target.closest('[data-dl-split]')) closeAll(null);
  });
  document.addEventListener('keydown', function (ev) {
    if (ev.key === 'Escape') closeAll(null);
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initAll);
  } else {
    initAll();
  }
  // Re-init after HTMX swaps (hx-boost full-page navigations)
  document.addEventListener('htmx:afterSwap', initAll);
})();
