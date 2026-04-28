(function(){
'use strict';

var WOOD_COLORS = {
  oak:      { plank: '#b8945f', log: '#6b5839' },
  spruce:   { plank: '#6b4226', log: '#3a2718' },
  birch:    { plank: '#d5c98c', log: '#d5cda1' },
  dark_oak: { plank: '#3e2912', log: '#382a15' },
  jungle:   { plank: '#b88764', log: '#564a2e' },
  acacia:   { plank: '#a85632', log: '#676157' },
  cherry:   { plank: '#e8c4b8', log: '#3b2022' },
  crimson:  { plank: '#6b3344', log: '#5c2133' },
  warped:   { plank: '#2b6b5e', log: '#3a3f55' }
};

var WOOL_COLORS = {
  white: '#e8e8e8', orange: '#f07613', magenta: '#bd44b3', light_blue: '#3ab3da',
  yellow: '#fed83d', lime: '#80c71f', pink: '#f38caa', gray: '#474f52',
  light_gray: '#9c9d97', cyan: '#169c9d', purple: '#8932b7', blue: '#3c44aa',
  brown: '#835432', green: '#5d7c15', red: '#b02e26', black: '#1d1c21'
};

var BG_COLOR = '#2a5a78';
var GRID_COLOR = 'rgba(140,196,232,0.15)';
var GRID_MAJOR_COLOR = 'rgba(157,208,238,0.22)';
var GRID_MAJOR_INTERVAL = 5;

function getBlockColor(type, materials) {
  if (!materials) materials = {};
  if (type === 7) {
    return WOOL_COLORS[materials.envelopeColor || materials.bladeColor || 'white'] || WOOL_COLORS.white;
  }
  if (type === 9) {
    return WOOL_COLORS[materials.bladeColor || 'white'] || WOOL_COLORS.white;
  }
  if (type === 8 && materials.frameMaterial === 'andesite_casing') {
    return '#8a8a8a';
  }
  var w = WOOD_COLORS[materials.woodType || materials.frameWoodType || 'spruce'] || WOOD_COLORS.spruce;
  if (type === 8) return w.log;
  var hex = w.plank;
  var r = parseInt(hex.slice(1,3), 16);
  var g = parseInt(hex.slice(3,5), 16);
  var b = parseInt(hex.slice(5,7), 16);
  switch (type) {
    case 2: case 3: r = Math.min(255, r+16); g = Math.min(255, g+16); b = Math.min(255, b+16); break;
    case 4: r = Math.max(0, r-16); g = Math.max(0, g-16); b = Math.max(0, b-16); break;
    case 5: r = Math.max(0, r-32); g = Math.max(0, g-32); b = Math.max(0, b-32); break;
    case 6: r = Math.max(0, r-48); g = Math.max(0, g-48); b = Math.max(0, b-48); break;
  }
  return '#' + ((1<<24)+(r<<16)+(g<<8)+b).toString(16).slice(1);
}

function getBlockLabel(type, materials) {
  if (!materials) materials = {};
  if (type === 7) {
    var envMat = materials.envelopeMaterial || 'wool';
    var envCol = materials.envelopeColor || materials.bladeColor || 'white';
    var colorName = envCol.replace(/_/g, ' ');
    if (envMat === 'envelope') return colorName + ' envelope';
    return colorName + ' wool';
  }
  if (type === 9) {
    var sailCol = materials.bladeColor || 'white';
    var sailMat = materials.bladeMaterial || 'wool';
    if (sailMat === 'sail') return sailCol.replace(/_/g, ' ') + ' sail';
    return sailCol.replace(/_/g, ' ') + ' wool';
  }
  if (type === 8) {
    if (materials.frameMaterial === 'andesite_casing') return 'andesite casing';
    var wood = materials.woodType || materials.frameWoodType || 'spruce';
    return wood.replace(/_/g, ' ') + ' log';
  }
  var wood2 = materials.woodType || materials.frameWoodType || 'spruce';
  var wn = wood2.replace(/_/g, ' ');
  switch (type) {
    case 1: return wn + ' planks';
    case 2: return wn + ' slab (bottom)';
    case 3: return wn + ' slab (top)';
    case 4: return wn + ' stairs';
    case 5: return wn + ' fence';
    case 6: return wn + ' trapdoor';
  }
  return 'block';
}

function buildLayers(data) {
  var blocks = data.blocks;
  var layers = {};
  for (var i = 0; i < blocks.length; i++) {
    var b = blocks[i];
    var y = b.y;
    if (!layers[y]) layers[y] = [];
    layers[y].push(b);
  }
  var keys = Object.keys(layers).map(Number).sort(function(a,b){return a-b;});
  return keys.map(function(y) { return { y: y, blocks: layers[y] }; });
}

function buildRadialBands(data) {
  var blocks = data.blocks;
  var cx = data.sizeX / 2;
  var cz = data.sizeZ / 2;
  var bands = {};
  for (var i = 0; i < blocks.length; i++) {
    var b = blocks[i];
    var dx = b.x - cx + 0.5;
    var dz = b.z - cz + 0.5;
    var dist = Math.floor(Math.sqrt(dx*dx + dz*dz));
    if (!bands[dist]) bands[dist] = [];
    bands[dist].push(b);
  }
  var keys = Object.keys(bands).map(Number).sort(function(a,b){return a-b;});
  return keys.map(function(d) { return { distance: d, blocks: bands[d] }; });
}

function el(tag, cls, text) {
  var e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text) e.textContent = text;
  return e;
}

function addDragToScroll(scrollContainer, canvas) {
  var dragging = false;
  var startX = 0, startY = 0, scrollLeft = 0, scrollTop = 0;

  canvas.addEventListener('mousedown', function(e) {
    if (e.button !== 0) return;
    if (scrollContainer.scrollWidth <= scrollContainer.clientWidth &&
        scrollContainer.scrollHeight <= scrollContainer.clientHeight) return;
    dragging = true;
    startX = e.clientX;
    startY = e.clientY;
    scrollLeft = scrollContainer.scrollLeft;
    scrollTop = scrollContainer.scrollTop;
    canvas.style.cursor = 'grabbing';
    e.preventDefault();
  });

  window.addEventListener('mousemove', function(e) {
    if (!dragging) return;
    scrollContainer.scrollLeft = scrollLeft - (e.clientX - startX);
    scrollContainer.scrollTop = scrollTop - (e.clientY - startY);
  });

  window.addEventListener('mouseup', function() {
    if (!dragging) return;
    dragging = false;
    canvas.style.cursor = '';
  });

  return function isDragging() { return dragging; };
}

function drawGrid(ctx, gridW, gridH, cellSize, canvasW, canvasH) {
  for (var gx = 0; gx <= gridW; gx++) {
    var isMajorX = gx % GRID_MAJOR_INTERVAL === 0;
    ctx.strokeStyle = isMajorX ? GRID_MAJOR_COLOR : GRID_COLOR;
    ctx.lineWidth = isMajorX ? 1 : 0.5;
    ctx.beginPath();
    ctx.moveTo(gx * cellSize, 0);
    ctx.lineTo(gx * cellSize, canvasH);
    ctx.stroke();
  }
  for (var gz = 0; gz <= gridH; gz++) {
    var isMajorZ = gz % GRID_MAJOR_INTERVAL === 0;
    ctx.strokeStyle = isMajorZ ? GRID_MAJOR_COLOR : GRID_COLOR;
    ctx.lineWidth = isMajorZ ? 1 : 0.5;
    ctx.beginPath();
    ctx.moveTo(0, gz * cellSize);
    ctx.lineTo(canvasW, gz * cellSize);
    ctx.stroke();
  }
}

function drawBlocks(ctx, blocks, minX, minZ, cellSize, materials, gridLookup) {
  var typeCounts = {};
  for (var j = 0; j < blocks.length; j++) {
    var b = blocks[j];
    var px = (b.x - minX) * cellSize;
    var py = (b.z - minZ) * cellSize;
    var color = getBlockColor(b.type, materials);

    if (gridLookup) gridLookup[(b.x - minX) + ',' + (b.z - minZ)] = b;

    ctx.fillStyle = color;
    var inset = Math.max(0.5, cellSize * 0.04);
    ctx.fillRect(px + inset, py + inset, cellSize - inset * 2, cellSize - inset * 2);

    if (cellSize >= 10) {
      ctx.strokeStyle = 'rgba(0,0,0,0.35)';
      ctx.lineWidth = 0.5;
      ctx.strokeRect(px + inset, py + inset, cellSize - inset * 2, cellSize - inset * 2);
    }

    if (b.type === 4 && cellSize >= 12 && b.props) {
      ctx.fillStyle = 'rgba(0,0,0,0.4)';
      var facing = b.props.facing;
      var cx2 = px + cellSize / 2;
      var cy2 = py + cellSize / 2;
      var arrowSize = cellSize * 0.25;
      ctx.beginPath();
      if (facing === 'north') { ctx.moveTo(cx2, cy2 - arrowSize); ctx.lineTo(cx2 - arrowSize, cy2 + arrowSize); ctx.lineTo(cx2 + arrowSize, cy2 + arrowSize); }
      else if (facing === 'south') { ctx.moveTo(cx2, cy2 + arrowSize); ctx.lineTo(cx2 - arrowSize, cy2 - arrowSize); ctx.lineTo(cx2 + arrowSize, cy2 - arrowSize); }
      else if (facing === 'east') { ctx.moveTo(cx2 + arrowSize, cy2); ctx.lineTo(cx2 - arrowSize, cy2 - arrowSize); ctx.lineTo(cx2 - arrowSize, cy2 + arrowSize); }
      else if (facing === 'west') { ctx.moveTo(cx2 - arrowSize, cy2); ctx.lineTo(cx2 + arrowSize, cy2 - arrowSize); ctx.lineTo(cx2 + arrowSize, cy2 + arrowSize); }
      ctx.closePath(); ctx.fill();
      if (b.props.half === 'top') {
        ctx.fillStyle = 'rgba(255,255,255,0.25)';
        ctx.beginPath(); ctx.arc(cx2, cy2, cellSize * 0.08, 0, Math.PI * 2); ctx.fill();
      }
    }

    if (b.type === 5 && cellSize >= 12) {
      ctx.fillStyle = 'rgba(255,255,255,0.2)';
      ctx.beginPath(); ctx.arc(px + cellSize/2, py + cellSize/2, cellSize * 0.15, 0, Math.PI * 2); ctx.fill();
    }

    if ((b.type === 2 || b.type === 3) && cellSize >= 14) {
      ctx.fillStyle = 'rgba(0,0,0,0.2)';
      ctx.fillRect(px + inset, py + cellSize * 0.45, cellSize - inset * 2, cellSize * 0.1);
    }

    if (!typeCounts[b.type]) typeCounts[b.type] = 0;
    typeCounts[b.type]++;
  }
  return typeCounts;
}

function drawCoordLabels(ctx, gridW, gridH, cellSize, minX, minZ, canvasH) {
  if (cellSize < 8) return;
  ctx.fillStyle = 'rgba(255,255,255,0.35)';
  ctx.font = (Math.max(8, cellSize * 0.35)) + 'px monospace';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'bottom';
  for (var lx = 0; lx < gridW; lx += Math.max(1, Math.floor(gridW / 10))) {
    ctx.fillText(String(lx + minX), (lx + 0.5) * cellSize, canvasH - 2);
  }
  ctx.textAlign = 'left';
  ctx.textBaseline = 'middle';
  for (var lz = 0; lz < gridH; lz += Math.max(1, Math.floor(gridH / 10))) {
    ctx.fillText(String(lz + minZ), 3, (lz + 0.5) * cellSize);
  }
}

function buildLegend(container, typeCounts, materials) {
  while (container.firstChild) container.removeChild(container.firstChild);
  var legendGrid = el('div', 'guide-legend-grid');
  var types = Object.keys(typeCounts).map(Number).sort(function(a,b){return a-b;});
  for (var ti = 0; ti < types.length; ti++) {
    var t = types[ti];
    var c = getBlockColor(t, materials);
    var label = getBlockLabel(t, materials);
    var item = el('div', 'guide-legend-item');
    var swatchCanvas = document.createElement('canvas');
    var swatchSize = 28;
    var swatchDpr = window.devicePixelRatio || 1;
    swatchCanvas.width = swatchSize * swatchDpr;
    swatchCanvas.height = swatchSize * swatchDpr;
    swatchCanvas.style.width = swatchSize + 'px';
    swatchCanvas.style.height = swatchSize + 'px';
    swatchCanvas.style.borderRadius = '4px';
    swatchCanvas.style.border = '1px solid rgba(255,255,255,0.15)';
    swatchCanvas.style.flexShrink = '0';
    swatchCanvas.style.imageRendering = 'pixelated';
    var sc = swatchCanvas.getContext('2d');
    sc.setTransform(swatchDpr, 0, 0, swatchDpr, 0, 0);
    sc.fillStyle = BG_COLOR;
    sc.fillRect(0, 0, swatchSize, swatchSize);
    sc.fillStyle = c;
    sc.fillRect(2, 2, swatchSize - 4, swatchSize - 4);
    sc.strokeStyle = 'rgba(0,0,0,0.3)';
    sc.lineWidth = 0.5;
    sc.strokeRect(2, 2, swatchSize - 4, swatchSize - 4);
    if (t === 4) {
      sc.fillStyle = 'rgba(0,0,0,0.4)';
      sc.beginPath();
      sc.moveTo(swatchSize/2, swatchSize*0.3);
      sc.lineTo(swatchSize*0.3, swatchSize*0.7);
      sc.lineTo(swatchSize*0.7, swatchSize*0.7);
      sc.closePath(); sc.fill();
    }
    if (t === 5) {
      sc.fillStyle = 'rgba(255,255,255,0.25)';
      sc.beginPath(); sc.arc(swatchSize/2, swatchSize/2, swatchSize*0.15, 0, Math.PI*2); sc.fill();
    }
    if (t === 2 || t === 3) {
      sc.fillStyle = 'rgba(0,0,0,0.25)';
      sc.fillRect(2, swatchSize*0.45, swatchSize-4, swatchSize*0.1);
    }
    item.appendChild(swatchCanvas);
    var labelWrap = el('div', 'guide-legend-label-wrap');
    labelWrap.appendChild(el('span', 'guide-legend-name', label));
    var countBadge = el('span', 'guide-legend-count', '×' + typeCounts[t]);
    labelWrap.appendChild(countBadge);
    item.appendChild(labelWrap);
    legendGrid.appendChild(item);
  }
  container.appendChild(legendGrid);
}

function createModal() {
  var existing = document.getElementById('guide-modal');
  if (existing) existing.remove();

  var modal = el('div', 'guide-modal');
  modal.id = 'guide-modal';

  var backdrop = el('div', 'guide-modal-backdrop');
  modal.appendChild(backdrop);

  var content = el('div', 'guide-modal-content');

  var header = el('div', 'guide-modal-header');
  var titleEl = el('h3', 'guide-modal-title', 'Step by Step Guide');
  var closeBtn = el('button', 'guide-modal-close');
  closeBtn.setAttribute('aria-label', 'Close');
  closeBtn.textContent = '×';
  header.appendChild(titleEl);
  header.appendChild(closeBtn);
  content.appendChild(header);

  var nav = el('div', 'guide-modal-nav');
  var prevBtn = el('button', 'btn btn-sm btn-outline-secondary guide-prev');
  prevBtn.textContent = '← Previous';
  prevBtn.disabled = true;
  var infoSpan = el('span', 'guide-step-info', 'Layer 1 of 1');
  var navRight = el('div', '');
  navRight.style.cssText = 'display:flex;align-items:center;gap:12px;';
  var zoomControls = el('div', 'guide-zoom-controls');
  var zoomOutBtn = el('button', '', '−');
  zoomOutBtn.title = 'Zoom out';
  var zoomLevelSpan = el('span', 'guide-zoom-level', '100%');
  var zoomInBtn = el('button', '', '+');
  zoomInBtn.title = 'Zoom in';
  zoomControls.appendChild(zoomOutBtn);
  zoomControls.appendChild(zoomLevelSpan);
  zoomControls.appendChild(zoomInBtn);
  var nextBtn = el('button', 'btn btn-sm btn-outline-secondary guide-next');
  nextBtn.textContent = 'Next →';
  navRight.appendChild(zoomControls);
  navRight.appendChild(nextBtn);
  nav.appendChild(prevBtn);
  nav.appendChild(infoSpan);
  nav.appendChild(navRight);
  content.appendChild(nav);

  var sliderWrap = el('div', 'guide-modal-slider');
  var sliderInput = document.createElement('input');
  sliderInput.type = 'range';
  sliderInput.className = 'form-range guide-slider';
  sliderInput.min = '0';
  sliderInput.max = '0';
  sliderInput.value = '0';
  sliderInput.step = '1';
  sliderWrap.appendChild(sliderInput);
  content.appendChild(sliderWrap);

  var body = el('div', 'guide-modal-body');
  body.style.position = 'relative';
  body.style.overflow = 'auto';
  body.style.maxHeight = '60vh';
  body.style.background = BG_COLOR;
  body.style.boxShadow = 'inset 0 0 60px 30px rgba(12,30,55,0.55)';
  var canvas = document.createElement('canvas');
  canvas.className = 'guide-canvas';
  canvas.style.display = 'block';
  canvas.style.margin = '0 auto';
  body.appendChild(canvas);
  content.appendChild(body);

  var legendEl = el('div', 'guide-modal-legend');
  content.appendChild(legendEl);

  var footer = el('div', 'guide-modal-footer');
  var countSpan = el('span', 'guide-block-count');
  footer.appendChild(countSpan);
  content.appendChild(footer);

  modal.appendChild(content);
  document.body.appendChild(modal);

  return {
    modal: modal,
    backdrop: backdrop,
    title: titleEl,
    closeBtn: closeBtn,
    prevBtn: prevBtn,
    nextBtn: nextBtn,
    info: infoSpan,
    slider: sliderInput,
    canvas: canvas,
    legend: legendEl,
    blockCount: countSpan,
    zoomInBtn: zoomInBtn,
    zoomOutBtn: zoomOutBtn,
    zoomLevelSpan: zoomLevelSpan
  };
}

function openGuide(data, mode) {
  if (!data || !data.blocks || data.blocks.length === 0) return;

  var materials = data.materials || {};
  var steps;
  if (mode === 'radial') {
    steps = buildRadialBands(data);
  } else {
    steps = buildLayers(data);
  }

  if (steps.length === 0) return;

  var globalMinX = Infinity, globalMaxX = -Infinity, globalMinZ = Infinity, globalMaxZ = -Infinity;
  for (var gi = 0; gi < data.blocks.length; gi++) {
    var gb = data.blocks[gi];
    if (gb.x < globalMinX) globalMinX = gb.x;
    if (gb.x > globalMaxX) globalMaxX = gb.x;
    if (gb.z < globalMinZ) globalMinZ = gb.z;
    if (gb.z > globalMaxZ) globalMaxZ = gb.z;
  }

  var ui = createModal();
  var currentStep = 0;
  var zoomLevel = 1.0;

  if (mode === 'radial') {
    ui.title.textContent = 'Center to Tip Guide';
  } else {
    ui.title.textContent = 'Layer by Layer Guide';
  }

  ui.slider.min = '0';
  ui.slider.max = String(steps.length - 1);

  var ctx = ui.canvas.getContext('2d');

  var modalGridLookup = {};
  var modalMinX = 0, modalMinZ = 0, modalCellSize = 1;

  var modalTooltip = document.createElement('div');
  modalTooltip.style.cssText = 'position:absolute;display:none;padding:4px 8px;background:rgba(0,0,0,0.85);color:#e8e8e8;font-size:12px;border-radius:4px;pointer-events:none;white-space:nowrap;z-index:10;border:1px solid rgba(191,144,69,0.4);';
  ui.canvas.parentElement.appendChild(modalTooltip);

  var isModalDragging = addDragToScroll(ui.canvas.parentElement, ui.canvas);

  function updateZoomLabel() {
    ui.zoomLevelSpan.textContent = Math.round(zoomLevel * 100) + '%';
  }

  function renderStep() {
    var step = steps[currentStep];
    var blocks = step.blocks;
    var prevBlocks = [];
    for (var si = 0; si < currentStep; si++) {
      prevBlocks = prevBlocks.concat(steps[si].blocks);
    }

    if (mode === 'radial') {
      ui.info.textContent = 'Ring ' + (currentStep + 1) + ' of ' + steps.length + ' (distance: ' + step.distance + ')';
    } else {
      ui.info.textContent = 'Layer ' + (currentStep + 1) + ' of ' + steps.length + ' (Y: ' + step.y + ', from bottom)';
    }

    ui.prevBtn.disabled = currentStep <= 0;
    ui.nextBtn.disabled = currentStep >= steps.length - 1;
    ui.slider.value = String(currentStep);

    var minX = globalMinX, maxX = globalMaxX, minZ = globalMinZ, maxZ = globalMaxZ;
    var gridW = maxX - minX + 1;
    var gridH = maxZ - minZ + 1;

    var containerEl = ui.canvas.parentElement;
    var availW = containerEl.clientWidth - 32;
    var availH = containerEl.clientHeight || 500;
    var baseCellSize = Math.max(4, Math.min(40, Math.floor(Math.min(availW / gridW, availH / gridH))));
    var cellSize = Math.max(2, Math.round(baseCellSize * zoomLevel));

    modalMinX = minX;
    modalMinZ = minZ;
    modalCellSize = cellSize;

    var canvasW = gridW * cellSize;
    var canvasH = gridH * cellSize;

    var dpr = window.devicePixelRatio || 1;
    ui.canvas.width = canvasW * dpr;
    ui.canvas.height = canvasH * dpr;
    ui.canvas.style.width = canvasW + 'px';
    ui.canvas.style.height = canvasH + 'px';
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    ctx.fillStyle = BG_COLOR;
    ctx.fillRect(0, 0, canvasW, canvasH);

    drawGrid(ctx, gridW, gridH, cellSize, canvasW, canvasH);

    if (prevBlocks.length > 0) {
      ctx.globalAlpha = 0.18;
      for (var pi = 0; pi < prevBlocks.length; pi++) {
        var pb = prevBlocks[pi];
        var ppx = (pb.x - minX) * cellSize;
        var ppy = (pb.z - minZ) * cellSize;
        var pcolor = getBlockColor(pb.type, materials);
        ctx.fillStyle = pcolor;
        var pinset = Math.max(0.5, cellSize * 0.04);
        ctx.fillRect(ppx + pinset, ppy + pinset, cellSize - pinset * 2, cellSize - pinset * 2);
      }
      ctx.globalAlpha = 1.0;
    }

    modalGridLookup = {};
    var typeCounts = drawBlocks(ctx, blocks, minX, minZ, cellSize, materials, modalGridLookup);

    drawCoordLabels(ctx, gridW, gridH, cellSize, minX, minZ, canvasH);

    while (ui.legend.firstChild) ui.legend.removeChild(ui.legend.firstChild);
    var legendGrid = el('div', 'guide-legend-grid');
    var types = Object.keys(typeCounts).map(Number).sort(function(a,b){return a-b;});
    for (var ti = 0; ti < types.length; ti++) {
      var t = types[ti];
      var c = getBlockColor(t, materials);
      var label = getBlockLabel(t, materials);
      var item = el('div', 'guide-legend-item');
      var swatch = el('span', 'guide-legend-swatch');
      swatch.style.background = c;
      item.appendChild(swatch);
      item.appendChild(el('span', 'guide-legend-label', label + ' × ' + typeCounts[t]));
      legendGrid.appendChild(item);
    }
    ui.legend.appendChild(legendGrid);

    ui.blockCount.textContent = blocks.length + ' blocks in this ' + (mode === 'radial' ? 'ring' : 'layer');
  }

  function goTo(idx) {
    currentStep = Math.max(0, Math.min(steps.length - 1, idx));
    renderStep();
  }

  function setZoom(newZoom) {
    zoomLevel = Math.max(0.25, Math.min(5, newZoom));
    updateZoomLabel();
    renderStep();
  }

  ui.zoomInBtn.addEventListener('click', function() { setZoom(zoomLevel * 1.25); });
  ui.zoomOutBtn.addEventListener('click', function() { setZoom(zoomLevel / 1.25); });

  ui.canvas.addEventListener('wheel', function(e) {
    e.preventDefault();
    var delta = e.deltaY > 0 ? 0.8 : 1.25;
    setZoom(zoomLevel * delta);
  }, { passive: false });

  ui.canvas.addEventListener('mousemove', function(e) {
    if (isModalDragging()) { modalTooltip.style.display = 'none'; return; }
    var rect = ui.canvas.getBoundingClientRect();
    var mx = (e.clientX - rect.left);
    var my = (e.clientY - rect.top);
    var gx = Math.floor(mx / modalCellSize);
    var gy = Math.floor(my / modalCellSize);
    var block = modalGridLookup[gx + ',' + gy];
    if (block) {
      modalTooltip.textContent = getBlockLabel(block.type, materials);
      modalTooltip.style.display = 'block';
      modalTooltip.style.left = (mx + 12) + 'px';
      modalTooltip.style.top = (my - 28) + 'px';
      ui.canvas.style.cursor = 'crosshair';
    } else {
      modalTooltip.style.display = 'none';
      var sc = ui.canvas.parentElement;
      ui.canvas.style.cursor = (sc.scrollWidth > sc.clientWidth || sc.scrollHeight > sc.clientHeight) ? 'grab' : 'default';
    }
  });

  ui.canvas.addEventListener('mouseleave', function() {
    modalTooltip.style.display = 'none';
    ui.canvas.style.cursor = '';
  });

  ui.prevBtn.addEventListener('click', function() { goTo(currentStep - 1); });
  ui.nextBtn.addEventListener('click', function() { goTo(currentStep + 1); });
  ui.slider.addEventListener('input', function() { goTo(parseInt(this.value)); });

  ui.closeBtn.addEventListener('click', function() { ui.modal.remove(); });
  ui.backdrop.addEventListener('click', function() { ui.modal.remove(); });

  document.addEventListener('keydown', function handler(e) {
    if (!document.getElementById('guide-modal')) {
      document.removeEventListener('keydown', handler);
      return;
    }
    if (e.key === 'Escape') { ui.modal.remove(); return; }
    if (e.key === 'ArrowLeft' || e.key === 'ArrowDown') { e.preventDefault(); goTo(currentStep - 1); }
    if (e.key === 'ArrowRight' || e.key === 'ArrowUp') { e.preventDefault(); goTo(currentStep + 1); }
  });

  updateZoomLabel();
  requestAnimationFrame(function() {
    ui.modal.classList.add('open');
    renderStep();
  });
}

function renderPage(data, mode) {
  if (!data || !data.blocks || data.blocks.length === 0) return;

  var materials = data.materials || {};
  var steps;
  if (mode === 'radial') {
    steps = buildRadialBands(data);
  } else {
    steps = buildLayers(data);
  }

  if (steps.length === 0) return;

  var globalMinX = Infinity, globalMaxX = -Infinity, globalMinZ = Infinity, globalMaxZ = -Infinity;
  for (var gi = 0; gi < data.blocks.length; gi++) {
    var gb = data.blocks[gi];
    if (gb.x < globalMinX) globalMinX = gb.x;
    if (gb.x > globalMaxX) globalMaxX = gb.x;
    if (gb.z < globalMinZ) globalMinZ = gb.z;
    if (gb.z > globalMaxZ) globalMaxZ = gb.z;
  }

  var titleEl = document.getElementById('guide-title');
  var subtitleEl = document.getElementById('guide-subtitle');
  var prevBtn = document.getElementById('guide-prev');
  var nextBtn = document.getElementById('guide-next');
  var infoEl = document.getElementById('guide-step-info');
  var slider = document.getElementById('guide-slider');
  var canvas = document.getElementById('guide-canvas');
  var legendEl = document.getElementById('guide-legend');
  var blockCountEl = document.getElementById('guide-block-count');
  var canvasWrap = document.getElementById('guide-canvas-wrap');

  if (!canvas) return;

  // Add zoom controls to nav
  var navEl = document.getElementById('guide-nav');
  var zoomLevel = 1.0;
  var zoomControls = el('div', 'guide-zoom-controls');
  var zoomOutBtn = el('button', '', '−');
  zoomOutBtn.title = 'Zoom out';
  var zoomLevelSpan = el('span', 'guide-zoom-level', '100%');
  var zoomInBtn = el('button', '', '+');
  zoomInBtn.title = 'Zoom in';
  zoomControls.appendChild(zoomOutBtn);
  zoomControls.appendChild(zoomLevelSpan);
  zoomControls.appendChild(zoomInBtn);
  if (navEl) {
    navEl.insertBefore(zoomControls, nextBtn);
  }

  if (mode === 'radial') {
    titleEl.textContent = 'Center to Tip Guide';
    subtitleEl.textContent = data.blocks.length + ' total blocks across ' + steps.length + ' rings';
  } else {
    titleEl.textContent = 'Layer by Layer Guide';
    subtitleEl.textContent = data.blocks.length + ' total blocks across ' + steps.length + ' layers';
  }

  slider.min = '0';
  slider.max = String(steps.length - 1);
  var currentStep = 0;

  var ctx = canvas.getContext('2d');

  var gridLookup = {};
  var renderMinX = 0, renderMinZ = 0, renderCellSize = 1;

  var tooltip = document.createElement('div');
  tooltip.className = 'guide-tooltip';
  tooltip.style.cssText = 'position:absolute;display:none;padding:4px 8px;background:rgba(0,0,0,0.85);color:#e8e8e8;font-size:12px;border-radius:4px;pointer-events:none;white-space:nowrap;z-index:10;border:1px solid rgba(191,144,69,0.4);';
  canvas.parentElement.appendChild(tooltip);

  var isPageDragging = addDragToScroll(canvasWrap, canvas);

  function updateZoomLabel() {
    zoomLevelSpan.textContent = Math.round(zoomLevel * 100) + '%';
  }

  function renderStep() {
    var step = steps[currentStep];
    var blocks = step.blocks;
    var prevBlocks = [];
    for (var si = 0; si < currentStep; si++) {
      prevBlocks = prevBlocks.concat(steps[si].blocks);
    }

    if (mode === 'radial') {
      infoEl.textContent = 'Ring ' + (currentStep + 1) + ' of ' + steps.length + ' (distance: ' + step.distance + ')';
    } else {
      infoEl.textContent = 'Layer ' + (currentStep + 1) + ' of ' + steps.length + ' (Y: ' + step.y + ', from bottom)';
    }

    prevBtn.disabled = currentStep <= 0;
    nextBtn.disabled = currentStep >= steps.length - 1;
    slider.value = String(currentStep);

    var minX = globalMinX, maxX = globalMaxX, minZ = globalMinZ, maxZ = globalMaxZ;
    var gridW = maxX - minX + 1;
    var gridH = maxZ - minZ + 1;

    var containerEl = canvas.parentElement;
    var availW = containerEl.clientWidth - 32;
    var availH = Math.min(600, window.innerHeight * 0.55);
    var baseCellSize = Math.max(4, Math.min(40, Math.floor(Math.min(availW / gridW, availH / gridH))));
    var cellSize = Math.max(2, Math.round(baseCellSize * zoomLevel));

    renderMinX = minX;
    renderMinZ = minZ;
    renderCellSize = cellSize;

    var canvasW = gridW * cellSize;
    var canvasH = gridH * cellSize;

    var dpr = window.devicePixelRatio || 1;
    canvas.width = canvasW * dpr;
    canvas.height = canvasH * dpr;
    canvas.style.width = canvasW + 'px';
    canvas.style.height = canvasH + 'px';
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    ctx.fillStyle = BG_COLOR;
    ctx.fillRect(0, 0, canvasW, canvasH);

    drawGrid(ctx, gridW, gridH, cellSize, canvasW, canvasH);

    if (prevBlocks.length > 0) {
      ctx.globalAlpha = 0.18;
      for (var pi = 0; pi < prevBlocks.length; pi++) {
        var pb = prevBlocks[pi];
        var ppx = (pb.x - minX) * cellSize;
        var ppy = (pb.z - minZ) * cellSize;
        var pcolor = getBlockColor(pb.type, materials);
        ctx.fillStyle = pcolor;
        var pinset = Math.max(0.5, cellSize * 0.04);
        ctx.fillRect(ppx + pinset, ppy + pinset, cellSize - pinset * 2, cellSize - pinset * 2);
      }
      ctx.globalAlpha = 1.0;
    }

    gridLookup = {};
    var typeCounts = drawBlocks(ctx, blocks, minX, minZ, cellSize, materials, gridLookup);

    drawCoordLabels(ctx, gridW, gridH, cellSize, minX, minZ, canvasH);

    buildLegend(legendEl, typeCounts, materials);

    blockCountEl.textContent = blocks.length + ' blocks in this ' + (mode === 'radial' ? 'ring' : 'layer');
  }

  function goTo(idx) {
    currentStep = Math.max(0, Math.min(steps.length - 1, idx));
    renderStep();
  }

  function setZoom(newZoom) {
    zoomLevel = Math.max(0.25, Math.min(5, newZoom));
    updateZoomLabel();
    renderStep();
  }

  zoomInBtn.addEventListener('click', function() { setZoom(zoomLevel * 1.25); });
  zoomOutBtn.addEventListener('click', function() { setZoom(zoomLevel / 1.25); });

  canvas.addEventListener('wheel', function(e) {
    e.preventDefault();
    var delta = e.deltaY > 0 ? 0.8 : 1.25;
    setZoom(zoomLevel * delta);
  }, { passive: false });

  canvas.addEventListener('mousemove', function(e) {
    if (isPageDragging()) { tooltip.style.display = 'none'; return; }
    var rect = canvas.getBoundingClientRect();
    var mx = (e.clientX - rect.left);
    var my = (e.clientY - rect.top);
    var gx = Math.floor(mx / renderCellSize);
    var gy = Math.floor(my / renderCellSize);
    var block = gridLookup[gx + ',' + gy];
    if (block) {
      tooltip.textContent = getBlockLabel(block.type, materials);
      tooltip.style.display = 'block';
      var tx = e.clientX - rect.left + 12;
      var ty = e.clientY - rect.top - 28;
      tooltip.style.left = tx + 'px';
      tooltip.style.top = ty + 'px';
      canvas.style.cursor = 'crosshair';
    } else {
      tooltip.style.display = 'none';
      canvas.style.cursor = (canvasWrap.scrollWidth > canvasWrap.clientWidth || canvasWrap.scrollHeight > canvasWrap.clientHeight) ? 'grab' : 'default';
    }
  });

  canvas.addEventListener('mouseleave', function() {
    tooltip.style.display = 'none';
    canvas.style.cursor = '';
  });

  prevBtn.addEventListener('click', function() { goTo(currentStep - 1); });
  nextBtn.addEventListener('click', function() { goTo(currentStep + 1); });
  slider.addEventListener('input', function() { goTo(parseInt(this.value)); });

  document.addEventListener('keydown', function(e) {
    if (e.key === 'ArrowLeft' || e.key === 'ArrowDown') { e.preventDefault(); goTo(currentStep - 1); }
    if (e.key === 'ArrowRight' || e.key === 'ArrowUp') { e.preventDefault(); goTo(currentStep + 1); }
  });

  updateZoomLabel();
  renderStep();
}

window.GeneratorGuide = {
  open: openGuide,
  renderPage: renderPage
};

})();
