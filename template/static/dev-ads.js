// dev-ads.js — NitroPay placeholder ads for local development.
// Loaded instead of ads-2143.js when DEV=true.
// Renders labelled placeholder boxes inside ad containers so layout
// and sticky-stack behaviour can be verified without real ads.
(function () {
  'use strict';

  var BORDER = 'rgba(191,144,69,0.30)';
  var BG     = 'rgba(191,144,69,0.06)';
  var COLOR  = 'rgba(191,144,69,0.55)';

  // --------------- placeholder builder ---------------

  function makePlaceholder(w, h, lines) {
    var el = document.createElement('div');
    el.className = 'dev-ad';
    el.style.cssText =
      'width:' + w + 'px;max-width:100%;height:' + h + 'px;' +
      'background:' + BG + ';border:2px dashed ' + BORDER + ';' +
      'display:flex;align-items:center;justify-content:center;flex-direction:column;' +
      'font-family:var(--cm-font-mono,monospace);color:' + COLOR + ';' +
      'box-sizing:border-box;text-align:center;gap:2px;overflow:hidden;';

    for (var i = 0; i < lines.length; i++) {
      var span = document.createElement('span');
      span.style.fontSize = i === 0 ? '11px' : '10px';
      if (i === 0) span.style.fontWeight = '600';
      span.textContent = lines[i];
      el.appendChild(span);
    }
    return el;
  }

  // --------------- floating video-nc (picture-in-picture) ---------------

  var videoNcObservers = [];

  function createFloatingVideoNc(id) {
    var el = document.createElement('div');
    el.className = 'dev-ad dev-ad-pip';
    el.style.cssText =
      'position:fixed;bottom:16px;right:16px;z-index:9999;' +
      'width:300px;height:170px;' +
      'background:var(--cm-secondary-bg, #1f2121);' +
      'border:2px dashed ' + BORDER + ';' +
      'display:flex;align-items:center;justify-content:center;flex-direction:column;' +
      'font-family:var(--cm-font-mono,monospace);color:' + COLOR + ';' +
      'box-shadow:0 8px 32px rgba(0,0,0,0.5);border-radius:6px;' +
      'transform:translateY(calc(100% + 20px));opacity:0;' +
      'transition:transform .35s cubic-bezier(.4,0,.2,1), opacity .35s ease;';

    var close = document.createElement('button');
    close.textContent = '×';
    close.style.cssText =
      'position:absolute;top:4px;right:8px;background:none;border:none;' +
      'color:' + COLOR + ';font-size:18px;cursor:pointer;padding:2px;line-height:1;';
    close.addEventListener('click', function () {
      el.style.transform = 'translateY(calc(100% + 20px))';
      el.style.opacity = '0';
      setTimeout(function () { el.remove(); }, 350);
    });

    var label = document.createElement('span');
    label.style.cssText = 'font-size:11px;font-weight:600;';
    label.textContent = 'VIDEO AD (PIP)';
    var sub = document.createElement('span');
    sub.style.fontSize = '10px';
    sub.textContent = id;

    el.appendChild(close);
    el.appendChild(label);
    el.appendChild(sub);
    return el;
  }

  function setupVideoNcFloat(container, id) {
    if (typeof IntersectionObserver === 'undefined') return;

    var pip = null;
    var dismissed = false;

    var observer = new IntersectionObserver(function (entries) {
      if (dismissed) return;
      var entry = entries[0];
      if (!entry.isIntersecting && !pip) {
        pip = createFloatingVideoNc(id);
        document.body.appendChild(pip);
        var origClose = pip.querySelector('button');
        origClose.addEventListener('click', function () { dismissed = true; pip = null; });
        requestAnimationFrame(function () {
          requestAnimationFrame(function () {
            if (pip) {
              pip.style.transform = 'translateY(0)';
              pip.style.opacity = '1';
            }
          });
        });
      } else if (entry.isIntersecting && pip) {
        pip.style.transform = 'translateY(calc(100% + 20px))';
        pip.style.opacity = '0';
        var el = pip;
        pip = null;
        setTimeout(function () { el.remove(); }, 350);
      }
    }, { threshold: 0.1 });

    observer.observe(container);
    videoNcObservers.push(observer);
  }

  // --------------- per-format renderers ---------------

  function renderVideoNc(container, id) {
    var w = 300, h = 250;
    container.appendChild(makePlaceholder(w, h, [
      'VIDEO AD', id, w + ' × ' + h
    ]));
    setupVideoNcFloat(container, id);
  }

  function renderStickyStack(container, id) {
    var isNarrow = !!(container.closest('.ad-rail-sm') ||
                      container.closest('.generator-ad-rail-sm'));
    var w = isNarrow ? 160 : 300;
    var h = isNarrow ? 600 : 800;

    container.appendChild(makePlaceholder(w, h, [
      'STICKY AD STACK', id, w + ' × ' + h
    ]));

    if (!isNarrow) {
      renderKinSlot(container.parentElement || container);
    }
  }

  function renderKinSlot(parent) {
    var tiles = window._kinTilesForPreview;
    if (!tiles || !tiles.length || !parent) return;
    if (parent.querySelector(':scope > .kin-slot')) return;

    var slot = document.createElement('div');
    slot.className = 'kin-slot';

    var idx = Math.floor(Math.random() * tiles.length);
    var chosen = tiles[idx];
    if (typeof chosen === 'string') {
      var wrapper = document.createElement('div');
      wrapper.insertAdjacentHTML('afterbegin', chosen);
      var tile = wrapper.firstChild;
      if (tile) {
        tile.setAttribute('target', '_blank');
        tile.setAttribute('hx-boost', 'false');
        slot.appendChild(tile);
      }
    }
    parent.appendChild(slot);
  }

  function renderFloating(id) {
    var wrap = document.createElement('div');
    wrap.className = 'dev-ad dev-ad-floating';
    wrap.style.cssText =
      'position:fixed;bottom:16px;right:16px;z-index:9999;' +
      'width:300px;height:170px;' +
      'background:var(--cm-secondary-bg,' + BG + ');' +
      'border:2px dashed ' + BORDER + ';' +
      'display:flex;align-items:center;justify-content:center;flex-direction:column;' +
      'font-family:var(--cm-font-mono,monospace);color:' + COLOR + ';' +
      'box-shadow:0 4px 24px rgba(0,0,0,0.3);border-radius:4px;';

    var close = document.createElement('button');
    close.textContent = '×';
    close.style.cssText =
      'position:absolute;top:4px;right:8px;background:none;border:none;' +
      'color:' + COLOR + ';font-size:18px;cursor:pointer;padding:0;line-height:1;';
    close.addEventListener('click', function () { wrap.remove(); });

    var label = document.createElement('span');
    label.style.cssText = 'font-size:11px;font-weight:600;';
    label.textContent = 'FLOATING VIDEO AD';
    var sub = document.createElement('span');
    sub.style.fontSize = '10px';
    sub.textContent = id;

    wrap.appendChild(close);
    wrap.appendChild(label);
    wrap.appendChild(sub);
    document.body.appendChild(wrap);
  }

  function renderMobile(container, id) {
    var p = makePlaceholder(320, 100, ['MOBILE AD', id, '320 × 100']);
    p.style.margin = '0 auto';
    container.appendChild(p);
  }

  // --------------- main render dispatcher ---------------

  function renderAd(id, config) {
    if (config.mediaQuery && !window.matchMedia(config.mediaQuery).matches) {
      return;
    }

    var format = config.format || 'display';

    if (format === 'floating') {
      renderFloating(id);
      return;
    }

    var container = document.getElementById(id);
    if (!container) return;

    switch (format) {
      case 'video-nc':
        renderVideoNc(container, id);
        break;
      case 'sticky-stack':
        renderStickyStack(container, id);
        break;
      default:
        renderMobile(container, id);
    }
  }

  // --------------- intercept createAd ---------------

  function devCreateAd(id, config) {
    return new Promise(function (resolve) {
      requestAnimationFrame(function () {
        renderAd(id, config || {});
        resolve({ onNavigate: function () {} });
      });
    });
  }

  var queue = (window.nitroAds && window.nitroAds.queue) || [];
  for (var i = 0; i < queue.length; i++) {
    var item = queue[i];
    if (item[0] === 'createAd') {
      var args = item[1];
      var resolver = item[2];
      (function (a, r) {
        requestAnimationFrame(function () {
          renderAd(a[0], a[1] || {});
          if (r) r({ onNavigate: function () {} });
        });
      })(args, resolver);
    }
  }
  queue.length = 0;

  window.nitroAds.createAd = devCreateAd;

  document.addEventListener('htmx:beforeSwap', function () {
    var floats = document.querySelectorAll('.dev-ad-floating, .dev-ad-pip');
    for (var j = 0; j < floats.length; j++) {
      floats[j].remove();
    }
    for (var k = 0; k < videoNcObservers.length; k++) {
      videoNcObservers[k].disconnect();
    }
    videoNcObservers.length = 0;
  });
})();
