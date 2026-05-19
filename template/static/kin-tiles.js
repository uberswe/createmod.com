(function() {
  var arrow = '<svg viewBox="0 0 24 24" width="16" height="16" class="arrow"><path d="M5 12l14 0"/><path d="M13 18l6 -6"/><path d="M13 6l6 6"/></svg>';

  function buildPixelServersSvg() {
    var svg = '<svg viewBox="0 0 300 130" width="300" height="130" class="kin-scene pix-bg" preserveAspectRatio="xMidYMid slice">';
    var stars = [[60,18],[120,32],[210,14],[260,28],[40,40],[280,52]];
    for (var i = 0; i < stars.length; i++) {
      svg += '<rect x="'+stars[i][0]+'" y="'+stars[i][1]+'" width="2" height="2" fill="#5a5a55"/>';
    }
    var ac = ['#7d7d7d','#5c5c5c'];
    for (var i = 0; i < 20; i++) {
      svg += '<rect x="'+(i*15)+'" y="108" width="15" height="15" fill="'+ac[i%2]+'" stroke="#3a3a3a" stroke-width="0.5"/>';
    }
    for (var i = 0; i < 14; i++) {
      svg += '<rect x="'+(6+i*20)+'" y="'+(113+(i%2)*5)+'" width="2" height="2" fill="#3a3a3a" opacity="0.6"/>';
    }
    var figs = [
      {x:36,skin:'#a86e4e',hair:'#3a2615',shirt:'#3a8acf',shade:'#2d6db0',pants:'#28385a'},
      {x:96,skin:'#e2c08c',hair:'#a04a2c',shirt:'#5d4a3a',shade:'#4a3a2c',pants:'#3a2e22'},
      {x:156,skin:'#7d4a2e',hair:'#1a1208',shirt:'#bf9045',shade:'#9e7735',pants:'#3a2c12'},
      {x:216,skin:'#d6a37a',hair:'#3a3a3a',shirt:'#2fb344',shade:'#1f7a2d',pants:'#1f2121'}
    ];
    var py = 64;
    for (var i = 0; i < figs.length; i++) {
      var f = figs[i], px = f.x;
      svg += '<rect x="'+px+'" y="'+py+'" width="12" height="12" fill="'+f.skin+'" stroke="'+f.hair+'" stroke-width="0.5"/>';
      svg += '<rect x="'+px+'" y="'+py+'" width="12" height="4" fill="'+f.hair+'"/>';
      svg += '<rect x="'+(px+2.5)+'" y="'+(py+5)+'" width="2.5" height="2" fill="#fff"/>';
      svg += '<rect x="'+(px+3.5)+'" y="'+(py+5)+'" width="1.5" height="2" fill="#1f2121"/>';
      svg += '<rect x="'+(px+7)+'" y="'+(py+5)+'" width="2.5" height="2" fill="#fff"/>';
      svg += '<rect x="'+(px+8)+'" y="'+(py+5)+'" width="1.5" height="2" fill="#1f2121"/>';
      svg += '<rect x="'+(px+4)+'" y="'+(py+9)+'" width="4" height="1" fill="#5c3a25"/>';
      svg += '<rect x="'+(px+1)+'" y="'+(py+12)+'" width="10" height="10" fill="'+f.shirt+'"/>';
      svg += '<rect x="'+(px+1)+'" y="'+(py+14)+'" width="10" height="2" fill="'+f.shade+'"/>';
      svg += '<rect x="'+(px-2)+'" y="'+(py+12)+'" width="3" height="10" fill="'+f.shirt+'" stroke="'+f.shade+'" stroke-width="0.3"/>';
      svg += '<rect x="'+(px+11)+'" y="'+(py+12)+'" width="3" height="10" fill="'+f.shirt+'" stroke="'+f.shade+'" stroke-width="0.3"/>';
      svg += '<rect x="'+(px-2)+'" y="'+(py+19)+'" width="3" height="3" fill="'+f.skin+'"/>';
      svg += '<rect x="'+(px+11)+'" y="'+(py+19)+'" width="3" height="3" fill="'+f.skin+'"/>';
      svg += '<rect x="'+(px+1)+'" y="'+(py+22)+'" width="4" height="8" fill="'+f.pants+'"/>';
      svg += '<rect x="'+(px+7)+'" y="'+(py+22)+'" width="4" height="8" fill="'+f.pants+'"/>';
      svg += '<rect x="'+(px-2)+'" y="'+(py+30)+'" width="16" height="2" fill="#000" opacity="0.35"/>';
    }
    svg += '</svg>';
    return svg;
  }

  function buildPixelBlocksSvg() {
    var svg = '<svg viewBox="0 0 300 130" width="300" height="130" class="kin-scene pix-bg" preserveAspectRatio="xMidYMid slice">';
    var BS = 20;
    var types = [
      ['#6db84f','#7a5a3a','#2a1d10'],
      ['#9c8a6e','#a89072','#3a2e1e'],
      ['#7d7d7d','#888888','#3a3a3a'],
      ['#bf9045','#a37a39','#3a2c12'],
      ['#d63939','#7a1f1f','#2a0c0c'],
      ['#6c6c6c','#5a5a5a','#2a2a2a'],
      ['#3e6dc2','#2d5099','#13234a'],
      ['#c4a661','#9c8244','#3a2e1c'],
      ['#5a4530','#3e2f1f','#1e1610'],
      ['#a48857','#8b7144','#3a2c1a']
    ];
    var layout = [
      [0,1,8],[1,1,0],[2,1,2],[3,1,0],[4,1,8],[5,1,2],[6,1,0],
      [0,0,3],[1,0,7],[2,0,4],[3,0,6],[4,0,5],[5,0,3],[6,0,9]
    ];
    for (var i = 0; i < layout.length; i++) {
      var t = types[layout[i][2]];
      var x = 50 + layout[i][0] * BS, y = 36 + layout[i][1] * BS;
      svg += '<rect x="'+x+'" y="'+y+'" width="'+BS+'" height="'+BS+'" fill="'+t[1]+'" stroke="'+t[2]+'" stroke-width="1"/>';
      svg += '<rect x="'+x+'" y="'+y+'" width="'+BS+'" height="'+(BS*0.3)+'" fill="'+t[0]+'"/>';
      svg += '<rect x="'+(x+3)+'" y="'+(y+8)+'" width="2" height="2" fill="'+t[2]+'" opacity="0.45"/>';
      svg += '<rect x="'+(x+10)+'" y="'+(y+12)+'" width="2" height="2" fill="'+t[0]+'" opacity="0.4"/>';
      svg += '<rect x="'+(x+14)+'" y="'+(y+5)+'" width="2" height="2" fill="'+t[2]+'" opacity="0.35"/>';
      svg += '<rect x="'+x+'" y="'+(y+BS*0.3)+'" width="'+BS+'" height="1" fill="'+t[2]+'" opacity="0.5"/>';
    }
    svg += '<g transform="translate(225,56)"><rect x="-1.5" y="-10" width="3" height="20" fill="#bf9045" opacity="0.65"/><rect x="-10" y="-1.5" width="20" height="3" fill="#bf9045" opacity="0.65"/></g>';
    svg += '</svg>';
    return svg;
  }

  function buildPixelDataSvg() {
    var BS = 22, cols = 10, rows = 4, startX = 14, startY = 12;
    var pal = [
      ['#6db84f','#7a5a3a'],['#9c8a6e','#a89072'],['#7d7d7d','#888888'],
      ['#bf9045','#a37a39'],['#d63939','#7a1f1f'],['#6c6c6c','#5a5a5a'],
      ['#3e6dc2','#2d5099'],['#c4a661','#9c8244'],['#5a4530','#3e2f1f'],
      ['#2f7a3a','#1c4d24'],['#b5d4dc','#7eaab6'],['#7f5c39','#5a3f25'],
      ['#e8d6a8','#bca771'],['#3a3a4a','#252531'],['#f3a82a','#c8841a'],
      ['#a8a8a8','#7c7c7c']
    ];
    var svg = '<svg viewBox="0 0 300 110" width="300" height="110" class="kin-scene pix-bg" preserveAspectRatio="xMidYMid slice">';
    for (var r = 0; r < rows; r++) {
      for (var c = 0; c < cols; c++) {
        var x = startX + c * BS, y = startY + r * BS;
        var p = pal[(c + r * 3) % pal.length];
        svg += '<rect x="'+(x+1)+'" y="'+(y+1)+'" width="'+(BS-3)+'" height="'+(BS-3)+'" fill="'+p[1]+'" stroke="#0c0b0a" stroke-width="0.5"/>';
        svg += '<rect x="'+(x+1)+'" y="'+(y+1)+'" width="'+(BS-3)+'" height="5" fill="'+p[0]+'"/>';
        svg += '<rect x="'+(x+4)+'" y="'+(y+9)+'" width="2" height="2" fill="#000" opacity="0.3"/>';
        svg += '<rect x="'+(x+12)+'" y="'+(y+13)+'" width="2" height="2" fill="'+p[0]+'" opacity="0.5"/>';
      }
    }
    svg += '<rect x="'+(startX+3*BS-1)+'" y="'+(startY+1*BS-1)+'" width="'+(BS+1)+'" height="'+(BS+1)+'" fill="none" stroke="#bf9045" stroke-width="1.5"/>';
    svg += '</svg>';
    return svg;
  }

  var pixelServersSvg = buildPixelServersSvg();
  var pixelBlocksSvg = buildPixelBlocksSvg();
  var pixelDataSvg = buildPixelDataSvg();

  var staticTiles = [
    '<a class="kin-tile" href="https://createmodservers.com" rel="noopener">' +
      pixelServersSvg +
      '<div class="kin-body">' +
        '<div class="kin-eyebrow">Looking for a server?</div>' +
        '<h3>Find a Create Mod server to play on.</h3>' +
        '<p>Browse player counts, mod versions, and join info from a curated list of public servers.</p>' +
        '<div class="kin-foot"><span class="url">createmodservers.com</span>' + arrow + '</div>' +
      '</div>' +
    '</a>',

    '<a class="kin-tile" href="https://createmodservers.com" rel="noopener">' +
      '<div class="kin-iconrow">' +
        '<div class="mark"><svg viewBox="0 0 24 24" width="22" height="22" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M3 4m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v2a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z"/><path d="M3 14m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v2a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z"/><path d="M7 8l0 .01"/><path d="M7 18l0 .01"/></svg></div>' +
        '<div style="flex:1">' +
          '<div class="kin-eyebrow" style="margin-bottom:4px">Servers</div>' +
          '<h3 style="margin-bottom:0">Find a Create Mod server to play on.</h3>' +
        '</div>' +
      '</div>' +
      '<div class="kin-body" style="padding-top:10px">' +
        '<p>A directory of public Create-mod servers with player counts, versions, and more.</p>' +
        '<div class="kin-foot"><span class="url">createmodservers.com</span>' + arrow + '</div>' +
      '</div>' +
    '</a>',

    '<a class="kin-tile" href="https://schematics.gg" rel="noopener">' +
      pixelBlocksSvg +
      '<div class="kin-body">' +
        '<div class="kin-eyebrow">Coming soon</div>' +
        '<h3>Not looking for Create Mod schematics?</h3>' +
        '<p>Find vanilla and modded schematics on <strong style="color:var(--cm-body-color)">schematics.gg</strong>, a broader index of community builds.</p>' +
        '<p style="margin-top:8px;font-size:11px;color:var(--cm-body-color-muted,var(--cm-text-secondary))">Supports Litematica, WorldEdit, NBT, BuildCraft and .zip world files.</p>' +
        '<div class="kin-foot"><span class="url">schematics.gg</span>' + arrow + '</div>' +
      '</div>' +
    '</a>',

    '<a class="kin-tile" href="https://schematics.gg" rel="noopener">' +
      '<div class="kin-body" style="padding-bottom:4px">' +
        '<div class="kin-eyebrow">Coming soon</div>' +
        '<h3>Not looking for Create Mod schematics?</h3>' +
        '<p>Find vanilla and modded schematics on <strong style="color:var(--cm-body-color)">schematics.gg</strong>.</p>' +
      '</div>' +
      '<div class="kin-chips">' +
        '<span class="chip brass">.litematic</span>' +
        '<span class="chip brass">.schem</span>' +
        '<span class="chip brass">.nbt</span>' +
        '<span class="chip">.bo3</span>' +
        '<span class="chip">.schematic</span>' +
        '<span class="chip">.zip</span>' +
      '</div>' +
      '<div class="kin-body" style="padding-top:10px">' +
        '<p style="font-size:11px">Supports Litematica, WorldEdit, NBT, BuildCraft and .zip world files.</p>' +
        '<div class="kin-foot"><span class="url">schematics.gg</span>' + arrow + '</div>' +
      '</div>' +
    '</a>',

    '<a class="kin-tile" href="https://schematics.gg" rel="noopener">' +
      '<div class="kin-blueprint">' +
        '<svg viewBox="0 0 200 90" width="100%" height="90" style="display:block">' +
          '<g fill="none" stroke="#d4a54e" stroke-width="1.5" stroke-linejoin="round">' +
            '<rect x="20" y="20" width="80" height="50"/>' +
            '<rect x="32" y="32" width="14" height="14"/>' +
            '<rect x="58" y="32" width="14" height="14"/>' +
            '<rect x="32" y="52" width="40" height="14"/>' +
            '<path d="M120 30 L160 18 L200 30 L200 70 L160 82 L120 70 Z"/>' +
            '<path d="M120 30 L160 42 L200 30 M160 42 L160 82"/>' +
          '</g>' +
          '<g font-family="Menlo,monospace" font-size="7" fill="#d4a54e">' +
            '<text x="20" y="14">PLAN</text>' +
            '<text x="120" y="14">ISO</text>' +
          '</g>' +
          '<g stroke="#7d5f24" stroke-width="0.5">' +
            '<line x1="20" y1="74" x2="100" y2="74"/>' +
            '<line x1="20" y1="72" x2="20" y2="76"/>' +
            '<line x1="100" y1="72" x2="100" y2="76"/>' +
          '</g>' +
          '<text x="55" y="83" font-family="Menlo,monospace" font-size="6" fill="#a87f2e">16&times;8&times;6</text>' +
        '</svg>' +
      '</div>' +
      '<div class="kin-body">' +
        '<div class="kin-eyebrow">Coming soon</div>' +
        '<h3>Not looking for Create Mod schematics?</h3>' +
        '<p>Find vanilla and modded schematics on <strong style="color:var(--cm-body-color)">schematics.gg</strong>.</p>' +
        '<p style="margin-top:8px;font-size:11px;color:var(--cm-body-color-muted,var(--cm-text-secondary))">Supports Litematica, WorldEdit, NBT, BuildCraft and .zip world files.</p>' +
        '<div class="kin-foot"><span class="url">schematics.gg</span>' + arrow + '</div>' +
      '</div>' +
    '</a>',

    '<a class="kin-tile" href="https://blocksitems.com" rel="noopener">' +
      '<div class="kin-code">' +
        'GET <span class="meth">blocksitems.com</span>/api/blocks/create:brass_block\n' +
        '{\n' +
        '  <span class="key">"id"</span>: <span class="str">"create:brass_block"</span>,\n' +
        '  <span class="key">"name"</span>: <span class="str">"Brass Block"</span>,\n' +
        '  <span class="key">"hardness"</span>: <span class="num">3</span>,\n' +
        '  <span class="key">"mod"</span>: <span class="str">"create"</span>\n' +
        '}' +
      '</div>' +
      '<div class="kin-body">' +
        '<div class="kin-eyebrow">Open Data</div>' +
        '<h3>Need block or item data for Minecraft?</h3>' +
        '<p><strong style="color:var(--cm-body-color)">BlocksItems.com</strong> is an open database with a free-to-use API.</p>' +
        '<div class="kin-foot"><span class="url">blocksitems.com</span>' + arrow + '</div>' +
      '</div>' +
    '</a>',

    '<a class="kin-tile" href="https://blocksitems.com" rel="noopener">' +
      pixelDataSvg +
      '<div class="kin-body">' +
        '<div class="kin-eyebrow">Open Data</div>' +
        '<h3>Need block or item data for Minecraft?</h3>' +
        '<p><strong style="color:var(--cm-body-color)">BlocksItems.com</strong> is an open database with a free-to-use API.</p>' +
        '<div class="kin-foot"><span class="url">blocksitems.com</span>' + arrow + '</div>' +
      '</div>' +
    '</a>',

    '<a class="kin-tile" href="https://blocksitems.com" rel="noopener">' +
      '<div class="kin-body" style="padding-bottom:2px">' +
        '<div class="kin-eyebrow">Open Data</div>' +
        '<h3>Need block or item data for Minecraft?</h3>' +
      '</div>' +
      '<div class="kin-rows">' +
        '<div class="row"><span class="swatch" style="background:linear-gradient(180deg,#e8e8e8 0 40%,#bcbcbc 40% 100%)"></span><span class="name">minecraft:white_wool</span><span class="id">1,847&times;</span></div>' +
        '<div class="row"><span class="swatch" style="background:linear-gradient(180deg,#2a2a2a 0 40%,#141414 40% 100%)"></span><span class="name">minecraft:black_wool</span><span class="id">&nbsp;&nbsp;412&times;</span></div>' +
        '<div class="row"><span class="swatch" style="background:linear-gradient(180deg,#6f4f30 0 40%,#4a341e 40% 100%)"></span><span class="name">minecraft:spruce_planks</span><span class="id">2,096&times;</span></div>' +
        '<div class="row"><span class="swatch" style="background:linear-gradient(180deg,#d1a866 0 40%,#9e7735 40% 100%)"></span><span class="name">create:brass_block</span><span class="id">&nbsp;&nbsp;&nbsp;96&times;</span></div>' +
        '<div class="row"><span class="swatch" style="background:linear-gradient(180deg,#c8704a 0 40%,#8a4f31 40% 100%)"></span><span class="name">minecraft:copper_block</span><span class="id">&nbsp;&nbsp;184&times;</span></div>' +
      '</div>' +
      '<div class="kin-body" style="padding-top:8px">' +
        '<p style="font-size:11.5px">Open database &middot; free API &middot; 100k+ blocks &amp; items.</p>' +
        '<div class="kin-foot"><span class="url">blocksitems.com</span>' + arrow + '</div>' +
      '</div>' +
    '</a>'
  ];

  window._kinTilesForPreview = staticTiles;

  function buildServerTile(servers) {
    var top = servers.filter(function(s) { return s.is_online; })
                     .slice(0, 3);
    if (top.length === 0) return null;

    var a = document.createElement('a');
    a.className = 'kin-tile';
    a.href = 'https://createmodservers.com';
    a.rel = 'noopener';

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
      var dot = document.createElement('span');
      dot.className = 'dot';
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

  var liveTile = null;
  var liveFetched = false;

  function fetchLiveServers() {
    if (liveFetched) return;
    liveFetched = true;
    fetch('/api/servers')
      .then(function(r) { return r.json(); })
      .then(function(data) {
        if (data && data.length) {
          liveTile = buildServerTile(data);
        }
      })
      .catch(function() {});
  }

  function getAllTiles() {
    var all = staticTiles.slice();
    if (liveTile) {
      all.push(liveTile);
    }
    return all;
  }

  function insertTile(container) {
    if (container.querySelector('.kin-tile')) return;
    var all = getAllTiles();
    var idx = Math.floor(Math.random() * all.length);
    var chosen = all[idx];
    var tile;
    if (typeof chosen === 'string') {
      var wrapper = document.createElement('div');
      wrapper.insertAdjacentHTML('afterbegin', chosen);
      tile = wrapper.firstChild;
    } else {
      tile = chosen.cloneNode(true);
    }
    if (tile) {
      tile.setAttribute('target', '_blank');
      tile.setAttribute('hx-boost', 'false');
      container.appendChild(tile);
    }
  }

  var railSelectors = [
    '.ad-rail',
    '.ad-rail-sm',
    '.generator-ad-rail',
    '.generator-ad-rail-sm',
    '.search-ad-rail-wide',
    '.search-ad-rail',
    '.guide-ad-rail',
    '.guide-ad-rail-sm'
  ];

  function findAdRails() {
    return document.querySelectorAll(railSelectors.join(','));
  }

  function ensureKinSlot(rail) {
    var slot = rail.querySelector('.kin-slot');
    if (slot) return slot;
    slot = document.createElement('div');
    slot.className = 'kin-slot';
    rail.appendChild(slot);
    return slot;
  }

  function fillSlots() {
    var rails = findAdRails();
    for (var i = 0; i < rails.length; i++) {
      if (rails[i].offsetParent === null) continue;
      var slot = ensureKinSlot(rails[i]);
      insertTile(slot);
    }
  }

  fetchLiveServers();
  fillSlots();

  document.addEventListener('htmx:afterSettle', function(evt) {
    if (evt.detail && evt.detail.target === document.body) {
      fillSlots();
    }
  });
})();
