(function() {
  var arrow = '<svg viewBox="0 0 24 24" width="16" height="16" class="arrow"><path d="M5 12l14 0"/><path d="M13 18l6 -6"/><path d="M13 6l6 6"/></svg>';

  function buildServerTile(servers) {
    var top = servers.filter(function(s) { return s.is_online; })
                     .slice(0, 3);
    if (top.length === 0) return null;

    var a = document.createElement('a');
    a.className = 'kin-tile';
    a.href = 'https://createmodservers.com';
    a.rel = 'noopener';
    a.target = '_blank';
    a.setAttribute('hx-boost', 'false');

    var body1 = document.createElement('div');
    body1.className = 'kin-body';
    body1.style.paddingBottom = '4px';
    var eyebrow = document.createElement('div');
    eyebrow.className = 'kin-eyebrow';
    eyebrow.textContent = 'Online right now';
    var h3 = document.createElement('h3');
    h3.textContent = 'Find a Create Mod server to play on.';
    body1.appendChild(eyebrow);
    body1.appendChild(h3);
    a.appendChild(body1);

    var list = document.createElement('div');
    list.className = 'kin-list';
    for (var i = 0; i < top.length; i++) {
      var s = top[i];
      var row = document.createElement('div');
      row.className = 'row';
      var dot = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
      dot.setAttribute('viewBox', '0 0 8 8');
      dot.setAttribute('width', '8');
      dot.setAttribute('height', '8');
      dot.style.cssText = 'flex-shrink:0';
      var circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      circle.setAttribute('cx', '4');
      circle.setAttribute('cy', '4');
      circle.setAttribute('r', '4');
      circle.setAttribute('fill', 'var(--cm-success, #2fb344)');
      dot.appendChild(circle);
      var name = document.createElement('span');
      name.className = 'name';
      name.textContent = s.name;
      var players = document.createElement('span');
      players.className = 'players';
      players.textContent = String(s.players_online) + ' online';
      row.appendChild(dot);
      row.appendChild(name);
      row.appendChild(players);
      list.appendChild(row);
    }
    a.appendChild(list);

    var body2 = document.createElement('div');
    body2.className = 'kin-body';
    body2.style.paddingTop = '6px';
    var foot = document.createElement('div');
    foot.className = 'kin-foot';
    foot.style.marginTop = '4px';
    foot.style.paddingTop = '8px';
    var urlSpan = document.createElement('span');
    urlSpan.className = 'url';
    urlSpan.textContent = 'createmodservers.com';
    foot.appendChild(urlSpan);
    foot.insertAdjacentHTML('beforeend', arrow);
    body2.appendChild(foot);
    a.appendChild(body2);

    return a;
  }

  function replaceTile(tile) {
    var slots = document.querySelectorAll('.kin-slot');
    for (var i = 0; i < slots.length; i++) {
      if (slots[i].offsetParent === null) continue;
      slots[i].innerHTML = '';
      slots[i].appendChild(tile);
      return;
    }
  }

  function fetchAndReplace() {
    fetch('/api/servers')
      .then(function(r) { return r.json(); })
      .then(function(data) {
        if (data && data.length) {
          var tile = buildServerTile(data);
          if (tile) replaceTile(tile);
        }
      })
      .catch(function() {});
  }

  function init() {
    setTimeout(fetchAndReplace, 2000);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  document.addEventListener('htmx:afterSettle', function() {
    setTimeout(fetchAndReplace, 2000);
  });
})();
