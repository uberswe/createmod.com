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
  var rotationDeg = clamp(p.rotation || 0, 0, 360);
  var rotationRad = rotationDeg * Math.PI / 180;
  var orientation = p.orientation === 'vertical' ? 'vertical' : 'horizontal';

  var seen = {};
  var blocks = [];

  for (var b = 0; b < blades; b++) {
    var angle = (b / blades) * 2 * Math.PI + rotationRad;
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

  // Vertical orientation: rotate XZ disc into XY plane
  if (orientation === 'vertical') {
    for (var vi = 0; vi < blocks.length; vi++) {
      blocks[vi].y = blocks[vi].z;
      blocks[vi].z = 0;
    }
  }

  // Normalize
  var minX = 1e9, minY = 1e9, minZ = 1e9, maxX = -1e9, maxY = -1e9, maxZ = -1e9;
  for (var i = 0; i < blocks.length; i++) {
    var bl = blocks[i];
    if (bl.x < minX) minX = bl.x;
    if (bl.y < minY) minY = bl.y;
    if (bl.z < minZ) minZ = bl.z;
    if (bl.x > maxX) maxX = bl.x;
    if (bl.y > maxY) maxY = bl.y;
    if (bl.z > maxZ) maxZ = bl.z;
  }
  if (blocks.length === 0) { minX = 0; minY = 0; minZ = 0; maxX = 0; maxY = 0; maxZ = 0; }
  for (var i = 0; i < blocks.length; i++) {
    blocks[i].x -= minX;
    blocks[i].y -= minY;
    blocks[i].z -= minZ;
  }

  return {
    blocks: blocks,
    sizeX: maxX - minX + 1,
    sizeY: maxY - minY + 1,
    sizeZ: maxZ - minZ + 1,
    materials: { bladeMaterial: bladeMat, bladeColor: bladeColor, orientation: orientation }
  };
}

// ===== Balloon Generator =====

function generateBalloon(p) {
  var lx = clamp(p.lengthX || 12, 6, 500);
  var wz = clamp(p.widthZ || 12, 4, 250);
  var hy = clamp(p.heightY || 16, 4, 250);
  var cylinderMid = clamp(p.cylinderMid || 0, 0, 0.85);
  var frontTaper = clamp(p.frontTaper || 0, 0, 1);
  var rearTaper = clamp(p.rearTaper || 0, 0, 1);
  var topFlatten = clamp(p.topFlatten || 0, 0, 0.5);
  var bottomFlatten = clamp(p.bottomFlatten || 0, 0, 0.5);
  var hollow = p.hollow !== false;
  var shell = clamp(p.shell || 1, 1, 5);
  var ribEnabled = !!p.ribEnabled;
  var ribSpacing = clamp(p.ribSpacing || 4, 2, 12);
  var ribOffset = clamp(p.ribOffset || 0, 0, ribSpacing - 1);
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

    // Pass 1c: thicken shell using frontier-based BFS
    if (shell > 1) {
      var frontier = {};
      for (var fk in shellKeys) {
        var fp = fk.split(',');
        var fpx = parseInt(fp[0]), fpy = parseInt(fp[1]), fpz = parseInt(fp[2]);
        for (var fdi = 0; fdi < 6; fdi++) {
          var fnb = coordKey(fpx + dirs[fdi][0], fpy + dirs[fdi][1], fpz + dirs[fdi][2]);
          if (insideKeys[fnb] && !shellKeys[fnb]) frontier[fnb] = true;
        }
      }
      for (var layer = 1; layer < shell; layer++) {
        var nextFrontier = {};
        for (var ffk in frontier) {
          shellKeys[ffk] = true;
          var ffp = ffk.split(',');
          var ffpx = parseInt(ffp[0]), ffpy = parseInt(ffp[1]), ffpz = parseInt(ffp[2]);
          for (var fdi2 = 0; fdi2 < 6; fdi2++) {
            var fnb2 = coordKey(ffpx + dirs[fdi2][0], ffpy + dirs[fdi2][1], ffpz + dirs[fdi2][2]);
            if (insideKeys[fnb2] && !shellKeys[fnb2] && !frontier[fnb2]) nextFrontier[fnb2] = true;
          }
        }
        frontier = nextFrontier;
      }
    }

    for (var sk in shellKeys) {
      grid[sk] = BT.WOOL;
    }
  }

  // Pass 3: ribbing — replace shell wool with log at rib columns, then back each rib with envelope
  if (ribEnabled && ribSpacing > 0) {
    var isRibCol = function(x) {
      return ((x - ribOffset) % ribSpacing + ribSpacing) % ribSpacing === 0;
    };
    var ribKeys = [];
    for (var gk in grid) {
      if (grid[gk] !== BT.WOOL) continue;
      var gp = gk.split(',');
      if (isRibCol(parseInt(gp[0]))) {
        grid[gk] = BT.LOG;
        ribKeys.push(gk);
      }
    }
    if (hollow) {
      for (var ri = 0; ri < ribKeys.length; ri++) {
        var rp = ribKeys[ri].split(',');
        var rx = parseInt(rp[0]), ry = parseInt(rp[1]), rz = parseInt(rp[2]);
        for (var di = 0; di < 6; di++) {
          var nbk = coordKey(rx + dirs[di][0], ry + dirs[di][1], rz + dirs[di][2]);
          if (grid[nbk] === undefined && insideKeys[nbk]) {
            grid[nbk] = BT.WOOL;
          }
        }
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

function generateHullV1(p) {
  var L = clamp(p.length || 40, 20, 500);
  var B = clamp(p.beam || 10, 4, 100);
  var D = clamp(p.depth || 6, 3, 40);
  var bottomPinch = clamp(p.bottomPinch || 0.3, 0.1, 0.7);
  var hullFlare = clamp(p.hullFlare || 0, 0, 0.6);
  var flareCurve = clamp(p.flareCurve || 2.6, 1.2, 4.0);
  var tumblehome = clamp(p.tumblehome || 0, 0, 0.4);
  var tumbleCurve = clamp(p.tumbleCurve || 3, 1.5, 5.0);
  var sheerCurve = clamp(p.sheerCurve || 0, 0, 0.75);
  var sheerCurveExp = clamp(p.sheerCurveExp || 2, 1.0, 4.0);
  var bowLength = clamp(p.bowLength || 8, 2, Math.floor(L / 2));
  var bowSharpness = clamp(p.bowSharpness || 1.3, 0.4, 4.0);
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
  var bowStyle = p.bowStyle || 'default';
  if (['default','pointed','clipper','raked','plumb'].indexOf(bowStyle) < 0) bowStyle = 'default';

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
      var base;
      switch (bowStyle) {
        case 'pointed':
          base = Math.pow(Math.max(st, 0), bowSharpness * 1.5);
          break;
        case 'clipper':
          base = st * st * st;
          if (bowCurve < 0) base = base * (1 + bowCurve) + Math.pow(st, 5) * (-bowCurve);
          break;
        case 'raked':
          var rakeT = Math.pow(st, 0.5);
          base = Math.pow(rakeT, bowSharpness * 0.8);
          break;
        case 'plumb':
          base = st < 0.15 ? st / 0.15 * 0.15 : 1.0;
          break;
        default:
          base = Math.pow(Math.max(st, 0), bowSharpness);
          if (bowCurve !== 0) {
            if (bowCurve > 0) {
              var convex = Math.sqrt(Math.max(st, 0));
              base = base * (1 - bowCurve) + convex * bowCurve;
            } else {
              var concave = st * st * st;
              base = base * (1 + bowCurve) + concave * (-bowCurve);
            }
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


// ===== Hull Generator v2 (lofted geometry) =====
// Mirrors internal/generator/hull_v2.go — keep the two in sync.

function generateHullV2(p) {
  var L = clamp(p.length || 40, 20, 500);
  var B = clamp(p.beam || 10, 4, 100);
  var D = clamp(p.depth || 6, 3, 40);
  var bottomPinch = clamp(p.bottomPinch || 0.3, 0.1, 0.7);
  var hullFlare = clamp(p.hullFlare || 0, 0, 0.6);
  var flareCurve = clamp(p.flareCurve || 2.6, 1.2, 4.0);
  var tumblehome = clamp(p.tumblehome || 0, 0, 0.4);
  var tumbleCurve = clamp(p.tumbleCurve || 3, 1.5, 5.0);
  var sheerCurve = clamp(p.sheerCurve || 0, 0, 0.75);
  var sheerCurveExp = clamp(p.sheerCurveExp || 2, 1.0, 4.0);
  var bowLength = clamp(p.bowLength || 8, 2, Math.floor(L / 2));
  var bowSharpness = clamp(p.bowSharpness || 1.3, 0.4, 4.0);
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
  var castleLength = clamp(p.castleLength || 0, 0, 30);
  if (castleLength > Math.floor(L * 55 / 100)) castleLength = Math.floor(L * 55 / 100);
  var forecastleHeight = clamp(p.forecastleHeight || 0, 0, 3);
  var forecastleLength = clamp(p.forecastleLength || 0, 0, 20);
  if (forecastleLength > Math.floor(L * 50 / 100)) forecastleLength = Math.floor(L * 50 / 100);
  var hasGunPorts = !!p.hasGunPorts;
  var gunPortRow = clamp(p.gunPortRow || 2, 1, 6);
  var gunPortSpacing = clamp(p.gunPortSpacing || 4, 2, 8);
  var midWidthBias = clamp(p.midWidthBias || 0, 0, 1.0);
  var bowStyle = p.bowStyle;
  var validBows = { 'default':1, pointed:1, clipper:1, raked:1, plumb:1 };
  if (!validBows[bowStyle]) bowStyle = 'default';

  // v2 params
  var deadrise = clamp(p.deadrise || 0, 0, 0.7);
  var midFullness = clamp(p.midFullness || 0, 0, 1);
  var bowSectionV = clamp(p.bowSectionV || 0, 0, 1);
  var sternFullness = clamp(p.sternFullness || 0, 0, 1);
  var stemRake = clamp(p.stemRake || 0, 0, 1.2);
  var stemCurve = clamp(p.stemCurve || 0, -1, 1);
  var sternRake = clamp(p.sternRake || 0, 0, 1);
  var rocker = clamp(p.rocker || 0, 0, 0.5);
  var parallelMidbody = clamp(p.parallelMidbody || 0, 0, 0.6);
  var stemPostHeight = clamp(p.stemPostHeight || 0, 0, 8);
  var sternPostHeight = clamp(p.sternPostHeight || 0, 0, 8);
  var doubleEnder = !!p.doubleEnder;
  var closedHull = !!p.closedHull;
  var sweep = clamp(p.sweep || 0, 0, 2);

  // Defaults mirroring HullParams.Validate for version >= 3
  if (midFullness === 0) midFullness = 0.65;
  if (bowSectionV === 0) bowSectionV = 0.55;
  if (sternFullness === 0) sternFullness = 0.5;
  if (stemRake === 0 && stemCurve === 0) {
    if (bowStyle === 'clipper') { stemRake = 0.8; stemCurve = -0.6; }
    else if (bowStyle === 'raked') { stemRake = 0.6; }
    else if (bowStyle === 'pointed') { stemRake = 0.2; }
    else if (bowStyle !== 'plumb') { stemRake = 0.35; stemCurve = 0.15; }
  }
  if (sternRake === 0) {
    if (sternStyle === 'square') sternRake = 0.2;
    else if (sternStyle === 'round') sternRake = 0.35;
  }

  var length = L, depth = D, halfBeam = B / 2;
  function sstep(t) { t = clamp01(t); return t * t * t * (t * (t * 6 - 15) + 10); }
  function clamp01(v) { return v < 0 ? 0 : (v > 1 ? 1 : v); }
  function lerp(a, b, t) { return a + (b - a) * t; }

  // Sweep bends the whole hull — keel and deck together — up toward the
  // ends, applied as a vertical shear so sections keep their shape.
  function sweepAt(zNorm) {
    if (sweep <= 0) return 0;
    var t = Math.abs(zNorm - 0.5) * 2;
    return sweep * depth * t * t;
  }

  var stemSetbackMax = stemRake * depth;
  function stemSetbackAt(yNorm) {
    var t = 1 - clamp01(yNorm);
    var shape = t;
    if (stemCurve > 0) shape = lerp(t, t * t, stemCurve);
    else if (stemCurve < 0) shape = lerp(t, Math.sqrt(t), -stemCurve);
    return stemSetbackMax * shape;
  }
  var sternSetbackMax = sternRake * depth;
  function sternSetbackAt(yNorm) {
    if (doubleEnder) return stemSetbackAt(yNorm);
    return sternSetbackMax * (1 - clamp01(yNorm));
  }

  function keelYAtF(zNorm) {
    var rise = 0;
    if (rocker > 0) {
      var t = Math.abs(zNorm - 0.5) * 2;
      rise = rocker * depth * Math.pow(t, keelCurveVal);
    }
    if (bowKeelRise > 0 && bowKeelLength > 0) {
      var start = 1 - bowKeelLength / length;
      if (zNorm > start) {
        var tb = (zNorm - start) / Math.max(1 - start, 0.001);
        var r = Math.pow(tb, keelCurveVal) * bowKeelRise * depth;
        if (r > rise) rise = r;
      }
    }
    var sRise = doubleEnder ? bowKeelRise : sternKeelRise;
    var sLen = doubleEnder ? bowKeelLength : sternKeelLength;
    if (sRise > 0 && sLen > 0) {
      var end = sLen / length;
      if (zNorm < end) {
        var ts = (end - zNorm) / Math.max(end, 0.001);
        var r2 = Math.pow(ts, keelCurveVal) * sRise * depth;
        if (r2 > rise) rise = r2;
      }
    }
    return rise;
  }

  var bowLenF = bowLength, sternLenF = doubleEnder ? bowLength : sternLength;
  var midLo = sternLenF / length;
  var midHi = 1 - bowLenF / length;
  var midCenter = lerp(midLo, midHi, 0.5 - midWidthBias * 0.35);
  var pmHalf = parallelMidbody / 2;
  var fullLo = Math.max(midLo, midCenter - pmHalf);
  var fullHi = Math.min(midHi, midCenter + pmHalf);

  function planAt(zNorm, yNorm) {
    if (zNorm < midLo) {
      var t = zNorm / Math.max(midLo, 0.001);
      var st = sstep(t);
      if (doubleEnder) return Math.pow(st, bowSharpness);
      if (sternStyle === 'square') {
        // Flat transom above the waterline only; the run tapers underneath.
        var f = Math.pow(st, sternSharpness);
        var floorF = lerp(0.12, 0.72, sstep(clamp01(yNorm)));
        return f < floorF ? floorF : f;
      }
      if (sternStyle === 'round') return Math.pow(st, sternSharpness * 0.55);
      return Math.pow(st, sternSharpness);
    }
    if (zNorm > midHi) {
      var t2 = (1 - zNorm) / Math.max(1 - midHi, 0.001);
      var st2 = sstep(t2);
      var base = Math.pow(st2, bowSharpness);
      if (bowCurve > 0) base = lerp(base, Math.sqrt(st2), bowCurve);
      else if (bowCurve < 0) base = lerp(base, st2 * st2 * st2, -bowCurve);
      return base;
    }
    if (zNorm >= fullLo && zNorm <= fullHi) return 1;
    if (zNorm < fullLo) {
      var t3 = (zNorm - midLo) / Math.max(fullLo - midLo, 0.001);
      return lerp(0.78, 1, sstep(t3));
    }
    var t4 = (midHi - zNorm) / Math.max(midHi - fullHi, 0.001);
    return lerp(0.84, 1, sstep(t4));
  }

  function fullnessToK(f) { return lerp(1.6, 0.55, clamp01(f)); }
  function sectionKAt(zNorm) {
    var midK = fullnessToK(midFullness);
    var bowK = midK + bowSectionV * 0.9;
    var sternK = doubleEnder ? bowK : fullnessToK(lerp(sternFullness, midFullness, 0.35));
    if (zNorm > midHi) {
      var t = (zNorm - midHi) / Math.max(1 - midHi, 0.001);
      return lerp(midK, bowK, sstep(t));
    }
    if (zNorm < midLo) {
      var t2 = (midLo - zNorm) / Math.max(midLo, 0.001);
      return lerp(midK, sternK, sstep(t2));
    }
    return midK;
  }

  var keelHalf = bottomPinch * 0.25;
  function sectionAt(yNorm, k) {
    var yc = clamp01(yNorm);
    var body = Math.pow(Math.sin(yc * Math.PI / 2), k + deadrise * 0.8);
    var base = keelHalf + (1 - keelHalf) * body;
    var flare = hullFlare * Math.pow(yc, flareCurve);
    var tumble = tumblehome * Math.pow(yc, tumbleCurve);
    var above = yNorm - 1;
    if (above < 0) above = 0;
    var castleTaper = above * 0.32 + above * above * 0.18;
    var r = base + flare - tumble - castleTaper;
    return r < 0.06 ? 0.06 : r;
  }

  // deckYAtFloat: continuous deck surface so the fitter can express sheer
  // with slabs. Mirrors Go deckYAtFloat.
  function deckYAtFloat(zf) {
    var y = depth;
    var zNorm = zf / Math.max(length - 1, 1);
    if (sheerCurve > 0) {
      var t = Math.abs(zNorm - 0.5) * 2;
      y += sheerCurve * depth * Math.pow(t, sheerCurveExp);
    }
    if (castleHeight > 0 && castleLength > 0) {
      var cL = castleLength;
      var blend = castleBlend;
      var b1 = cL * 0.55;
      if (b1 < blend) blend = b1;
      if (blend < 2) blend = 2;
      if (zf < cL - blend) y += castleHeight;
      else if (zf < cL) {
        var tc = (zf - (cL - blend)) / blend;
        y += castleHeight * (1 - sstep(tc));
      }
    }
    if (forecastleHeight > 0 && forecastleLength > 0) {
      var fL = forecastleLength;
      var blend2 = castleBlend;
      var b2 = fL * 0.55;
      if (b2 < blend2) blend2 = b2;
      if (blend2 < 2) blend2 = 2;
      var zFromBow = (length - 1) - zf;
      if (zFromBow < fL - blend2) y += forecastleHeight;
      else if (zFromBow < fL) {
        var tf = (zFromBow - (fL - blend2)) / blend2;
        y += forecastleHeight * (1 - sstep(tf));
      }
    }
    return y;
  }

  // Continuous hull volume test. Mirrors Go insideAt.
  function insideAt(xs, ys, zs) {
    if (zs < -0.49 || zs > length - 0.51) return false;
    var zNormBase = zs / Math.max(length - 1, 1);
    if (zNormBase < 0) zNormBase = 0;
    if (zNormBase > 1) zNormBase = 1;
    var keelY = keelYAtF(zNormBase);
    if (ys < keelY) return false;
    // Loft sections between keel line and deck (mirrors Go).
    var bottomSpan = depth - keelY;
    if (bottomSpan < 1) bottomSpan = 1;
    var yNorm;
    if (closedHull) {
      if (ys > 2 * depth) return false;
      if (ys <= depth) yNorm = (ys - keelY) / bottomSpan;
      else yNorm = (2 * depth - ys) / depth;
      if (yNorm < 0) return false;
    } else {
      if (ys > deckYAtFloat(zs)) return false;
      yNorm = (ys - keelY) / bottomSpan;
    }
    var sb = sternSetbackAt(yNorm);
    var stk = stemSetbackAt(yNorm);
    var zLo = sb, zHi = length - 1 - stk;
    if (zHi <= zLo || zs < zLo || zs > zHi) return false;
    var zN = clamp01((zs - zLo) / (zHi - zLo));
    var w = planAt(zN, yNorm) * sectionAt(yNorm, sectionKAt(zN)) * halfBeam;
    if (w < 0.15) return false;
    return Math.abs(xs) <= w;
  }

  // --- Grid extents ---
  var maxDeckY = D;
  var deckYArr = new Array(L), keelYArr = new Array(L);
  for (var z0 = 0; z0 < L; z0++) {
    var zn0 = z0 / Math.max(length - 1, 1);
    var sw0 = sweepAt(zn0);
    deckYArr[z0] = Math.round(deckYAtFloat(z0) + sw0);
    if (deckYArr[z0] > maxDeckY) maxDeckY = deckYArr[z0];
    keelYArr[z0] = Math.round(keelYAtF(zn0) + sw0);
  }
  var topY = closedHull ? 2 * D + Math.round(sweepAt(0)) : maxDeckY;

  // --- Column-quantized half-widths (mirrors Go hwRowAt) ---
  // Continuous half-width at integer row (y,z), or -1 when the row is
  // outside the hull. Quantization happens on whole rows — never per cell —
  // so every surface line stays coherent.
  function hwRowAt(y, z) {
    if (z < 0 || z >= L || y < 0 || y > topY) return -1;
    if (y < keelYArr[z]) return -1;
    if (!closedHull && y > deckYArr[z]) return -1;
    var ys = y, zs = z;
    var zNormBase = clamp01(zs / Math.max(length - 1, 1));
    // Undo the sweep shear so the loft below sees an unbent hull.
    ys -= sweepAt(zNormBase);
    var keelY = keelYAtF(zNormBase);
    var bottomSpan = depth - keelY;
    if (bottomSpan < 1) bottomSpan = 1;
    // The rounded keel and deck rows clamp to the continuous heights so the
    // rows the grid keeps get real loft widths.
    var yNorm;
    if (closedHull) {
      if (ys < keelY) ys = keelY;
      if (ys <= depth) yNorm = (ys - keelY) / bottomSpan;
      else yNorm = (2 * depth - ys) / depth;
      if (yNorm < 0) return -1;
    } else {
      var d = deckYAtFloat(zs);
      if (ys > d) ys = d;
      if (ys < keelY) ys = keelY;
      yNorm = (ys - keelY) / bottomSpan;
    }
    var sb = sternSetbackAt(yNorm);
    var stk = stemSetbackAt(yNorm);
    var zLo = sb, zHi = length - 1 - stk;
    if (zHi <= zLo || zs < zLo || zs > zHi) return -1;
    var zN = clamp01((zs - zLo) / (zHi - zLo));
    var w = planAt(zN, yNorm) * sectionAt(yNorm, sectionKAt(zN)) * halfBeam;
    if (w < 0.15) return -1;
    return w;
  }

  var rawHW = new Array(topY + 1);
  for (var ry = 0; ry <= topY; ry++) {
    rawHW[ry] = new Float64Array(L);
    for (var rz = 0; rz < L; rz++) rawHW[ry][rz] = hwRowAt(ry, rz);
  }

  // Fair the entrance and run along z (v1's smoothing).
  var bowStartZ = L - bowLength;
  var sternEndZ = doubleEnder ? bowLength : sternLength;
  for (var sy = 0; sy <= topY; sy++) {
    var smRow = new Float64Array(L);
    smRow.set(rawHW[sy]);
    for (var sz = 1; sz < L - 1; sz++) {
      if (sz >= sternEndZ && sz <= bowStartZ) continue;
      var prevW = rawHW[sy][sz - 1], curW = rawHW[sy][sz], nextW = rawHW[sy][sz + 1];
      if (curW < 0) continue;
      if (prevW < 0) prevW = curW;
      if (nextW < 0) nextW = curW;
      smRow[sz] = prevW * 0.25 + curW * 0.5 + nextW * 0.25;
    }
    rawHW[sy] = smRow;
  }

  // Quantize rows, then remove single-row spikes: a lone ±1 outlier between
  // equal neighbours (along z or along y) is rounding noise, not shape.
  var hwArr = new Array(topY + 1);
  for (var qy = 0; qy <= topY; qy++) {
    hwArr[qy] = new Int32Array(L);
    for (var qz = 0; qz < L; qz++) {
      if (rawHW[qy][qz] < 0) hwArr[qy][qz] = -1;
      else {
        var q = Math.round(rawHW[qy][qz] - 0.0001);
        hwArr[qy][qz] = q < 0 ? 0 : q;
      }
    }
  }
  for (var fy2 = 0; fy2 <= topY; fy2++) {
    for (var fz2 = 1; fz2 < L - 1; fz2++) {
      var a1 = hwArr[fy2][fz2 - 1], b1 = hwArr[fy2][fz2], c1 = hwArr[fy2][fz2 + 1];
      if (a1 >= 0 && b1 >= 0 && a1 === c1 && (b1 === a1 + 1 || b1 === a1 - 1)) hwArr[fy2][fz2] = a1;
    }
  }
  for (var fz3 = 0; fz3 < L; fz3++) {
    for (var fy3 = 1; fy3 < topY; fy3++) {
      var a2 = hwArr[fy3 - 1][fz3], b2 = hwArr[fy3][fz3], c2 = hwArr[fy3 + 1][fz3];
      if (a2 >= 0 && b2 >= 0 && a2 === c2 && (b2 === a2 + 1 || b2 === a2 - 1)) hwArr[fy3][fz3] = a2;
    }
  }

  function hasHull(x, y, z) {
    if (y < 0 || y > topY || z < 0 || z >= L) return false;
    var hw = hwArr[y][z];
    return hw >= 0 && x >= -hw && x <= hw;
  }

  var blockMap = {};
  function bKey(x, y, z) { return x + ',' + y + ',' + z; }
  function setBlock(x, y, z, type, props) {
    var b = { x: x, y: y, z: z, type: type };
    if (props) b.props = props;
    blockMap[bKey(x, y, z)] = b;
  }
  function getBlock(x, y, z) { return blockMap[bKey(x, y, z)]; }
  function copyProps(m) {
    if (!m) return null;
    var out = {};
    for (var k in m) out[k] = m[k];
    return out;
  }

  // --- Shell + deck ---
  // Keep only the shell (a face exposed to non-hull) plus the deck row.
  for (var sz3 = 0; sz3 < L; sz3++) {
    for (var sy3 = 0; sy3 <= topY; sy3++) {
      var shw = hwArr[sy3][sz3];
      if (shw < 0) continue;
      for (var sx3 = -shw; sx3 <= shw; sx3++) {
        var exposed = !hasHull(sx3 - 1, sy3, sz3) || !hasHull(sx3 + 1, sy3, sz3) ||
                      !hasHull(sx3, sy3 - 1, sz3) || !hasHull(sx3, sy3 + 1, sz3) ||
                      !hasHull(sx3, sy3, sz3 - 1) || !hasHull(sx3, sy3, sz3 + 1);
        var isDeck = !closedHull && sy3 === deckYArr[sz3];
        if (exposed || isDeck) setBlock(sx3, sy3, sz3, BT.PLANK);
      }
    }
  }

  // --- Step smoothing (mirrors Go) ---

  // One chamfer rule covers flare underhangs, bow/stern rakes and keel
  // rises: an empty cell directly under hull that also touches hull on a
  // horizontal face gets a top-half stair. Requiring the horizontal
  // neighbour keeps stairs seated in real step corners — no teeth hanging
  // from a face above. Facing (this codebase: the stair's LOW/open side)
  // points away from the supporting neighbour; in the bow/stern tapers
  // fore-aft support wins so the stem reads as one stepped line.
  var maxHW = 0;
  for (var my = 0; my <= topY; my++)
    for (var mz = 0; mz < L; mz++)
      if (hwArr[my][mz] > maxHW) maxHW = hwArr[my][mz];
  var inTaper = new Array(L);
  for (var itz = 0; itz < L; itz++) {
    var itn = itz / Math.max(length - 1, 1);
    inTaper[itz] = itn < midLo || itn > midHi;
  }
  function chamferFacing(x, y, z) {
    var n = hasHull(x, y, z - 1);
    var s = hasHull(x, y, z + 1);
    var w = hasHull(x - 1, y, z);
    var e = hasHull(x + 1, y, z);
    var lateral = '';
    if (x >= 0 && w) lateral = 'east';
    else if (x <= 0 && e) lateral = 'west';
    else if (w) lateral = 'east';
    else if (e) lateral = 'west';
    var foreAft = '';
    if (n) foreAft = 'south';
    else if (s) foreAft = 'north';
    if (inTaper[z]) return foreAft !== '' ? foreAft : lateral;
    return lateral !== '' ? lateral : foreAft;
  }
  for (var cz = 0; cz < L; cz++) {
    for (var cy = 0; cy < topY; cy++) {
      for (var cxx = -maxHW - 1; cxx <= maxHW + 1; cxx++) {
        if (hasHull(cxx, cy, cz) || !hasHull(cxx, cy + 1, cz)) continue;
        if (getBlock(cxx, cy, cz)) continue;
        var cf = chamferFacing(cxx, cy, cz);
        if (cf === '') continue;
        setBlock(cxx, cy, cz, BT.STAIR, { facing: cf, half: 'top', shape: 'straight', waterlogged: 'false' });
      }
    }
  }

  // Ledge caps: an empty cell directly on top of hull with horizontal hull
  // support gets a bottom-half stair, smoothing tumblehome ledges, castle
  // walls and the deck breaks a strong sweep or sheer produces. Above the
  // column's own deck only fore-aft caps are allowed — the gunwale edge
  // belongs to trim and railings.
  for (var lz2 = 0; lz2 < L; lz2++) {
    for (var ly2 = 1; ly2 <= topY; ly2++) {
      for (var lx2 = -maxHW - 1; lx2 <= maxHW + 1; lx2++) {
        if (hasHull(lx2, ly2, lz2) || !hasHull(lx2, ly2 - 1, lz2)) continue;
        if (getBlock(lx2, ly2, lz2)) continue;
        var aboveDeck = !closedHull && ly2 > deckYArr[lz2];
        var lf = chamferFacing(lx2, ly2, lz2);
        if (lf === '') continue;
        if (aboveDeck && lf !== 'south' && lf !== 'north') continue;
        setBlock(lx2, ly2, lz2, BT.STAIR, { facing: lf, half: 'bottom', shape: 'straight', waterlogged: 'false' });
      }
    }
  }

  // De-stack: same-facing same-half stair runs become planks above the
  // lowest stair (mirrors Go; v1's proven rule).
  {
    var stairList = [];
    for (var dk in blockMap) {
      if (blockMap[dk].type === BT.STAIR) stairList.push(blockMap[dk]);
    }
    stairList.sort(function (a, b) {
      if (a.y !== b.y) return b.y - a.y;
      if (a.z !== b.z) return a.z - b.z;
      return a.x - b.x;
    });
    for (var di2 = 0; di2 < stairList.length; di2++) {
      var st2 = stairList[di2];
      var cur2 = getBlock(st2.x, st2.y, st2.z);
      var below2 = getBlock(st2.x, st2.y - 1, st2.z);
      if (cur2 && cur2.type === BT.STAIR && below2 && below2.type === BT.STAIR &&
          below2.props && cur2.props &&
          below2.props.half === cur2.props.half && below2.props.facing === cur2.props.facing) {
        setBlock(st2.x, st2.y, st2.z, BT.PLANK);
      }
    }
  }

  function hwAt(y, z) {
    if (y < 0 || y > topY || z < 0 || z >= L) return -1;
    return hwArr[y][z];
  }
  function hKey(x, y, z) { return x + ',' + y + ',' + z; }
  var inHull = {};
  for (var hz = 0; hz < L; hz++)
    for (var hy = 0; hy <= topY; hy++) {
      var ihw = hwArr[hy][hz];
      for (var hx = -ihw; hx <= ihw; hx++) inHull[hKey(hx, hy, hz)] = 1;
    }


  // Stem/stern posts
  if (!closedHull) {
    var placePost = function (fromBow, height) {
      if (height <= 0) return;
      var zStart = fromBow ? L - 1 : 0;
      var zEnd = fromBow ? -1 : L;
      var step = fromBow ? -1 : 1;
      var zPost = -1;
      for (var zp = zStart; zp !== zEnd; zp += step) {
        if (hwAt(deckYArr[zp], zp) >= 0) { zPost = zp; break; }
      }
      if (zPost < 0) return;
      var deckY = deckYArr[zPost];
      for (var yp = deckY + 1; yp <= deckY + height; yp++) setBlock(0, yp, zPost, BT.PLANK);
      setBlock(0, deckY + height + 1, zPost, BT.STAIR,
        { facing: fromBow ? 'south' : 'north', half:'bottom', shape:'straight', waterlogged:'false' });
    };
    placePost(true, stemPostHeight);
    var sp = sternPostHeight;
    if (doubleEnder && sp === 0) sp = stemPostHeight;
    placePost(false, sp);
  }

  // Stern windows
  if (hasWindows && castleHeight >= 2 && castleLength > 0 && !closedHull) {
    var wz = 0, wy = D + 1;
    if (deckYArr[wz] > D) {
      var hwBack = hwAt(wy, wz);
      if (hwBack >= 1) {
        for (var wx = -hwBack + 1; wx <= hwBack - 1; wx += 2) {
          var wb = getBlock(wx, wy, wz);
          if (wb && wb.type === BT.PLANK) {
            setBlock(wx, wy, wz, BT.TRAPDOOR, { facing:'north', half:'bottom', open:'true', powered:'false', waterlogged:'false' });
          }
        }
      }
    }
  }

  // Gunwale trim + railings
  if (!closedHull) {
    for (var zg = 0; zg < L; zg++) {
      var deckYg = deckYArr[zg];
      var hwD = hwAt(deckYg, zg);
      if (hwD < 1) continue;
      var yg = deckYg + 1;
      var canInset = hwD >= 2;
      if (hasTrim && hasRailings && canInset) {
        if (!getBlock(hwD, yg, zg) && !hasHull(hwD, yg, zg)) setBlock(hwD, yg, zg, BT.SLAB_BOT, { type:'bottom', waterlogged:'false' });
        if (!getBlock(-hwD, yg, zg) && !hasHull(-hwD, yg, zg)) setBlock(-hwD, yg, zg, BT.SLAB_BOT, { type:'bottom', waterlogged:'false' });
        setBlock(hwD - 1, yg, zg, BT.FENCE);
        if (hwD - 1 > 0) setBlock(-(hwD - 1), yg, zg, BT.FENCE);
      } else if (hasTrim) {
        if (!getBlock(hwD, yg, zg) && !hasHull(hwD, yg, zg)) setBlock(hwD, yg, zg, BT.SLAB_BOT, { type:'bottom', waterlogged:'false' });
        if (hwD > 0 && !getBlock(-hwD, yg, zg) && !hasHull(-hwD, yg, zg)) setBlock(-hwD, yg, zg, BT.SLAB_BOT, { type:'bottom', waterlogged:'false' });
      } else if (hasRailings) {
        setBlock(hwD, yg, zg, BT.FENCE);
        if (hwD > 0) setBlock(-hwD, yg, zg, BT.FENCE);
      }
    }
    for (var kf in blockMap) {
      var bf = blockMap[kf];
      if (bf.type !== BT.FENCE) continue;
      var belowF = getBlock(bf.x, bf.y - 1, bf.z);
      if (belowF && belowF.type === BT.SLAB_BOT) delete blockMap[bKey(bf.x, bf.y - 1, bf.z)];
    }
  }

  // Gun ports
  if (hasGunPorts && gunPortRow > 0 && !closedHull) {
    var yPort = D - gunPortRow;
    var midKeel = keelYArr[Math.floor(L / 2)];
    if (midKeel + 1 > yPort) yPort = midKeel + 1;
    for (var zp2 = 3; zp2 < L - 3; zp2 += gunPortSpacing) {
      var hwP = hwAt(yPort, zp2);
      if (hwP < 1) continue;
      setBlock(hwP, yPort, zp2, BT.TRAPDOOR, { facing:'east', half:'bottom', open:'true', powered:'false', waterlogged:'false' });
      if (hwP > 0) setBlock(-hwP, yPort, zp2, BT.TRAPDOOR, { facing:'west', half:'bottom', open:'true', powered:'false', waterlogged:'false' });
    }
  }

  // Connectivity cleanup: keep only blocks 6-connected to the hull volume
  {
    var reached = {};
    var queue = [];
    for (var kq in inHull) queue.push(kq);
    var DIRS = [[1,0,0],[-1,0,0],[0,1,0],[0,-1,0],[0,0,1],[0,0,-1]];
    while (queue.length > 0) {
      var cur2 = queue.shift().split(',');
      var cx = +cur2[0], cy = +cur2[1], cz = +cur2[2];
      for (var di = 0; di < 6; di++) {
        var nk = (cx + DIRS[di][0]) + ',' + (cy + DIRS[di][1]) + ',' + (cz + DIRS[di][2]);
        if (reached[nk] || inHull[nk]) continue;
        if (blockMap[nk]) { reached[nk] = 1; queue.push(nk); }
      }
    }
    for (var kr in blockMap) {
      if (!inHull[kr] && !reached[kr]) delete blockMap[kr];
    }
  }

  // Fence connection states
  for (var kc in blockMap) {
    var bc = blockMap[kc];
    if (bc.type !== BT.FENCE) continue;
    var connD = function (dx, dz) {
      var n = getBlock(bc.x + dx, bc.y, bc.z + dz);
      if (!n) return 'false';
      return (n.type === BT.FENCE || n.type === BT.PLANK || n.type === BT.STAIR) ? 'true' : 'false';
    };
    bc.props = { east: connD(1, 0), west: connD(-1, 0), south: connD(0, 1), north: connD(0, -1), waterlogged: 'false' };
  }

  // Stair corner shapes
  {
    var facingVec = { south:[0,1], north:[0,-1], east:[1,0], west:[-1,0] };
    var leftOf = { south:'east', north:'west', east:'north', west:'south' };
    var rightOf = { south:'west', north:'east', east:'south', west:'north' };
    for (var kk in blockMap) {
      var bs = blockMap[kk];
      if (bs.type !== BT.STAIR || !bs.props) continue;
      var fv = facingVec[bs.props.facing];
      if (!fv) continue;
      var front = getBlock(bs.x + fv[0], bs.y, bs.z + fv[1]);
      var back = getBlock(bs.x - fv[0], bs.y, bs.z - fv[1]);
      if (back && back.type === BT.STAIR && back.props && back.props.half === bs.props.half) {
        if (back.props.facing === leftOf[bs.props.facing]) { bs.props.shape = 'inner_left'; continue; }
        if (back.props.facing === rightOf[bs.props.facing]) { bs.props.shape = 'inner_right'; continue; }
      }
      if (front && front.type === BT.STAIR && front.props && front.props.half === bs.props.half) {
        if (front.props.facing === leftOf[bs.props.facing]) { bs.props.shape = 'outer_left'; continue; }
        if (front.props.facing === rightOf[bs.props.facing]) { bs.props.shape = 'outer_right'; continue; }
      }
    }
  }

  // Emit
  var result = [];
  var minX = Infinity, minY = Infinity, minZ = Infinity;
  var maxX = -Infinity, maxY = -Infinity, maxZ = -Infinity;
  for (var ke in blockMap) {
    var be = blockMap[ke];
    result.push(be);
    if (be.x < minX) minX = be.x;
    if (be.y < minY) minY = be.y;
    if (be.z < minZ) minZ = be.z;
    if (be.x > maxX) maxX = be.x;
    if (be.y > maxY) maxY = be.y;
    if (be.z > maxZ) maxZ = be.z;
  }
  if (result.length === 0) { minX = 0; minY = 0; minZ = 0; maxX = 0; maxY = 0; maxZ = 0; }
  for (var ri = 0; ri < result.length; ri++) {
    result[ri].x -= minX;
    result[ri].y -= minY;
    result[ri].z -= minZ;
  }

  return {
    blocks: result,
    sizeX: maxX - minX + 1,
    sizeY: maxY - minY + 1,
    sizeZ: maxZ - minZ + 1,
    materials: { woodType: p.woodType || 'spruce' }
  };
}

// Version dispatch: <= 2 runs the frozen v1 algorithm (old share links must
// reproduce identical hulls); >= 3 (or missing) runs v2.
function generateHull(p) {
  var v = p && p.version ? p.version : 3;
  if (v >= 3) return generateHullV2(p);
  return generateHullV1(p);
}

// Expose
window.GeneratorEngine = {
  propeller: generatePropeller,
  balloon: generateBalloon,
  hull: generateHull
};

})();
