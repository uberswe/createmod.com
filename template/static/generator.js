(function(){
'use strict';

if (window._generatorInit) return;
window._generatorInit = true;

var THREE;
var scene, camera, renderer, controls;
var blockMeshes = [];
var container;
var animId;
var currentData = null;
var cameraUserInteracted = false;

var WOOD_COLORS = {
  oak:      { plank: 0xb8945f, log: 0x6b5839 },
  spruce:   { plank: 0x6b4226, log: 0x3a2718 },
  birch:    { plank: 0xd5c98c, log: 0xd5cda1 },
  dark_oak: { plank: 0x3e2912, log: 0x382a15 },
  jungle:   { plank: 0xb88764, log: 0x564a2e },
  acacia:   { plank: 0xa85632, log: 0x676157 },
  cherry:   { plank: 0xe8c4b8, log: 0x3b2022 },
  crimson:  { plank: 0x6b3344, log: 0x5c2133 },
  warped:   { plank: 0x2b6b5e, log: 0x3a3f55 }
};

var WOOL_COLORS = {
  white:      0xe8e8e8,
  orange:     0xf07613,
  magenta:    0xbd44b3,
  light_blue: 0x3ab3da,
  yellow:     0xfed83d,
  lime:       0x80c71f,
  pink:       0xf38caa,
  gray:       0x474f52,
  light_gray: 0x9c9d97,
  cyan:       0x169c9d,
  purple:     0x8932b7,
  blue:       0x3c44aa,
  brown:      0x835432,
  green:      0x5d7c15,
  red:        0xb02e26,
  black:      0x1d1c21
};

var ANDESITE_CASING_COLOR = 0x8a8a8a;

function getWoodColor(woodType, blockType) {
  var w = WOOD_COLORS[woodType] || WOOD_COLORS.spruce;
  if (blockType === 8) return w.log;
  return w.plank;
}

function getBlockColor(blockType, materials) {
  if (!materials) materials = {};
  var mat = materials;

  if (blockType === 7) {
    var color = mat.envelopeColor || mat.bladeColor || 'white';
    return WOOL_COLORS[color] || WOOL_COLORS.white;
  }
  if (blockType === 9) {
    var sailColor = mat.bladeColor || 'white';
    return WOOL_COLORS[sailColor] || WOOL_COLORS.white;
  }
  if (mat.envelopeMaterial && (blockType === 2 || blockType === 3 || blockType === 4)) {
    return WOOL_COLORS[mat.envelopeColor || 'white'] || WOOL_COLORS.white;
  }
  if (blockType === 8 && mat.frameMaterial === 'andesite_casing') {
    return ANDESITE_CASING_COLOR;
  }
  var woodType = mat.woodType || mat.frameWoodType || 'spruce';
  return getWoodColor(woodType, blockType);
}

function initScene(containerId) {
  container = document.getElementById(containerId);
  if (!container || !window.THREE) return false;
  THREE = window.THREE;

  scene = new THREE.Scene();
  scene.background = new THREE.Color(0x0f1628);
  scene.fog = new THREE.FogExp2(0x0f1628, 0.003);

  camera = new THREE.PerspectiveCamera(40, container.clientWidth / container.clientHeight, 0.1, 2000);
  camera.position.set(30, 20, 30);

  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false });
  renderer.setSize(container.clientWidth, container.clientHeight);
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
  while (container.firstChild) container.removeChild(container.firstChild);
  container.appendChild(renderer.domElement);

  var ambient = new THREE.AmbientLight(0xffeedd, 0.7);
  scene.add(ambient);
  var key = new THREE.DirectionalLight(0xfff5e0, 1.2);
  key.position.set(50, 80, 40);
  scene.add(key);
  var fill = new THREE.DirectionalLight(0x88aaff, 0.6);
  fill.position.set(-30, 20, -30);
  scene.add(fill);
  var rim = new THREE.DirectionalLight(0xffaa88, 0.4);
  rim.position.set(-20, 40, 60);
  scene.add(rim);

  controls = new OrbitControls(camera, renderer.domElement);
  controls.enableDamping = true;
  controls.dampingFactor = 0.08;
  controls.target.set(0, 0, 0);

  var grid = new THREE.GridHelper(200, 100, 0x1d3456, 0x14253f);
  grid.position.y = -0.5;
  scene.add(grid);

  animate();

  window.addEventListener('resize', onResize);

  if (!window._genCleanupInit) {
    window._genCleanupInit = true;
    document.addEventListener('htmx:beforeSwap', function() {
      if (animId) { cancelAnimationFrame(animId); animId = null; }
      clearBlocks();
      if (renderer) { renderer.dispose(); renderer = null; }
      scene = null; camera = null; controls = null; container = null;
    });
  }

  return true;
}

function onResize() {
  if (!container || !camera || !renderer) return;
  camera.aspect = container.clientWidth / container.clientHeight;
  camera.updateProjectionMatrix();
  renderer.setSize(container.clientWidth, container.clientHeight);
}

function animate() {
  animId = requestAnimationFrame(animate);
  if (controls) controls.update();
  if (renderer && scene && camera) renderer.render(scene, camera);
}

function disposeObject(obj) {
  if (obj.geometry) obj.geometry.dispose();
  if (obj.material) obj.material.dispose();
  if (obj.children) {
    for (var i = obj.children.length - 1; i >= 0; i--) {
      disposeObject(obj.children[i]);
    }
  }
}

function clearBlocks() {
  for (var i = 0; i < blockMeshes.length; i++) {
    scene.remove(blockMeshes[i]);
    disposeObject(blockMeshes[i]);
  }
  blockMeshes = [];
}

function renderBlocks(data) {
  if (!scene || !THREE) return;
  clearBlocks();
  currentData = data;

  var blocks = data.blocks;
  if (!blocks || blocks.length === 0) return;

  var materials = data.materials || {};

  var groups = {};
  for (var i = 0; i < blocks.length; i++) {
    var b = blocks[i];
    var t = b.type || 1;
    if (!groups[t]) groups[t] = [];
    groups[t].push(b);
  }

  var cx = data.sizeX / 2;
  var cy = 0;
  var cz = data.sizeZ / 2;

  var boxGeo = new THREE.BoxGeometry(0.95, 0.95, 0.95);
  var slabGeo = new THREE.BoxGeometry(0.95, 0.47, 0.95);
  var trapdoorGeo = new THREE.BoxGeometry(0.9, 0.18, 0.9);
  var sailGeo = new THREE.BoxGeometry(0.95, 0.32, 0.95);

  var dummy = new THREE.Object3D();

  function makeStairGeo() {
    var bottom = new THREE.BoxGeometry(0.95, 0.47, 0.95);
    bottom.translate(0, -0.24, 0);
    var top = new THREE.BoxGeometry(0.95, 0.47, 0.475);
    top.translate(0, 0.24, -0.2375);
    var merged = new THREE.BufferGeometry();
    var posA = bottom.getAttribute('position').array;
    var posB = top.getAttribute('position').array;
    var norA = bottom.getAttribute('normal').array;
    var norB = top.getAttribute('normal').array;
    var idxA = bottom.index.array;
    var idxB = top.index.array;
    var vCount = posA.length / 3;
    var pos = new Float32Array(posA.length + posB.length);
    var nor = new Float32Array(norA.length + norB.length);
    pos.set(posA); pos.set(posB, posA.length);
    nor.set(norA); nor.set(norB, norA.length);
    var idx = new Uint16Array(idxA.length + idxB.length);
    idx.set(idxA);
    for (var si = 0; si < idxB.length; si++) idx[idxA.length + si] = idxB[si] + vCount;
    merged.setAttribute('position', new THREE.BufferAttribute(pos, 3));
    merged.setAttribute('normal', new THREE.BufferAttribute(nor, 3));
    merged.setIndex(new THREE.BufferAttribute(idx, 1));
    bottom.dispose(); top.dispose();
    return merged;
  }

  var FACING_ROT = { south: 0, west: -Math.PI / 2, north: Math.PI, east: Math.PI / 2 };

  var fencePostGeo = new THREE.BoxGeometry(0.25, 0.95, 0.25);
  var fenceBarGeo = new THREE.BoxGeometry(0.125, 0.19, 0.35);

  function buildFenceMesh(block, mat, cx, cy, cz) {
    var group = new THREE.Group();
    var post = new THREE.Mesh(fencePostGeo, mat);
    group.add(post);
    var props = block.props || {};
    var dirs = [
      { key: 'north', dx: 0, dz: -1 },
      { key: 'south', dx: 0, dz: 1 },
      { key: 'east',  dx: 1, dz: 0 },
      { key: 'west',  dx: -1, dz: 0 }
    ];
    for (var di = 0; di < dirs.length; di++) {
      var d = dirs[di];
      if (props[d.key] !== 'true') continue;
      var offX = d.dx * 0.35;
      var offZ = d.dz * 0.35;
      var barW = d.dx !== 0 ? 0.35 : 0.125;
      var barD = d.dz !== 0 ? 0.35 : 0.125;
      var barGeoH = new THREE.BoxGeometry(barW, 0.125, barD);
      var barHi = new THREE.Mesh(barGeoH, mat);
      barHi.position.set(offX, 0.18, offZ);
      group.add(barHi);
      var barGeoL = new THREE.BoxGeometry(barW, 0.125, barD);
      var barLo = new THREE.Mesh(barGeoL, mat);
      barLo.position.set(offX, -0.18, offZ);
      group.add(barLo);
    }
    group.position.set(block.x - cx, (block.y - cy), block.z - cz);
    return group;
  }

  for (var type in groups) {
    var arr = groups[type];
    var t = parseInt(type);
    var color = getBlockColor(t, materials);
    var roughness = 0.75;
    var metalness = 0.08;
    if (t === 7) { roughness = 0.92; metalness = 0.0; }
    else if (t === 8) { roughness = 0.8; metalness = 0.05; }
    else if (t === 9) { roughness = 0.95; metalness = 0.0; }
    if (t === 8 && materials.frameMaterial === 'andesite_casing') {
      roughness = 0.6; metalness = 0.3;
    }
    var mat = new THREE.MeshStandardMaterial({
      color: color,
      roughness: roughness,
      metalness: metalness
    });

    if (t === 4) {
      var stairGroups = {};
      for (var si = 0; si < arr.length; si++) {
        var sb = arr[si];
        var facing = (sb.props && sb.props.facing) || 'south';
        var half = (sb.props && sb.props.half) || 'bottom';
        var key = facing + '_' + half;
        if (!stairGroups[key]) stairGroups[key] = [];
        stairGroups[key].push(sb);
      }
      for (var sk in stairGroups) {
        var sArr = stairGroups[sk];
        var parts = sk.split('_');
        var sFacing = parts[0];
        var sHalf = parts[1];
        var sGeo = makeStairGeo();
        var sMesh = new THREE.InstancedMesh(sGeo, mat, sArr.length);
        for (var sj = 0; sj < sArr.length; sj++) {
          dummy.position.set(sArr[sj].x - cx, (sArr[sj].y - cy), sArr[sj].z - cz);
          dummy.rotation.set(0, 0, 0);
          dummy.scale.set(1, 1, 1);
          dummy.rotation.y = FACING_ROT[sFacing] || 0;
          if (sHalf === 'top') dummy.scale.y = -1;
          dummy.updateMatrix();
          sMesh.setMatrixAt(sj, dummy.matrix);
        }
        sMesh.instanceMatrix.needsUpdate = true;
        scene.add(sMesh);
        blockMeshes.push(sMesh);
      }
      dummy.rotation.set(0, 0, 0);
      dummy.scale.set(1, 1, 1);
      continue;
    }

    if (t === 5) {
      for (var fi = 0; fi < arr.length; fi++) {
        var fMesh = buildFenceMesh(arr[fi], mat, cx, cy, cz);
        scene.add(fMesh);
        blockMeshes.push(fMesh);
      }
      continue;
    }

    var geo;
    if (t === 2 || t === 3) {
      geo = slabGeo;
    } else if (t === 6) {
      geo = trapdoorGeo;
    } else if (t === 9) {
      geo = sailGeo;
    } else {
      geo = boxGeo;
    }

    var mesh = new THREE.InstancedMesh(geo, mat, arr.length);
    for (var j = 0; j < arr.length; j++) {
      var yOff = 0;
      if (t === 2) yOff = -0.24;
      if (t === 3) yOff = 0.24;
      dummy.position.set(arr[j].x - cx, (arr[j].y - cy) + yOff, arr[j].z - cz);
      dummy.updateMatrix();
      mesh.setMatrixAt(j, dummy.matrix);
    }
    mesh.instanceMatrix.needsUpdate = true;
    scene.add(mesh);
    blockMeshes.push(mesh);
  }

  if (!cameraUserInteracted) {
    var maxDim = Math.max(data.sizeX, data.sizeY, data.sizeZ);
    var dist = maxDim * 1.2;
    var centerY = data.sizeY / 2;
    camera.position.set(dist * 0.7, centerY + dist * 0.4, dist * 0.7);
    controls.target.set(0, centerY, 0);
    controls.update();
  }
}

function OrbitControls(cam, domElement) {
  this.camera = cam;
  this.domElement = domElement;
  this.target = new THREE.Vector3();
  this.enableDamping = true;
  this.dampingFactor = 0.08;

  var scope = this;
  var spherical = new THREE.Spherical();
  var sphericalDelta = new THREE.Spherical();
  var panOffset = new THREE.Vector3();
  var rotateStart = new THREE.Vector2();
  var panStart = new THREE.Vector2();
  var state = 0;
  var scale = 1;

  function getDistance() {
    return scope.camera.position.clone().sub(scope.target).length();
  }

  spherical.setFromVector3(cam.position.clone().sub(this.target));

  domElement.addEventListener('pointerdown', function(e) {
    cameraUserInteracted = true;
    if (e.button === 0 && !e.shiftKey) {
      state = 1;
      rotateStart.set(e.clientX, e.clientY);
    } else if (e.button === 0 && e.shiftKey || e.button === 2) {
      state = 2;
      panStart.set(e.clientX, e.clientY);
    }
    domElement.setPointerCapture(e.pointerId);
  });

  domElement.addEventListener('pointermove', function(e) {
    if (state === 1) {
      var dx = e.clientX - rotateStart.x;
      var dy = e.clientY - rotateStart.y;
      sphericalDelta.theta -= dx * 0.005;
      sphericalDelta.phi -= dy * 0.005;
      rotateStart.set(e.clientX, e.clientY);
    } else if (state === 2) {
      var dx2 = e.clientX - panStart.x;
      var dy2 = e.clientY - panStart.y;
      var dist = getDistance();
      var offset = new THREE.Vector3();
      offset.setFromMatrixColumn(scope.camera.matrix, 0);
      offset.multiplyScalar(-dx2 * dist * 0.002);
      panOffset.add(offset);
      offset.setFromMatrixColumn(scope.camera.matrix, 1);
      offset.multiplyScalar(dy2 * dist * 0.002);
      panOffset.add(offset);
      panStart.set(e.clientX, e.clientY);
    }
  });

  domElement.addEventListener('pointerup', function(e) {
    state = 0;
    domElement.releasePointerCapture(e.pointerId);
  });

  domElement.addEventListener('wheel', function(e) {
    e.preventDefault();
    cameraUserInteracted = true;
    if (e.deltaY > 0) scale *= 1.1;
    else scale *= 0.9;
  }, { passive: false });

  domElement.addEventListener('contextmenu', function(e) { e.preventDefault(); });

  var touchDist = 0;

  domElement.addEventListener('touchstart', function(e) {
    cameraUserInteracted = true;
    if (e.touches.length === 1) {
      state = 1;
      rotateStart.set(e.touches[0].clientX, e.touches[0].clientY);
    } else if (e.touches.length === 2) {
      state = 2;
      var tdx = e.touches[0].clientX - e.touches[1].clientX;
      var tdy = e.touches[0].clientY - e.touches[1].clientY;
      touchDist = Math.sqrt(tdx*tdx + tdy*tdy);
      panStart.set(
        (e.touches[0].clientX + e.touches[1].clientX) / 2,
        (e.touches[0].clientY + e.touches[1].clientY) / 2
      );
    }
  }, { passive: false });

  domElement.addEventListener('touchmove', function(e) {
    e.preventDefault();
    if (state === 1 && e.touches.length === 1) {
      var tdx = e.touches[0].clientX - rotateStart.x;
      var tdy = e.touches[0].clientY - rotateStart.y;
      sphericalDelta.theta -= tdx * 0.005;
      sphericalDelta.phi -= tdy * 0.005;
      rotateStart.set(e.touches[0].clientX, e.touches[0].clientY);
    } else if (state === 2 && e.touches.length === 2) {
      var tdx2 = e.touches[0].clientX - e.touches[1].clientX;
      var tdy2 = e.touches[0].clientY - e.touches[1].clientY;
      var newDist = Math.sqrt(tdx2*tdx2 + tdy2*tdy2);
      scale *= touchDist / newDist;
      touchDist = newDist;
    }
  }, { passive: false });

  domElement.addEventListener('touchend', function() { state = 0; });

  this.update = function() {
    var offset = scope.camera.position.clone().sub(scope.target);
    spherical.setFromVector3(offset);

    if (scope.enableDamping) {
      spherical.theta += sphericalDelta.theta * scope.dampingFactor;
      spherical.phi += sphericalDelta.phi * scope.dampingFactor;
    } else {
      spherical.theta += sphericalDelta.theta;
      spherical.phi += sphericalDelta.phi;
    }

    spherical.phi = Math.max(0.01, Math.min(Math.PI - 0.01, spherical.phi));
    spherical.radius *= scale;
    spherical.radius = Math.max(2, Math.min(1000, spherical.radius));

    scope.target.add(panOffset);

    offset.setFromSpherical(spherical);
    scope.camera.position.copy(scope.target).add(offset);
    scope.camera.lookAt(scope.target);

    if (scope.enableDamping) {
      sphericalDelta.theta *= (1 - scope.dampingFactor);
      sphericalDelta.phi *= (1 - scope.dampingFactor);
    } else {
      sphericalDelta.set(0, 0, 0);
    }
    panOffset.set(0, 0, 0);
    scale = 1;
  };
}

var pendingRequest = null;
var debounceTimer = null;

function generate(apiUrl, params, onDone) {
  if (!params.version) params.version = CURRENT_VERSION;
  if (debounceTimer) clearTimeout(debounceTimer);
  if (pendingRequest) pendingRequest.abort();

  debounceTimer = setTimeout(function() {
    var ctrl = new AbortController();
    pendingRequest = ctrl;

    var blockCount = document.getElementById('gen-block-count');
    if (blockCount) blockCount.textContent = 'Generating...';

    fetch(apiUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      signal: ctrl.signal
    })
    .then(function(r) {
      if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
      return r.json();
    })
    .then(function(data) {
      pendingRequest = null;
      renderBlocks(data);
      if (blockCount) blockCount.textContent = data.blocks.length + ' blocks';
      if (onDone) onDone(data);
    })
    .catch(function(err) {
      if (err.name !== 'AbortError') {
        console.error('Generator error:', err);
        if (blockCount) blockCount.textContent = 'Error';
      }
    });
  }, 150);
}

function downloadNBT(downloadUrl, params) {
  if (!params.version) params.version = CURRENT_VERSION;
  var btn = document.getElementById('gen-download-btn');
  if (btn) btn.disabled = true;

  fetch(downloadUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params)
  })
  .then(function(r) {
    if (!r.ok) throw new Error('Download failed');
    var cd = r.headers.get('Content-Disposition') || '';
    var m = cd.match(/filename="([^"]+)"/);
    var fname = m ? m[1] : 'schematic.nbt';
    return r.blob().then(function(blob) { return { blob: blob, fname: fname }; });
  })
  .then(function(res) {
    var url = URL.createObjectURL(res.blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = res.fname;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  })
  .catch(function(err) {
    console.error('Download error:', err);
    if (typeof showToast === 'function') showToast('Download failed. Please try again.', 'danger');
  })
  .finally(function() {
    if (btn) btn.disabled = false;
  });
}

var COLOR_TO_CODE = {
  white:'wh',orange:'or',magenta:'ma',light_blue:'lb',yellow:'ye',lime:'li',pink:'pk',
  gray:'gy',light_gray:'lg',cyan:'cy',purple:'pu',blue:'bl',brown:'br',green:'gn',red:'re',black:'bk'
};
var CODE_TO_COLOR = {};
for (var ck in COLOR_TO_CODE) CODE_TO_COLOR[COLOR_TO_CODE[ck]] = ck;

var WOOD_TO_CODE = {
  oak:'o',spruce:'s',birch:'b',dark_oak:'d',jungle:'j',acacia:'a',cherry:'ch',crimson:'cr',warped:'wa'
};
var CODE_TO_WOOD = {};
for (var wk in WOOD_TO_CODE) CODE_TO_WOOD[WOOD_TO_CODE[wk]] = wk;

var ENUM_MAPS = {
  airfoilShape: { linear:'l', curved:'c' },
  bladeMaterial: { wool:'w', sail:'s' },
  envelopeMaterial: { wool:'w', envelope:'e' },
  frameMaterial: { wood:'w', andesite_casing:'a' },
  sternStyle: { round:'r', square:'s', pointed:'p' },
  color: COLOR_TO_CODE,
  wood: WOOD_TO_CODE
};

var ENUM_REVERSE = {};
for (var ek in ENUM_MAPS) {
  ENUM_REVERSE[ek] = {};
  for (var ev in ENUM_MAPS[ek]) ENUM_REVERSE[ek][ENUM_MAPS[ek][ev]] = ev;
}

var SCHEMAS = {
  p: [
    { k:'blades', t:'i' }, { k:'length', t:'i' }, { k:'rootChord', t:'i' },
    { k:'tipChord', t:'i' }, { k:'sweepDegrees', t:'f' }, { k:'swept', t:'b' },
    { k:'airfoilShape', t:'e', m:'airfoilShape' }, { k:'bladeMaterial', t:'e', m:'bladeMaterial' },
    { k:'bladeColor', t:'e', m:'color' }
  ],
  b: [
    { k:'lengthX', t:'i' }, { k:'widthZ', t:'i' }, { k:'heightY', t:'i' },
    { k:'cylinderMid', t:'f' }, { k:'frontTaper', t:'f' }, { k:'rearTaper', t:'f' },
    { k:'topFlatten', t:'f' }, { k:'bottomFlatten', t:'f' }, { k:'hollow', t:'b' },
    { k:'shell', t:'i' }, { k:'ribEnabled', t:'b' }, { k:'ribSpacing', t:'i' },
    { k:'keelEnabled', t:'b' }, { k:'keelDepth', t:'i' }, { k:'finEnabled', t:'b' },
    { k:'sideFinEnabled', t:'b' }, { k:'finHeight', t:'i' }, { k:'finLength', t:'i' },
    { k:'envelopeMaterial', t:'e', m:'envelopeMaterial' }, { k:'envelopeColor', t:'e', m:'color' },
    { k:'frameMaterial', t:'e', m:'frameMaterial' }, { k:'frameWoodType', t:'e', m:'wood' }
  ],
  h: [
    { k:'woodType', t:'e', m:'wood' }, { k:'length', t:'i' }, { k:'beam', t:'i' },
    { k:'depth', t:'i' }, { k:'bottomPinch', t:'f' }, { k:'hullFlare', t:'f' },
    { k:'flareCurve', t:'f' }, { k:'tumblehome', t:'f' }, { k:'tumbleCurve', t:'f' },
    { k:'sheerCurve', t:'f' }, { k:'sheerCurveExp', t:'f' },
    { k:'bowLength', t:'i' }, { k:'bowSharpness', t:'f' }, { k:'bowKeelRise', t:'f' },
    { k:'bowKeelLength', t:'i' }, { k:'sternStyle', t:'e', m:'sternStyle' },
    { k:'sternLength', t:'i' }, { k:'sternSharpness', t:'f' },
    { k:'sternKeelRise', t:'f' }, { k:'sternKeelLength', t:'i' },
    { k:'keelCurve', t:'f' }, { k:'castleBlend', t:'i' },
    { k:'hasRailings', t:'b' }, { k:'hasTrim', t:'b' }, { k:'hasWindows', t:'b' },
    { k:'castleHeight', t:'i' }, { k:'castleLength', t:'i' },
    { k:'forecastleHeight', t:'i' }, { k:'forecastleLength', t:'i' },
    { k:'hasGunPorts', t:'b' }, { k:'gunPortRow', t:'i' }, { k:'gunPortSpacing', t:'i' }
  ]
};

var CURRENT_VERSION = 1;

function encodeCompact(prefix, params) {
  var schema = SCHEMAS[prefix];
  if (!schema) return '';
  var ver = params.version || CURRENT_VERSION;
  var vals = [prefix + ver];
  for (var i = 0; i < schema.length; i++) {
    var s = schema[i];
    var v = params[s.k];
    if (v === undefined || v === null) v = '';
    switch (s.t) {
      case 'b': vals.push(v ? '1' : '0'); break;
      case 'i': vals.push(String(Math.round(Number(v)))); break;
      case 'f': vals.push(String(Number(v))); break;
      case 'e':
        var map = ENUM_MAPS[s.m];
        vals.push(map && map[v] ? map[v] : String(v).slice(0, 3));
        break;
    }
  }
  return vals.join('.');
}

function decodeCompact(hash) {
  if (!hash) return { params: {}, view: null };
  var str = hash.charAt(0) === '#' ? hash.slice(1) : hash;
  var view = null;
  if (str.indexOf('/g') === str.length - 2) {
    view = 'guide';
    str = str.slice(0, -2);
  }
  if (!str || str.length < 3) return { params: {}, view: view };
  var parts = str.split('.');
  var header = parts[0];
  var prefix = header.charAt(0);
  var version = parseInt(header.slice(1), 10) || CURRENT_VERSION;
  var schema = SCHEMAS[prefix];
  if (!schema) return { params: {}, view: view };
  var params = { version: version };
  for (var i = 0; i < schema.length; i++) {
    var s = schema[i];
    var raw = parts[i + 1];
    if (raw === undefined || raw === '') continue;
    switch (s.t) {
      case 'b': params[s.k] = raw === '1'; break;
      case 'i': params[s.k] = parseInt(raw, 10); break;
      case 'f': params[s.k] = parseFloat(raw); break;
      case 'e':
        var rev = ENUM_REVERSE[s.m];
        params[s.k] = rev && rev[raw] ? rev[raw] : raw;
        break;
    }
  }
  return { params: params, view: view };
}

var PREFIX_TO_TYPE = { p: 'propeller', b: 'balloon', h: 'hull' };

function toBase64Url(str) {
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function fromBase64Url(b64) {
  var s = b64.replace(/-/g, '+').replace(/_/g, '/');
  while (s.length % 4) s += '=';
  return atob(s);
}

function getGeneratorBasePath(prefix) {
  var type = PREFIX_TO_TYPE[prefix] || 'propeller';
  return '/generators/' + type;
}

function updateHash(prefix, params) {
  var compact = encodeCompact(prefix, params);
  var encoded = toBase64Url(compact);
  var newPath = getGeneratorBasePath(prefix) + '/' + encoded;
  if (window.location.pathname !== newPath) {
    history.replaceState(null, '', newPath);
  }
  var type = PREFIX_TO_TYPE[prefix];
  if (type) {
    try { localStorage.setItem('gen_hash_' + type, encoded); } catch(e) {}
  }
}

function getShareURL(prefix, params, view) {
  var compact = encodeCompact(prefix, params);
  var encoded = toBase64Url(compact);
  var url = window.location.origin + getGeneratorBasePath(prefix) + '/' + encoded;
  if (view === 'guide') url += '/guide';
  return url;
}

function applyHashParams(setParamsFn, initHash, generatorType) {
  if (initHash) {
    var compact;
    try { compact = fromBase64Url(initHash); } catch(e) { compact = ''; }
    var decoded = decodeCompact(compact);
    if (Object.keys(decoded.params).length > 0) {
      setParamsFn(decoded.params);
    }
    return decoded;
  }
  var hash = window.location.hash;
  if (hash) {
    var decoded2 = decodeCompact(hash);
    if (Object.keys(decoded2.params).length > 0) {
      setParamsFn(decoded2.params);
    }
    return decoded2;
  }
  if (generatorType) {
    try {
      var stored = localStorage.getItem('gen_hash_' + generatorType);
      if (stored) {
        var storedCompact = fromBase64Url(stored);
        var decoded3 = decodeCompact(storedCompact);
        if (Object.keys(decoded3.params).length > 0) {
          setParamsFn(decoded3.params);
          var prefix = storedCompact.charAt(0);
          updateHash(prefix, decoded3.params);
          return decoded3;
        }
      }
    } catch(e) {}
  }
  return { params: {} };
}

window.GeneratorApp = {
  initScene: initScene,
  renderBlocks: renderBlocks,
  generate: generate,
  downloadNBT: downloadNBT,
  updateHash: updateHash,
  getShareURL: getShareURL,
  applyHashParams: applyHashParams,
  decodeCompact: decodeCompact,
  encodeCompact: encodeCompact,
  toBase64Url: toBase64Url,
  fromBase64Url: fromBase64Url
};

})();
