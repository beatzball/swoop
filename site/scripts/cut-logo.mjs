// Cut the logo's raster files from public/logo.svg: logo.png (512px, which
// the share cards inline), apple-touch-icon.png, favicon-32.png and
// favicon.ico. Chromium draws the SVG, because it is already here for the
// e2e suite and draws it the way a browser will. Run from site/:
//
//   node scripts/cut-logo.mjs && cwebp -lossless public/logo.png -o public/logo.webp
import { chromium } from '@playwright/test';
import { readFileSync, writeFileSync } from 'node:fs';

const svg = readFileSync('public/logo.svg', 'utf8');
const browser = await chromium.launch();
const page = await browser.newPage();
for (const [size, out] of [
  [512, 'public/logo.png'],
  [180, 'public/apple-touch-icon.png'],
  [32, 'public/favicon-32.png'],
]) {
  await page.setViewportSize({ width: size, height: size });
  await page.setContent(
    `<body style="margin:0;background:transparent">${svg.replace('<svg ', `<svg width="${size}" height="${size}" `)}</body>`,
  );
  await page.screenshot({ path: out, omitBackground: true });
}
await browser.close();

// An .ico may hold a PNG as it is: a six-byte header, one sixteen-byte
// directory entry, then the PNG.
const png = readFileSync('public/favicon-32.png');
const ico = Buffer.alloc(22);
ico.writeUInt16LE(0, 0); // reserved
ico.writeUInt16LE(1, 2); // type: icon
ico.writeUInt16LE(1, 4); // one image
ico.writeUInt8(32, 6); // width
ico.writeUInt8(32, 7); // height
ico.writeUInt16LE(1, 10); // color planes
ico.writeUInt16LE(32, 12); // bits per pixel
ico.writeUInt32LE(png.length, 14);
ico.writeUInt32LE(22, 18); // where the PNG starts
writeFileSync('public/favicon.ico', Buffer.concat([ico, png]));
