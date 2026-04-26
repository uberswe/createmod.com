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

var BLOCK_COLORS = {
  7: 0xeaeaea,  // wool - brighter white
  1: 0x7a5c3a,  // plank - warmer spruce
  2: 0x7a5c3a,  // slab bottom
  3: 0x7a5c3a,  // slab top
  4: 0x7a5c3a,  // stair
  5: 0x7a5c3a,  // fence
  6: 0x6b5030,  // trapdoor - slightly darker
  8: 0x3d2b15   // log - dark bark
};

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

  var ambient = new THREE.AmbientLight(0xffeedd, 0.45);
  scene.add(ambient);
  var key = new THREE.DirectionalLight(0xfff5e0, 0.9);
  key.position.set(50, 80, 40);
  scene.add(key);
  var fill = new THREE.DirectionalLight(0x88aaff, 0.35);
  fill.position.set(-30, 20, -30);
  scene.add(fill);
  var rim = new THREE.DirectionalLight(0xffaa88, 0.25);
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

  // Clean up on HTMX navigation away
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

function clearBlocks() {
  for (var i = 0; i < blockMeshes.length; i++) {
    scene.remove(blockMeshes[i]);
    if (blockMeshes[i].geometry) blockMeshes[i].geometry.dispose();
    if (blockMeshes[i].material) blockMeshes[i].material.dispose();
  }
  blockMeshes = [];
}

function renderBlocks(data) {
  if (!scene || !THREE) return;
  clearBlocks();
  currentData = data;

  var blocks = data.blocks;
  if (!blocks || blocks.length === 0) return;

  // Group blocks by type for instanced rendering
  var groups = {};
  for (var i = 0; i < blocks.length; i++) {
    var b = blocks[i];
    var t = b.type || 1;
    if (!groups[t]) groups[t] = [];
    groups[t].push(b);
  }

  var cx = data.sizeX / 2;
  var cy = data.sizeY / 2;
  var cz = data.sizeZ / 2;

  var boxGeo = new THREE.BoxGeometry(0.95, 0.95, 0.95);
  var slabGeo = new THREE.BoxGeometry(0.95, 0.47, 0.95);
  var fenceGeo = new THREE.BoxGeometry(0.25, 0.95, 0.25);
  var trapdoorGeo = new THREE.BoxGeometry(0.9, 0.18, 0.9);

  var dummy = new THREE.Object3D();

  for (var type in groups) {
    var arr = groups[type];
    var t = parseInt(type);
    var color = BLOCK_COLORS[t] || 0x888888;
    var roughness = 0.75;
    var metalness = 0.08;
    if (t === 7) { roughness = 0.92; metalness = 0.0; }
    else if (t === 8) { roughness = 0.8; metalness = 0.05; }
    var mat = new THREE.MeshStandardMaterial({
      color: color,
      roughness: roughness,
      metalness: metalness
    });

    var geo;
    if (t === 2 || t === 3) {
      geo = slabGeo;
    } else if (t === 5) {
      geo = fenceGeo;
    } else if (t === 6) {
      geo = trapdoorGeo;
    } else {
      geo = boxGeo;
    }

    var mesh = new THREE.InstancedMesh(geo, mat, arr.length);
    for (var j = 0; j < arr.length; j++) {
      var yOff = 0;
      if (t === 2) yOff = -0.24;
      if (t === 3) yOff = 0.24;
      dummy.position.set(arr[j].x - cx, arr[j].y - cy + yOff, arr[j].z - cz);
      dummy.updateMatrix();
      mesh.setMatrixAt(j, dummy.matrix);
    }
    mesh.instanceMatrix.needsUpdate = true;
    scene.add(mesh);
    blockMeshes.push(mesh);
  }

  // Adjust camera
  var maxDim = Math.max(data.sizeX, data.sizeY, data.sizeZ);
  var dist = maxDim * 1.2;
  camera.position.set(dist * 0.7, dist * 0.5, dist * 0.7);
  controls.target.set(0, 0, 0);
  controls.update();
}

// Simple orbit controls (no dependency on three/examples)
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
  var state = 0; // 0=none, 1=rotate, 2=pan
  var scale = 1;

  function getDistance() {
    return scope.camera.position.clone().sub(scope.target).length();
  }

  spherical.setFromVector3(cam.position.clone().sub(this.target));

  domElement.addEventListener('pointerdown', function(e) {
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
    if (e.deltaY > 0) scale *= 1.1;
    else scale *= 0.9;
  }, { passive: false });

  domElement.addEventListener('contextmenu', function(e) { e.preventDefault(); });

  // Touch support
  var touchDist = 0;

  domElement.addEventListener('touchstart', function(e) {
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

// Debounced API call
var pendingRequest = null;
var debounceTimer = null;

function generate(apiUrl, params, onDone) {
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

window.GeneratorApp = {
  initScene: initScene,
  renderBlocks: renderBlocks,
  generate: generate,
  downloadNBT: downloadNBT
};

})();
