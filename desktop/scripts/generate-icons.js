/**
 * Generate placeholder app icons for MycelicMemory Desktop.
 * Creates a 512x512 PNG with a brain/network motif using pure Node.js (no external deps).
 * electron-builder auto-converts PNG → ICO/ICNS at package time.
 */

const zlib = require('zlib');
const fs = require('fs');
const path = require('path');

const SIZE = 512;
const CENTER = SIZE / 2;

function createPixels() {
  const pixels = Buffer.alloc(SIZE * SIZE * 4);

  for (let y = 0; y < SIZE; y++) {
    for (let x = 0; x < SIZE; x++) {
      const idx = (y * SIZE + x) * 4;
      const dx = x - CENTER;
      const dy = y - CENTER;
      const dist = Math.sqrt(dx * dx + dy * dy);
      const maxRadius = SIZE * 0.45;

      if (dist > maxRadius) {
        // Transparent outside circle
        pixels[idx] = 0;
        pixels[idx + 1] = 0;
        pixels[idx + 2] = 0;
        pixels[idx + 3] = 0;
        continue;
      }

      // Normalized distance from center (0 = center, 1 = edge)
      const nd = dist / maxRadius;

      // Gradient: teal (#14b8a6) at center → purple (#8b5cf6) at edge
      const r = Math.round(20 + (139 - 20) * nd);
      const g = Math.round(184 + (92 - 184) * nd);
      const b = Math.round(166 + (246 - 166) * nd);

      // Draw network nodes and connections
      let a = 255;

      // Inner glow ring
      if (nd > 0.80 && nd < 0.90) {
        const ringFade = 1 - Math.abs(nd - 0.85) / 0.05;
        const bright = Math.round(40 * ringFade);
        pixels[idx] = Math.min(255, r + bright);
        pixels[idx + 1] = Math.min(255, g + bright);
        pixels[idx + 2] = Math.min(255, b + bright);
        pixels[idx + 3] = a;
        continue;
      }

      // Soft edge fade
      if (nd > 0.90) {
        a = Math.round(255 * (1 - (nd - 0.90) / 0.10));
      }

      // Network nodes - 6 nodes in a hexagonal pattern + center
      const nodes = [
        { nx: CENTER, ny: CENTER },
        { nx: CENTER, ny: CENTER - maxRadius * 0.45 },
        { nx: CENTER + maxRadius * 0.39, ny: CENTER - maxRadius * 0.225 },
        { nx: CENTER + maxRadius * 0.39, ny: CENTER + maxRadius * 0.225 },
        { nx: CENTER, ny: CENTER + maxRadius * 0.45 },
        { nx: CENTER - maxRadius * 0.39, ny: CENTER + maxRadius * 0.225 },
        { nx: CENTER - maxRadius * 0.39, ny: CENTER - maxRadius * 0.225 },
      ];

      // Check if pixel is on a connection line between nodes
      let onLine = false;
      const lineWidth = SIZE * 0.012;
      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const lineDist = pointToLineDist(x, y, nodes[i].nx, nodes[i].ny, nodes[j].nx, nodes[j].ny);
          if (lineDist < lineWidth) {
            onLine = true;
            break;
          }
        }
        if (onLine) break;
      }

      // Check if pixel is on a node
      let onNode = false;
      let nodeGlow = 0;
      const nodeRadius = SIZE * 0.04;
      const glowRadius = SIZE * 0.06;
      for (const node of nodes) {
        const nodeDist = Math.sqrt((x - node.nx) ** 2 + (y - node.ny) ** 2);
        if (nodeDist < nodeRadius) {
          onNode = true;
          break;
        } else if (nodeDist < glowRadius) {
          nodeGlow = Math.max(nodeGlow, 1 - (nodeDist - nodeRadius) / (glowRadius - nodeRadius));
        }
      }

      if (onNode) {
        // Bright white node
        pixels[idx] = 255;
        pixels[idx + 1] = 255;
        pixels[idx + 2] = 255;
        pixels[idx + 3] = a;
      } else if (onLine) {
        // Semi-transparent white line
        pixels[idx] = Math.min(255, r + 120);
        pixels[idx + 1] = Math.min(255, g + 120);
        pixels[idx + 2] = Math.min(255, b + 120);
        pixels[idx + 3] = a;
      } else if (nodeGlow > 0) {
        // Node glow
        const glowAmt = Math.round(60 * nodeGlow);
        pixels[idx] = Math.min(255, r + glowAmt);
        pixels[idx + 1] = Math.min(255, g + glowAmt);
        pixels[idx + 2] = Math.min(255, b + glowAmt);
        pixels[idx + 3] = a;
      } else {
        pixels[idx] = r;
        pixels[idx + 1] = g;
        pixels[idx + 2] = b;
        pixels[idx + 3] = a;
      }
    }
  }

  return pixels;
}

function pointToLineDist(px, py, x1, y1, x2, y2) {
  const dx = x2 - x1;
  const dy = y2 - y1;
  const lenSq = dx * dx + dy * dy;
  if (lenSq === 0) return Math.sqrt((px - x1) ** 2 + (py - y1) ** 2);

  let t = ((px - x1) * dx + (py - y1) * dy) / lenSq;
  t = Math.max(0, Math.min(1, t));

  const closestX = x1 + t * dx;
  const closestY = y1 + t * dy;
  return Math.sqrt((px - closestX) ** 2 + (py - closestY) ** 2);
}

function intToBytes(n, count) {
  const bytes = [];
  for (let i = count - 1; i >= 0; i--) {
    bytes.push((n >> (i * 8)) & 0xff);
  }
  return bytes;
}

function crc32(buf) {
  let crc = 0xffffffff;
  for (let i = 0; i < buf.length; i++) {
    crc ^= buf[i];
    for (let j = 0; j < 8; j++) {
      crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0);
    }
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function createChunk(type, data) {
  const typeData = Buffer.concat([Buffer.from(type, 'ascii'), data]);
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(typeData));
  return Buffer.concat([length, typeData, crc]);
}

function createPNG(width, height, pixels) {
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);

  const ihdrData = Buffer.alloc(13);
  ihdrData.writeUInt32BE(width, 0);
  ihdrData.writeUInt32BE(height, 4);
  ihdrData[8] = 8;  // bit depth
  ihdrData[9] = 6;  // RGBA
  ihdrData[10] = 0; // compression
  ihdrData[11] = 0; // filter
  ihdrData[12] = 0; // interlace

  const ihdr = createChunk('IHDR', ihdrData);

  // Create filtered scanlines (filter byte 0 = None for each row)
  const rowSize = 1 + width * 4;
  const rawData = Buffer.alloc(height * rowSize);
  for (let y = 0; y < height; y++) {
    rawData[y * rowSize] = 0; // filter: None
    pixels.copy(rawData, y * rowSize + 1, y * width * 4, (y + 1) * width * 4);
  }

  const compressed = zlib.deflateSync(rawData, { level: 9 });
  const idat = createChunk('IDAT', compressed);
  const iend = createChunk('IEND', Buffer.alloc(0));

  return Buffer.concat([signature, ihdr, idat, iend]);
}

// Generate
console.log('Generating 512x512 icon...');
const pixels = createPixels();
const png = createPNG(SIZE, SIZE, pixels);

const outputDir = path.join(__dirname, '..', 'resources', 'icons');
fs.mkdirSync(outputDir, { recursive: true });

const outputPath = path.join(outputDir, 'icon.png');
fs.writeFileSync(outputPath, png);
console.log(`Written: ${outputPath} (${png.length} bytes)`);

// Also create a 256x256 version for Windows ICO fallback
// electron-builder handles the actual ICO creation, but having a smaller PNG helps
const SMALL = 256;
console.log('Generating 256x256 icon...');
const smallPixels = Buffer.alloc(SMALL * SMALL * 4);
for (let y = 0; y < SMALL; y++) {
  for (let x = 0; x < SMALL; x++) {
    // Simple 2x downscale with nearest neighbor
    const srcX = Math.floor(x * SIZE / SMALL);
    const srcY = Math.floor(y * SIZE / SMALL);
    const srcIdx = (srcY * SIZE + srcX) * 4;
    const dstIdx = (y * SMALL + x) * 4;
    pixels.copy(smallPixels, dstIdx, srcIdx, srcIdx + 4);
  }
}
const smallPng = createPNG(SMALL, SMALL, smallPixels);
fs.writeFileSync(path.join(outputDir, '256x256.png'), smallPng);
console.log('Done!');
