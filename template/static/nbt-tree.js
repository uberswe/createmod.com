/* Shared lazy NBT tree renderer.
 *
 * window.NBTTree.mount(container, source) where source(params) returns a
 * Promise of the JSON API response. params: {path, depth, offset, snbt, q}.
 * The server pages children (default 200); "load more" extends the window,
 * so the DOM stays bounded without full list virtualization.
 */
(function () {
  'use strict';

  var PAGE = 200;

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text !== undefined) e.textContent = text;
    return e;
  }

  function mount(container, source) {
    container.replaceChildren();

    var toolbar = el('div', 'nbt-toolbar d-flex gap-3 align-items-center mb-2');
    var searchWrap = el('div', 'cm-searchwrap');
    searchWrap.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="10" cy="10" r="7"></circle><path d="M21 21l-6 -6"></path></svg>';
    var search = el('input', 'cm-input cm-input--search');
    search.type = 'text';
    search.placeholder = 'Search keys and values…';
    searchWrap.appendChild(search);
    var status = el('span', 'text-secondary small');
    toolbar.appendChild(searchWrap);
    toolbar.appendChild(status);
    container.appendChild(toolbar);

    var searchResults = el('div', 'nbt-search-results mb-2');
    container.appendChild(searchResults);

    var treeRoot = el('div', 'nbt-tree cm-tree');
    container.appendChild(treeRoot);

    function renderNode(node, parentEl) {
      var row = el('div', 'nbt-row');
      row.style.padding = '1px 0';
      var line = el('div', 'cm-tree__row flex-wrap');

      var toggle = el('span', 'nbt-toggle mark', node.hasChildren ? '▸' : '·');
      toggle.style.cssText = 'cursor:' + (node.hasChildren ? 'pointer' : 'default') + ';user-select:none;';
      line.appendChild(toggle);

      var name = el('span', 'nbt-name k', node.name || '(root)');
      line.appendChild(name);

      line.appendChild(el('span', 't', node.type));

      if (node.value) {
        var val = el('span', 'v', node.value);
        val.style.wordBreak = 'break-all';
        line.appendChild(val);
      }
      if (node.childCount) {
        line.appendChild(el('span', 'c', '(' + node.childCount + ')'));
      }
      if (node.create) {
        var badge = el('span', 'badge bg-orange-lt', 'create');
        line.appendChild(badge);
      }

      // hover actions: copy path, SNBT
      var actions = el('span', 'nbt-actions');
      actions.style.cssText = 'visibility:hidden;margin-left:auto;white-space:nowrap;';
      var copyBtn = el('a', 'text-secondary small', 'copy path');
      copyBtn.href = '#';
      copyBtn.addEventListener('click', function (ev) {
        ev.preventDefault();
        navigator.clipboard.writeText(node.displayPath || node.name).then(function () {
          copyBtn.textContent = 'copied!';
          setTimeout(function () { copyBtn.textContent = 'copy path'; }, 1200);
        });
      });
      actions.appendChild(copyBtn);
      if (node.hasChildren || node.type === 'compound' || node.type === 'list') {
        var snbtBtn = el('a', 'text-secondary small ms-2', 'SNBT');
        snbtBtn.href = '#';
        snbtBtn.addEventListener('click', function (ev) {
          ev.preventDefault();
          source({ path: node.path, snbt: 1 }).then(function (j) {
            showSNBT(node.displayPath || '(root)', j.snbt || j.error || '');
          });
        });
        actions.appendChild(snbtBtn);
      }
      line.appendChild(actions);
      line.addEventListener('mouseenter', function () { actions.style.visibility = 'visible'; });
      line.addEventListener('mouseleave', function () { actions.style.visibility = 'hidden'; });

      row.appendChild(line);
      var childrenEl = el('div', 'nbt-children');
      childrenEl.style.cssText = 'margin-left: 1.1em; border-left: 1px solid var(--cm-card-border-color, rgba(255,255,255,0.08)); padding-left: 0.5em; display: none;';
      row.appendChild(childrenEl);
      parentEl.appendChild(row);

      var loaded = false;
      var expanded = false;
      function expand() {
        if (!node.hasChildren) return;
        expanded = !expanded;
        toggle.textContent = expanded ? '▾' : '▸';
        childrenEl.style.display = expanded ? '' : 'none';
        if (expanded && !loaded) {
          loaded = true;
          loadChildren(node, childrenEl, 0);
        }
      }
      toggle.addEventListener('click', expand);
      name.addEventListener('dblclick', expand);
      return { row: row, childrenEl: childrenEl, expand: expand };
    }

    function loadChildren(node, childrenEl, offset) {
      var loading = el('div', 'text-secondary small', 'loading…');
      childrenEl.appendChild(loading);
      source({ path: node.path, depth: 1, offset: offset })
        .then(function (page) {
          loading.remove();
          (page.children || []).forEach(function (child) {
            renderNode(child, childrenEl);
          });
          var shown = offset + (page.children || []).length;
          if (shown < page.total) {
            var more = el('a', 'text-secondary small d-block', '… load more (' + shown + ' of ' + page.total + ')');
            more.href = '#';
            more.addEventListener('click', function (ev) {
              ev.preventDefault();
              more.remove();
              loadChildren(node, childrenEl, shown);
            });
            childrenEl.appendChild(more);
          }
        })
        .catch(function () { loading.textContent = 'failed to load'; });
    }

    function showSNBT(title, text) {
      var overlay = el('div');
      overlay.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,0.6);z-index:2000;display:flex;align-items:center;justify-content:center;padding:2rem;';
      var box = el('div', 'card');
      box.style.cssText = 'max-width:900px;max-height:80vh;width:100%;display:flex;flex-direction:column;';
      var head = el('div', 'card-header d-flex justify-content-between align-items-center');
      head.appendChild(el('span', 'fw-bold', title));
      var closeBtn = el('button', 'btn btn-sm btn-outline-secondary', 'close');
      closeBtn.addEventListener('click', function () { overlay.remove(); });
      head.appendChild(closeBtn);
      var body = el('pre', 'card-body font-monospace small');
      body.style.cssText = 'overflow:auto;white-space:pre-wrap;word-break:break-all;margin:0;';
      body.textContent = text;
      box.appendChild(head);
      box.appendChild(body);
      overlay.appendChild(box);
      overlay.addEventListener('click', function (ev) { if (ev.target === overlay) overlay.remove(); });
      document.body.appendChild(overlay);
    }

    var searchTimer = null;
    search.addEventListener('input', function () {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(function () {
        var q = search.value.trim();
        searchResults.replaceChildren();
        if (!q) { status.textContent = ''; return; }
        status.textContent = 'searching…';
        source({ q: q }).then(function (j) {
          status.textContent = (j.results || []).length + ' matches';
          (j.results || []).slice(0, 50).forEach(function (hit) {
            var row = el('div', 'cm-tree__row');
            row.style.paddingLeft = '10px';
            row.appendChild(el('span', 'k', hit.displayPath));
            if (hit.value) {
              var v = el('span', 'v', '= ' + hit.value);
              v.style.wordBreak = 'break-all';
              row.appendChild(v);
            }
            searchResults.appendChild(row);
          });
        }).catch(function () { status.textContent = 'search failed'; });
      }, 350);
    });

    // Initial load: root at depth 1
    source({ path: '', depth: 1 }).then(function (page) {
      var rootHandle = renderNode(page.node, treeRoot);
      (page.children || []).forEach(function (child) {
        renderNode(child, rootHandle.childrenEl);
      });
      rootHandle.childrenEl.style.display = '';
      var shown = (page.children || []).length;
      if (shown < page.total) {
        var more = el('a', 'text-secondary small d-block', '… load more (' + shown + ' of ' + page.total + ')');
        more.href = '#';
        more.addEventListener('click', function (ev) {
          ev.preventDefault();
          more.remove();
          loadChildren(page.node, rootHandle.childrenEl, shown);
        });
        rootHandle.childrenEl.appendChild(more);
      }
    }).catch(function (err) {
      treeRoot.textContent = 'Failed to load NBT: ' + (err && err.message ? err.message : 'unknown error');
    });
  }

  window.NBTTree = { mount: mount };
})();
