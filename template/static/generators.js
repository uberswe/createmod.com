// Client-side generator engines for propeller, balloon, and hull.
// Produces the same {blocks, sizeX, sizeY, sizeZ, materials} output
// that the server API returned, so the rest of the UI is unchanged.
(function(){
'use strict';

var BT = { AIR:0, PLANK:1, SLAB_BOT:2, SLAB_TOP:3, STAIR:4, FENCE:5, TRAPDOOR:6, WOOL:7, LOG:8, SAIL:9 };

function clamp(v, lo, hi) { return v < lo ? lo : v > hi ? hi : v; }
function smoothstep(t) { var x = clamp(t, 0, 1); return x * x * (3 - 2 * x); }
function smootherstep(t) { return t * t * t * (t * (t * 6 - 15) + 10); }
function symRound(v) { return v >= 0 ? Math.floor(v + 0.5) : -Math.floor(-v + 0.5); }

function sampleRange(lo, hi, step) {
  var out = [];
  for (var v = lo; v <= hi + 1e-9; v += step) out.push(v);
  return out;
}

// ===== Propeller Generator =====

function generatePropeller(p) {
  var blades = clamp(p.blades || 2, 2, 12);
  var length = clamp(p.length || 10, 3, 50);
  var rootChord = clamp(p.rootChord || 3, 1, 40);
  var tipChord = clamp(p.tipChord || 1, 0, 40);
  var sweepDeg = clamp(p.sweepDegrees || 0, 0, 90);
  var swept = !!p.swept;
  var airfoil = (p.airfoilShape === 'curved') ? 'curved' : 'linear';
  var bladeMat = (p.bladeMaterial === 'sail') ? 'sail' : 'wool';
  var bladeColor = p.bladeColor || 'white';
  var blockType = bladeMat === 'sail' ? BT.SAIL : BT.WOOL;
  var sweepRad = sweepDeg * Math.PI / 180;

  var seen = {};
  var blocks = [];

  for (var b = 0; b < blades; b++) {
    var angle = (b / blades) * 2 * Math.PI;
    var samples = sampleRange(0, length, 0.35);
    for (var si = 0; si < samples.length; si++) {
      var r = samples[si];
      var t = clamp(r / length, 0, 1);
      var chord = rootChord + (tipChord - rootChord) * t;
      if (airfoil === 'curved') {
        chord += Math.sin(t * Math.PI) * Math.min(1.3, rootChord * 0.4);
      }
      if (chord < 0.5) continue;

      var localAngle = angle;
      if (swept) localAngle += sweepRad * t;

      var halfC = Math.max(0, (chord - 1) / 2);
      var wSamples = sampleRange(-halfC, halfC, 0.35);
      for (var wi = 0; wi < wSamples.length; wi++) {
        var w = wSamples[wi];
        var bx = symRound(r * Math.cos(localAngle) - w * Math.sin(localAngle));
        var bz = symRound(r * Math.sin(localAngle) + w * Math.cos(localAngle));
        var k = bx + ',' + bz;
        if (seen[k]) continue;
        seen[k] = true;
        blocks.push({ x: bx, y: 0, z: bz, type: blockType });
      }
    }
  }

  // Normalize
  var minX = 1e9, minZ = 1e9, maxX = -1e9, maxZ = -1e9;
  for (var i = 0; i < blocks.length; i++) {
    var bl = blocks[i];
    if (bl.x < minX) minX = bl.x;
    if (bl.z < minZ) minZ = bl.z;
    if (bl.x > maxX) maxX = bl.x;
    if (bl.z > maxZ) maxZ = bl.z;
  }
  if (blocks.length === 0) { minX = 0; minZ = 0; maxX = 0; maxZ = 0; }
  for (var i = 0; i < blocks.length; i++) {
    blocks[i].x -= minX;
    blocks[i].z -= minZ;
  }

  return {
    blocks: blocks,
    sizeX: maxX - minX + 1,
    sizeY: 1,
    sizeZ: maxZ - minZ + 1,
    materials: { bladeMaterial: bladeMat, bladeColor: bladeColor }
  };
}

// ===== Balloon Generator =====

function generateBalloon(p) {
  var lx = clamp(p.lengthX || 12, 6, 120);
  var wz = clamp(p.widthZ || 12, 4, 60);
  var hy = clamp(p.heightY || 16, 4, 60);
  var cylinderMid = clamp(p.cylinderMid || 0, 0, 0.85);
  var frontTaper = clamp(p.frontTaper || 0, 0, 1);
  var rearTaper = clamp(p.rearTaper || 0, 0, 1);
  var topFlatten = clamp(p.topFlatten || 0, 0, 0.5);
  var bottomFlatten = clamp(p.bottomFlatten || 0, 0, 0.5);
  var hollow = p.hollow !== false;
  var shell = clamp(p.shell || 1, 1, 5);
  var ribEnabled = !!p.ribEnabled;
  var ribSpacing = clamp(p.ribSpacing || 4, 2, 12);
  var keelEnabled = !!p.keelEnabled;
  var keelDepth = clamp(p.keelDepth || 0, 0, 10);
  var finEnabled = !!p.finEnabled;
  var sideFinEnabled = !!p.sideFinEnabled;
  var finHeight = clamp(p.finHeight || 2, 2, 15);
  var finLength = clamp(p.finLength || 3, 3, 20);
  var envMat = p.envelopeMaterial || 'wool';
  if (envMat !== 'wool' && envMat !== 'envelope') envMat = 'wool';
  var envColor = p.envelopeColor || 'white';
  var frameMat = p.frameMaterial || 'wood';
  if (frameMat !== 'wood' && frameMat !== 'andesite_casing') frameMat = 'wood';
  var frameWood = p.frameWoodType || 'spruce';

  var rX = lx / 2;
  var rY = hy / 2;
  var rZ = wz / 2;

  var midHalf = rX * cylinderMid;
  var capLen = Math.max(1, rX - midHalf);

  var cX = Math.floor(rX);
  var cY = Math.floor(rY);
  var cZ = Math.floor(rZ);

  var sizeX = lx + 2;
  var sizeY = hy + 2;
  var sizeZ = wz + 2;

  function eDist(x, y, z) {
    var ax = x - cX;
    var nx;
    if (midHalf > 0 && Math.abs(ax) <= midHalf) {
      nx = 0;
    } else {
      var capOffset = Math.abs(ax);
      if (midHalf > 0) capOffset -= midHalf;
      nx = capOffset / capLen;
      if (ax < 0) nx = -nx;
    }

    var ny = (y - cY) / rY;
    var nz = (z - cZ) / rZ;

    if (frontTaper > 0 && nx < 0) {
      var t = Math.abs(nx);
      var sq = 1 + frontTaper * t * t * 3;
      ny *= sq;
      nz *= sq;
    }
    if (rearTaper > 0 && nx > 0) {
      var t2 = Math.abs(nx);
      var sq2 = 1 + rearTaper * t2 * t2 * 3;
      ny *= sq2;
      nz *= sq2;
    }

    if (topFlatten > 0 && ny < 0) {
      ny *= (1 - topFlatten * 0.5);
    }
    if (bottomFlatten > 0 && ny > 0) {
      ny *= (1 - bottomFlatten * 0.5);
    }

    return nx * nx + ny * ny + nz * nz;
  }

  var dirs = [[1,0,0],[-1,0,0],[0,1,0],[0,-1,0],[0,0,1],[0,0,-1]];

  // Pass 1a: collect inside set
  var insideKeys = {};
  function coordKey(x, y, z) { return x + ',' + y + ',' + z; }

  for (var x = 0; x < sizeX; x++) {
    for (var y = 0; y < sizeY; y++) {
      for (var z = 0; z < sizeZ; z++) {
        if (eDist(x, y, z) <= 1.0) {
          insideKeys[coordKey(x, y, z)] = true;
        }
      }
    }
  }

  var grid = {};

  if (!hollow) {
    for (var ik in insideKeys) {
      grid[ik] = BT.WOOL;
    }
  } else {
    // Pass 1b: surface layer
    var shellKeys = {};
    for (var ik2 in insideKeys) {
      var parts = ik2.split(',');
      var px = parseInt(parts[0]), py = parseInt(parts[1]), pz = parseInt(parts[2]);
      for (var di = 0; di < 6; di++) {
        var nbk = coordKey(px + dirs[di][0], py + dirs[di][1], pz + dirs[di][2]);
        if (!insideKeys[nbk]) {
          shellKeys[ik2] = true;
          break;
        }
      }
    }

    // Pass 1c: thicken shell
    for (var layer = 1; layer < shell; layer++) {
      var newLayer = [];
      for (var ik3 in insideKeys) {
        if (shellKeys[ik3]) continue;
        var parts2 = ik3.split(',');
        var px2 = parseInt(parts2[0]), py2 = parseInt(parts2[1]), pz2 = parseInt(parts2[2]);
        for (var di2 = 0; di2 < 6; di2++) {
          var nbk2 = coordKey(px2 + dirs[di2][0], py2 + dirs[di2][1], pz2 + dirs[di2][2]);
          if (shellKeys[nbk2]) {
            newLayer.push(ik3);
            break;
          }
        }
      }
      for (var ni = 0; ni < newLayer.length; ni++) {
        shellKeys[newLayer[ni]] = true;
      }
    }

    for (var sk in shellKeys) {
      grid[sk] = BT.WOOL;
    }
  }

  // Pass 3: ribbing
  if (ribEnabled && ribSpacing > 0) {
    for (var gk in grid) {
      if (grid[gk] !== BT.WOOL) continue;
      var gp = gk.split(',');
      if (parseInt(gp[0]) % ribSpacing === 0) {
        grid[gk] = BT.LOG;
      }
    }
  }

  // Pass 4: keel
  if (keelEnabled && keelDepth > 0) {
    var midZ = cZ;
    for (var kx = 0; kx < sizeX; kx++) {
      var minY = -1;
      for (var ky = 0; ky < sizeY; ky++) {
        if (grid[coordKey(kx, ky, midZ)] !== undefined) {
          minY = ky;
          break;
        }
      }
      if (minY >= 0) {
        for (var dy = 1; dy <= keelDepth; dy++) {
          grid[coordKey(kx, minY - dy, midZ)] = BT.LOG;
        }
      }
    }
  }

  // Pass 5: tail fins
  if (finEnabled && finHeight > 0 && finLength > 0) {
    var fMidZ = cZ;
    var finStartX = sizeX - finLength - 1;

    // Vertical fin (top in renderer)
    for (var fx = Math.max(0, finStartX); fx < sizeX; fx++) {
      var progress = (fx - finStartX) / finLength;
      var h = Math.ceil(finHeight * (1 - progress));

      var botY = -1;
      for (var fy = sizeY - 1; fy >= 0; fy--) {
        if (grid[coordKey(fx, fy, fMidZ)] !== undefined) {
          botY = fy;
          break;
        }
      }
      if (botY >= 0) {
        for (var fdy = 1; fdy <= h; fdy++) {
          grid[coordKey(fx, botY + fdy, fMidZ)] = BT.PLANK;
        }
      }
    }

    // Horizontal fins at center Y
    var fMidY = cY;
    for (var fx2 = Math.max(0, sizeX - finLength - 1); fx2 < sizeX; fx2++) {
      var progress2 = (fx2 - (sizeX - finLength - 1)) / finLength;
      var w = Math.ceil(finHeight * 0.7 * (1 - progress2));
      for (var fdz = 1; fdz <= w; fdz++) {
        if (grid[coordKey(fx2, fMidY, fMidZ)] !== undefined) {
          grid[coordKey(fx2, fMidY, fMidZ - fdz)] = BT.PLANK;
          grid[coordKey(fx2, fMidY, fMidZ + fdz)] = BT.PLANK;
        }
      }
    }
  }

  // Pass 5b: side fins
  if (sideFinEnabled && finHeight > 0 && finLength > 0) {
    var sfMidY = cY;
    var sfMidZ = cZ;
    var sfStartX = sizeX - finLength - 1;

    for (var sfx = Math.max(0, sfStartX); sfx < sizeX; sfx++) {
      var sfProgress = (sfx - sfStartX) / finLength;
      var sfH = Math.ceil(finHeight * 0.7 * (1 - sfProgress));

      var sfMinZ = -1, sfMaxZ = -1;
      for (var sfz = 0; sfz < sizeZ; sfz++) {
        if (grid[coordKey(sfx, sfMidY, sfz)] !== undefined) {
          if (sfMinZ < 0) sfMinZ = sfz;
          sfMaxZ = sfz;
        }
      }
      if (sfMinZ >= 0) {
        for (var sfdz = 1; sfdz <= sfH; sfdz++) {
          grid[coordKey(sfx, sfMidY, sfMinZ - sfdz)] = BT.PLANK;
          grid[coordKey(sfx, sfMidY, sfMaxZ + sfdz)] = BT.PLANK;
        }
      }
    }
  }

  // Build result — normalize coordinates
  var blocks = [];
  var minGridY = 0;
  for (var gk2 in grid) {
    var gp2 = gk2.split(',');
    var gy = parseInt(gp2[1]);
    if (gy < minGridY) minGridY = gy;
  }
  var yOffset = minGridY < 0 ? -minGridY : 0;

  var maxBX = 0, maxBY = 0, maxBZ = 0;
  for (var gk3 in grid) {
    var gp3 = gk3.split(',');
    var bx = parseInt(gp3[0]), by = parseInt(gp3[1]) + yOffset, bz = parseInt(gp3[2]);
    blocks.push({ x: bx, y: by, z: bz, type: grid[gk3] });
    if (bx > maxBX) maxBX = bx;
    if (by > maxBY) maxBY = by;
    if (bz > maxBZ) maxBZ = bz;
  }

  var woodType = frameWood;
  if (frameMat === 'andesite_casing') woodType = 'spruce';

  return {
    blocks: blocks,
    sizeX: maxBX + 1,
    sizeY: maxBY + 1,
    sizeZ: maxBZ + 1,
    materials: {
      woodType: woodType,
      envelopeMaterial: envMat,
      envelopeColor: envColor,
      frameMaterial: frameMat
    }
  };
}

// ===== Hull Generator =====

function generateHull(p) {
  var L = clamp(p.length || 40, 20, 200);
  var B = clamp(p.beam || 10, 4, 40);
  var D = clamp(p.depth || 6, 3, 20);
  var bottomPinch = clamp(p.bottomPinch || 0.3, 0.1, 0.7);
  var hullFlare = clamp(p.hullFlare || 0, 0, 0.6);
  var flareCurve = clamp(p.flareCurve || 2.6, 1.2, 4.0);
  var tumblehome = clamp(p.tumblehome || 0, 0, 0.4);
  var tumbleCurve = clamp(p.tumbleCurve || 3, 1.5, 5.0);
  var sheerCurve = clamp(p.sheerCurve || 0, 0, 0.75);
  var sheerCurveExp = clamp(p.sheerCurveExp || 2, 1.0, 4.0);
  var bowLength = clamp(p.bowLength || 8, 2, Math.floor(L / 2));
  var bowSharpness = clamp(p.bowSharpness || 1.3, 0.4, 2.5);
  var bowKeelRise = clamp(p.bowKeelRise || 0, 0, 1.5);
  var bowKeelLength = clamp(p.bowKeelLength || 0, 0, 40);
  var bowCurve = clamp(p.bowCurve || 0, -1.0, 1.0);
  var sternStyle = p.sternStyle;
  if (sternStyle !== 'square' && sternStyle !== 'round' && sternStyle !== 'pointed') sternStyle = 'round';
  var sternLength = clamp(p.sternLength || 5, 2, Math.floor(L / 2));
  var sternSharpness = clamp(p.sternSharpness || 0.7, 0.2, 2.0);
  var sternKeelRise = clamp(p.sternKeelRise || 0, 0, 1.5);
  var sternKeelLength = clamp(p.sternKeelLength || 0, 0, 30);
  var sternOverhang = clamp(p.sternOverhang || 0, 0, 1.0);
  var keelCurveVal = clamp(p.keelCurve || 1.7, 0.7, 3.5);
  var castleBlend = clamp(p.castleBlend || 4, 2, 12);
  var hasRailings = !!p.hasRailings;
  var hasTrim = !!p.hasTrim;
  var hasWindows = !!p.hasWindows;
  var castleHeight = clamp(p.castleHeight || 0, 0, 6);
  var castleLength = clamp(p.castleLength || 0, 0, Math.floor(L * 0.55));
  var forecastleHeight = clamp(p.forecastleHeight || 0, 0, 3);
  var forecastleLength = clamp(p.forecastleLength || 0, 0, Math.floor(L * 0.5));
  var hasGunPorts = !!p.hasGunPorts;
  var gunPortRow = clamp(p.gunPortRow || 2, 1, 6);
  var gunPortSpacing = clamp(p.gunPortSpacing || 4, 2, 8);
  var midWidthBias = clamp(p.midWidthBias || 0, 0, 1.0);

  var depth = D;
  var length = L;

  function crossSectionFactor(yNorm) {
    var yc = clamp(yNorm, 0, 1);
    var sy = smootherstep(yc);
    var base = bottomPinch + (1 - bottomPinch) * sy;
    var flare = hullFlare * Math.pow(yc, flareCurve);
    var tumble = tumblehome * Math.pow(yc, tumbleCurve);
    var above = yNorm - 1; if (above < 0) above = 0;
    var castleTaper = above * 0.32 + above * above * 0.18;
    var result = base + flare - tumble - castleTaper;
    return result < 0.12 ? 0.12 : result;
  }

  function longitudinalFactor(zNorm) {
    var bowStart = 1.0 - bowLength / length;
    var sternEnd = sternLength / length;

    if (zNorm <= sternEnd) {
      var t = zNorm / Math.max(sternEnd, 0.001);
      if (t < 0) t = 0;
      var st = smootherstep(t);
      if (sternStyle === 'square') {
        var f = Math.pow(st, sternSharpness);
        return f < 0.72 ? 0.72 : f;
      } else if (sternStyle === 'round') {
        return Math.pow(st, sternSharpness * 0.55);
      } else {
        return Math.pow(st, sternSharpness);
      }
    }
    if (zNorm >= bowStart) {
      var t = (1 - zNorm) / Math.max(1 - bowStart, 0.001);
      var st = smootherstep(Math.max(t, 0));
      var base = Math.pow(Math.max(st, 0), bowSharpness);
      if (bowCurve !== 0) {
        if (bowCurve > 0) {
          var convex = Math.sqrt(Math.max(st, 0));
          base = base * (1 - bowCurve) + convex * bowCurve;
        } else {
          var concave = st * st * st;
          base = base * (1 + bowCurve) + concave * (-bowCurve);
        }
      }
      return base;
    }
    if (midWidthBias > 0) {
      var bowSt = 1.0 - bowLength / length;
      var sternEn = sternLength / length;
      var midNorm = sternEn + (bowSt - sternEn) * (0.5 - midWidthBias * 0.35);
      if (zNorm < midNorm) {
        var t = (zNorm - sternEn) / Math.max(midNorm - sternEn, 0.001);
        return 0.85 + 0.15 * Math.pow(t, 0.6);
      }
    }
    return 1;
  }

  function halfWidthAt(y, z) {
    var yNorm = y / Math.max(depth, 1);
    var zNorm = z / Math.max(length - 1, 1);
    var base = crossSectionFactor(yNorm) * longitudinalFactor(zNorm) * (B / 2);
    if (sternOverhang > 0 && yNorm > 1.0 && zNorm < sternLength / length) {
      base += sternOverhang * 0.3 * (yNorm - 1.0) * (B / 2);
    }
    return base;
  }

  function keelYAt(z) {
    var zNorm = z / Math.max(length - 1, 1);
    var rise = 0;
    if (bowKeelRise > 0 && bowKeelLength > 0) {
      var start = 1.0 - bowKeelLength / length;
      if (zNorm > start) {
        var t = (zNorm - start) / Math.max(1 - start, 0.001);
        var r = Math.pow(t, keelCurveVal) * bowKeelRise;
        if (r > rise) rise = r;
      }
    }
    if (sternKeelRise > 0 && sternKeelLength > 0) {
      var end = sternKeelLength / length;
      if (zNorm < end) {
        var t = (end - zNorm) / Math.max(end, 0.001);
        var r = Math.pow(t, keelCurveVal) * sternKeelRise;
        if (r > rise) rise = r;
      }
    }
    return Math.round(rise * depth);
  }

  function deckYAt(z) {
    var y = depth;
    if (sheerCurve > 0) {
      var zNorm = z / Math.max(length - 1, 1);
      var t = Math.abs(zNorm - 0.5) * 2;
      y += sheerCurve * depth * Math.pow(t, sheerCurveExp);
    }
    if (castleHeight > 0 && castleLength > 0) {
      var cL = Math.min(castleLength, Math.floor(length * 0.55));
      var blend = Math.min(Math.floor(cL * 0.55), castleBlend);
      if (blend < 2) blend = 2;
      if (z < cL - blend) y += castleHeight;
      else if (z < cL) y += castleHeight * (1 - smoothstep((z - (cL - blend)) / blend));
    }
    if (forecastleHeight > 0 && forecastleLength > 0) {
      var fL = Math.min(forecastleLength, Math.floor(length * 0.5));
      var blend = Math.min(Math.floor(fL * 0.55), castleBlend);
      if (blend < 2) blend = 2;
      var zFromBow = L - 1 - z;
      if (zFromBow < fL - blend) y += forecastleHeight;
      else if (zFromBow < fL) y += forecastleHeight * (1 - smoothstep((zFromBow - (fL - blend)) / blend));
    }
    return Math.round(y);
  }

  // --- Pass 1: build hull volume
  var keelYArr = new Int32Array(L);
  var deckYArr = new Int32Array(L);
  var maxDeckY = D;

  for (var z = 0; z < L; z++) {
    deckYArr[z] = deckYAt(z);
    if (deckYArr[z] > maxDeckY) maxDeckY = deckYArr[z];
  }

  // hwArr[y][z] = half-width int
  var hwArr = [];
  for (var y = 0; y <= maxDeckY; y++) {
    hwArr[y] = new Int32Array(L);
    for (var z = 0; z < L; z++) hwArr[y][z] = -1;
  }

  // Compute raw half-widths
  var rawHW = [];
  for (var y = 0; y <= maxDeckY; y++) rawHW[y] = new Float64Array(L);
  for (var z = 0; z < L; z++) {
    keelYArr[z] = keelYAt(z);
    for (var y = keelYArr[z]; y <= deckYArr[z]; y++) {
      if (y <= maxDeckY) rawHW[y][z] = halfWidthAt(y, z);
    }
  }

  // Smooth half-widths in bow/stern
  var bowStartZ = L - bowLength;
  var sternEndZ = sternLength;
  for (var y = 0; y <= maxDeckY; y++) {
    var sm = new Float64Array(L);
    for (var z = 0; z < L; z++) sm[z] = rawHW[y][z];
    for (var z = 1; z < L - 1; z++) {
      if (z >= sternEndZ && z <= bowStartZ) continue;
      var prev = rawHW[y][z-1], cur = rawHW[y][z], next = rawHW[y][z+1];
      if (prev > 0 || cur > 0 || next > 0) sm[z] = prev * 0.25 + cur * 0.5 + next * 0.25;
    }
    rawHW[y] = sm;
  }

  // Build hull volume using typed key encoding for speed
  // Pack (x+200, y, z) into a single integer for fast lookup
  var XOFF = 200;
  var YS = 400; // x range stride
  var ZS = YS * (maxDeckY + 2); // y range stride
  var inHull = new Uint8Array(ZS * (L + 1));

  function hullKey(x, y, z) { return (x + XOFF) + y * YS + z * ZS; }
  function hasHull(x, y, z) {
    if (x + XOFF < 0 || x + XOFF >= YS || y < 0 || y > maxDeckY || z < 0 || z >= L) return false;
    return inHull[hullKey(x, y, z)] === 1;
  }

  for (var z = 0; z < L; z++) {
    for (var y = keelYArr[z]; y <= deckYArr[z]; y++) {
      var hw = y <= maxDeckY ? rawHW[y][z] : 0;
      if (hw < 0.15) continue;
      var maxX = Math.max(0, Math.round(hw - 0.0001));
      if (y <= maxDeckY) hwArr[y][z] = maxX;
      for (var x = -maxX; x <= maxX; x++) {
        inHull[hullKey(x, y, z)] = 1;
      }
    }
  }

  // --- Pass 2: shell
  var blocks = [];
  var blockMap = {};

  function bKey(x, y, z) { return x + ',' + y + ',' + z; }
  function setBlock(x, y, z, type, props) {
    var k = bKey(x, y, z);
    var b = { x: x, y: y, z: z, type: type };
    if (props) b.props = props;
    blockMap[k] = b;
  }
  function getBlock(x, y, z) { return blockMap[bKey(x, y, z)]; }

  for (var z = 0; z < L; z++) {
    for (var y = keelYArr[z]; y <= deckYArr[z]; y++) {
      var hw = y <= maxDeckY ? hwArr[y][z] : -1;
      if (hw < 0) continue;
      for (var x = -hw; x <= hw; x++) {
        if (!hasHull(x, y, z)) continue;
        var exposed = !hasHull(x-1,y,z) || !hasHull(x+1,y,z) ||
                      !hasHull(x,y-1,z) || !hasHull(x,y+1,z) ||
                      !hasHull(x,y,z-1) || !hasHull(x,y,z+1);
        if (exposed || y === deckYArr[z]) {
          setBlock(x, y, z, BT.PLANK);
        }
      }
    }
  }

  // --- Pass 3: lateral flare stairs
  for (var z = 0; z < L; z++) {
    for (var y = keelYArr[z]; y < deckYArr[z]; y++) {
      var hwHere = (y >= 0 && y <= maxDeckY) ? hwArr[y][z] : -1;
      var hwUp = (y+1 >= 0 && y+1 <= maxDeckY) ? hwArr[y+1][z] : -1;
      if (hwUp <= hwHere) continue;
      for (var xN = hwHere + 1; xN <= hwUp; xN++) {
        if (hasHull(xN, y, z)) continue;
        if (!hasHull(xN, y+1, z)) continue;
        var ex = getBlock(xN, y, z);
        if (ex && ex.type === BT.PLANK) continue;
        setBlock(xN, y, z, BT.STAIR, { facing:'east', half:'top', shape:'straight', waterlogged:'false' });
        setBlock(-xN, y, z, BT.STAIR, { facing:'west', half:'top', shape:'straight', waterlogged:'false' });
      }
    }
  }

  // --- Pass 4: longitudinal taper stairs
  function placeLongStair(x, y, z, facing) {
    if (hasHull(x, y, z)) return;
    var ex = getBlock(x, y, z);
    if (ex && (ex.type === BT.PLANK || ex.type === BT.STAIR)) return;
    setBlock(x, y, z, BT.STAIR, { facing:facing, half:'top', shape:'straight', waterlogged:'false' });
  }

  for (var y = 0; y <= maxDeckY; y++) {
    for (var z = 0; z < L; z++) {
      var hwThis = (y >= 0 && y <= maxDeckY) ? hwArr[y][z] : -1;
      if (hwThis < 0) continue;
      var hwFwd = (z+1 < L) ? hwArr[y][z+1] : -1;
      if (hwFwd >= 0 && hwFwd < hwThis) {
        for (var x = hwFwd + 1; x <= hwThis; x++) {
          placeLongStair(x, y, z+1, 'south');
          if (x !== 0) placeLongStair(-x, y, z+1, 'south');
        }
      }
      var hwBack = (z > 0) ? hwArr[y][z-1] : -1;
      if (hwBack >= 0 && hwBack < hwThis && z > 0) {
        for (var x = hwBack + 1; x <= hwThis; x++) {
          placeLongStair(x, y, z-1, 'north');
          if (x !== 0) placeLongStair(-x, y, z-1, 'north');
        }
      }
    }
  }

  // --- Pass 5: keel-rise stairs
  for (var z = 0; z < L - 1; z++) {
    var k0 = keelYArr[z], k1 = keelYArr[z+1];
    if (k1 === k0) continue;
    var dir, yFill, zFill, refZ;
    if (k1 > k0) { dir = 'bow'; yFill = k1 - 1; zFill = z + 1; refZ = z + 1; }
    else { dir = 'stern'; yFill = k0 - 1; zFill = z; refZ = z; }
    var hw = (yFill+1 >= 0 && yFill+1 <= maxDeckY && refZ >= 0 && refZ < L) ? hwArr[yFill+1][refZ] : -1;
    if (hw < 0) continue;
    var facing = dir === 'stern' ? 'north' : 'south';
    for (var x = -hw; x <= hw; x++) {
      if (hasHull(x, yFill, zFill)) continue;
      if (getBlock(x, yFill, zFill)) continue;
      setBlock(x, yFill, zFill, BT.STAIR, { facing:facing, half:'top', shape:'straight', waterlogged:'false' });
    }
  }

  // --- Pass 5.5: no stair stacking
  var stairList = [];
  for (var k in blockMap) {
    var b = blockMap[k];
    if (b.type === BT.STAIR) stairList.push(b);
  }
  stairList.sort(function(a, b) { return b.y - a.y; });
  for (var i = 0; i < stairList.length; i++) {
    var s = stairList[i];
    var below = getBlock(s.x, s.y - 1, s.z);
    if (below && below.type === BT.STAIR) setBlock(s.x, s.y, s.z, BT.PLANK);
  }

  // --- Pass 6: stern windows
  if (hasWindows && castleHeight >= 2 && castleLength > 0) {
    var wy = D + 1;
    if (deckYArr[0] > D) {
      var hwBack = (wy >= 0 && wy <= maxDeckY) ? hwArr[wy][0] : -1;
      if (hwBack >= 1) {
        for (var x = -hwBack + 1; x <= hwBack - 1; x += 2) {
          var ex = getBlock(x, wy, 0);
          if (ex && ex.type === BT.PLANK) {
            setBlock(x, wy, 0, BT.TRAPDOOR, { facing:'north', half:'bottom', open:'true', powered:'false', waterlogged:'false' });
          }
        }
      }
    }
  }

  // --- Pass 8/9: trim + railings
  for (var z = 0; z < L; z++) {
    var dY = deckYArr[z];
    var hw = (dY >= 0 && dY <= maxDeckY) ? hwArr[dY][z] : -1;
    if (hw < 1) continue;
    var y = dY + 1;
    var canInset = hw >= 2;

    if (hasTrim && hasRailings && canInset) {
      if (!getBlock(hw, y, z) && !hasHull(hw, y, z))
        setBlock(hw, y, z, BT.SLAB_BOT, { type:'bottom', waterlogged:'false' });
      if (!getBlock(-hw, y, z) && !hasHull(-hw, y, z))
        setBlock(-hw, y, z, BT.SLAB_BOT, { type:'bottom', waterlogged:'false' });
      setBlock(hw-1, y, z, BT.FENCE, { north:'false', south:'false', east:'false', west:'false', waterlogged:'false' });
      if (hw-1 > 0)
        setBlock(-(hw-1), y, z, BT.FENCE, { north:'false', south:'false', east:'false', west:'false', waterlogged:'false' });
    } else if (hasTrim && !hasRailings) {
      if (!getBlock(hw, y, z) && !hasHull(hw, y, z))
        setBlock(hw, y, z, BT.SLAB_BOT, { type:'bottom', waterlogged:'false' });
      if (hw > 0 && !getBlock(-hw, y, z) && !hasHull(-hw, y, z))
        setBlock(-hw, y, z, BT.SLAB_BOT, { type:'bottom', waterlogged:'false' });
    } else if (hasRailings) {
      setBlock(hw, y, z, BT.FENCE, { north:'false', south:'false', east:'false', west:'false', waterlogged:'false' });
      if (hw > 0)
        setBlock(-hw, y, z, BT.FENCE, { north:'false', south:'false', east:'false', west:'false', waterlogged:'false' });
    }
  }

  // Defensive: no fence over slab
  for (var k in blockMap) {
    var b = blockMap[k];
    if (b.type !== BT.FENCE) continue;
    var below = getBlock(b.x, b.y - 1, b.z);
    if (below && below.type === BT.SLAB_BOT) delete blockMap[bKey(b.x, b.y - 1, b.z)];
  }

  // --- Pass 10: gun ports
  if (hasGunPorts && gunPortRow > 0) {
    var midKeelY = keelYAt(Math.floor(L / 2));
    var yPort = D - gunPortRow;
    if (midKeelY + 1 > yPort) yPort = midKeelY + 1;
    for (var z = 3; z < L - 3; z += gunPortSpacing) {
      var hw = (yPort >= 0 && yPort <= maxDeckY) ? hwArr[yPort][z] : -1;
      if (hw < 1) continue;
      setBlock(hw, yPort, z, BT.TRAPDOOR, { facing:'east', half:'bottom', open:'true', powered:'false', waterlogged:'false' });
      if (hw > 0)
        setBlock(-hw, yPort, z, BT.TRAPDOOR, { facing:'west', half:'bottom', open:'true', powered:'false', waterlogged:'false' });
    }
  }

  // --- Pass 10.5: fence bridging
  var isSupport = function(b) {
    return b && (b.type === BT.PLANK || b.type === BT.FENCE || b.type === BT.STAIR || b.type === BT.SLAB_BOT);
  };
  var addBridge = function(x, y, z) {
    if (blockMap[bKey(x, y, z)]) return false;
    if (!isSupport(getBlock(x, y - 1, z))) return false;
    setBlock(x, y, z, BT.FENCE, { north:'false', south:'false', east:'false', west:'false', waterlogged:'false' });
    return true;
  };
  var fences = [];
  for (var k in blockMap) { if (blockMap[k].type === BT.FENCE) fences.push(blockMap[k]); }
  for (var fi = 0; fi < fences.length; fi++) {
    var f = fences[fi];
    for (var dzi = 0; dzi < 2; dzi++) {
      var dz = dzi === 0 ? -1 : 1;
      var direct = getBlock(f.x, f.y, f.z + dz);
      if (direct && direct.type === BT.FENCE) continue;
      var dyVals = [-2, -1, 0, 1, 2];
      var dxVals = [-1, 0, 1];
      for (var dyi = 0; dyi < dyVals.length; dyi++) {
        for (var dxi = 0; dxi < dxVals.length; dxi++) {
          var dy = dyVals[dyi], dx = dxVals[dxi];
          if (dx === 0 && dy === 0) continue;
          var n = getBlock(f.x + dx, f.y + dy, f.z + dz);
          if (!n || n.type !== BT.FENCE) continue;
          var commonY = Math.max(f.y, n.y);
          if (dx !== 0) addBridge(f.x + dx, commonY, f.z);
          if (dy !== 0) {
            for (var yStack = n.y + 1; yStack <= commonY; yStack++) addBridge(n.x, yStack, n.z);
          }
        }
      }
    }
  }

  // --- Pass 11: fence connections
  function checkDir(bx, by, bz, dx, dz) {
    var n = getBlock(bx + dx, by, bz + dz);
    if (!n) return 'false';
    return (n.type === BT.FENCE || n.type === BT.PLANK || n.type === BT.STAIR) ? 'true' : 'false';
  }
  for (var k in blockMap) {
    var b = blockMap[k];
    if (b.type !== BT.FENCE) continue;
    b.props = {
      east: checkDir(b.x, b.y, b.z, 1, 0),
      west: checkDir(b.x, b.y, b.z, -1, 0),
      south: checkDir(b.x, b.y, b.z, 0, 1),
      north: checkDir(b.x, b.y, b.z, 0, -1),
      waterlogged: 'false'
    };
  }

  // --- Pass 12: stair shapes
  var facingVec = { south:[0,1], north:[0,-1], east:[1,0], west:[-1,0] };
  var facingLeft = { south:'east', north:'west', east:'north', west:'south' };
  var facingRight = { south:'west', north:'east', east:'south', west:'north' };

  for (var k in blockMap) {
    var b = blockMap[k];
    if (b.type !== BT.STAIR) continue;
    var facing = b.props.facing;
    var fwd = facingVec[facing];
    var front = getBlock(b.x + fwd[0], b.y, b.z + fwd[1]);
    var back = getBlock(b.x - fwd[0], b.y, b.z - fwd[1]);
    if (back && back.type === BT.STAIR && back.props.half === b.props.half) {
      if (back.props.facing === facingLeft[facing]) { b.props.shape = 'inner_left'; continue; }
      if (back.props.facing === facingRight[facing]) { b.props.shape = 'inner_right'; continue; }
    }
    if (front && front.type === BT.STAIR && front.props.half === b.props.half) {
      if (front.props.facing === facingLeft[facing]) { b.props.shape = 'outer_left'; continue; }
      if (front.props.facing === facingRight[facing]) { b.props.shape = 'outer_right'; continue; }
    }
  }

  // --- Normalize
  var minX = 1e9, minY = 1e9, minZ = 1e9;
  var maxX = -1e9, maxY = -1e9, maxZ = -1e9;
  var result = [];
  for (var k in blockMap) {
    var b = blockMap[k];
    result.push(b);
    if (b.x < minX) minX = b.x;
    if (b.y < minY) minY = b.y;
    if (b.z < minZ) minZ = b.z;
    if (b.x > maxX) maxX = b.x;
    if (b.y > maxY) maxY = b.y;
    if (b.z > maxZ) maxZ = b.z;
  }
  if (result.length === 0) { minX = 0; minY = 0; minZ = 0; maxX = 0; maxY = 0; maxZ = 0; }
  for (var i = 0; i < result.length; i++) {
    result[i].x -= minX;
    result[i].y -= minY;
    result[i].z -= minZ;
  }

  return {
    blocks: result,
    sizeX: maxX - minX + 1,
    sizeY: maxY - minY + 1,
    sizeZ: maxZ - minZ + 1,
    materials: { woodType: p.woodType || 'spruce' }
  };
}

// Expose
window.GeneratorEngine = {
  propeller: generatePropeller,
  balloon: generateBalloon,
  hull: generateHull
};

})();
