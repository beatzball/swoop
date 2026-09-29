// Cut the logo's smaller files from public/logo.png, the owl drawing on a
// transparent ground: apple-touch-icon.png, favicon-32.png and favicon.ico.
// Chromium draws it, because it is already here for the e2e suite and
// scales the way a browser will. Run from site/:
//
//   node scripts/cut-logo.mjs && cwebp -q 92 public/logo.png -o public/logo.webp
import { chromium } from '@playwright/test';
import { readFileSync, writeFileSync } from 'node:fs';

const png = `data:image/png;base64,${readFileSync('public/logo.png').toString('base64')}`;
const browser = await chromium.launch();
const page = await browser.newPage();
for (const [size, out] of [
  [180, 'public/apple-touch-icon.png'],
  [32, 'public/favicon-32.png'],
]) {
  await page.setViewportSize({ width: size, height: size });
  await page.setContent(
    `<body style="margin:0;background:transparent"><img src="${png}" width="${size}" height="${size}" style="display:block"></body>`,
  );
  await page.screenshot({ path: out, omitBackground: true });
}
await browser.close();

// An .ico may hold a PNG as it is: a six-byte header, one sixteen-byte
// directory entry, then the PNG.
const icon = readFileSync('public/favicon-32.png');
const ico = Buffer.alloc(22);
ico.writeUInt16LE(0, 0); // reserved
ico.writeUInt16LE(1, 2); // type: icon
ico.writeUInt16LE(1, 4); // one image
ico.writeUInt8(32, 6); // width
ico.writeUInt8(32, 7); // height
ico.writeUInt16LE(1, 10); // color planes
ico.writeUInt16LE(32, 12); // bits per pixel
ico.writeUInt32LE(icon.length, 14);
ico.writeUInt32LE(22, 18); // where the PNG starts
writeFileSync('public/favicon.ico', Buffer.concat([ico, icon]));
