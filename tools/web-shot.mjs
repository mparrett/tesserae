// web-shot.mjs — headless-browser check of the browser build.
//   node tools/web-shot.mjs <url> <out-prefix> [presses]
// Loads the page (served by tools/serve-web.sh), waits for the title screen,
// screenshots it, presses Space <presses> times (prologue -> card -> level 1),
// screenshots again, and counts frames (writes that carry a DEC 2026 sync
// begin) over 5 s on each screen. Prints a JSON summary.
// Playwright comes from PW_MODULE (default: the joint-xsofy probe install).
const pwPath = process.env.PW_MODULE ||
  `${process.env.HOME}/projects-new/joint-xsofy/local-scripts/browser-smoke-playwright/node_modules/playwright/index.mjs`;
const { chromium } = await import(pwPath);
const [url, prefix, pressesArg] = process.argv.slice(2);
const presses = parseInt(pressesArg || '3');
const W = parseInt(process.env.VW || '1400'), H = parseInt(process.env.VH || '900');
const browser = await chromium.launch({ args: ['--use-angle=swiftshader', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: W, height: H } });
const errs = [], logs = [];
page.on('pageerror', e => errs.push(String(e).split('\n')[0]));
page.on('console', m => { const t = m.text(); if (m.type() === 'error') errs.push(t.slice(0, 200)); else logs.push(t.slice(0, 200)); });
const out = { url, viewport: [W, H] };
const screenText = () => page.evaluate(() => {
  const t = window._tessTerm; if (!t) return '';
  const b = t.buffer.active, rows = [];
  for (let y = 0; y < t.rows; y++) rows.push(b.getLine(y)?.translateToString(true) ?? '');
  return rows.join('\n');
});
async function fps(ms = 5000) {
  await page.evaluate(() => { window._frames = 0; window._bytes = 0; });
  await page.waitForTimeout(ms);
  const [f, b] = await page.evaluate(() => [window._frames, window._bytes]);
  return { fps: +(f * 1000 / ms).toFixed(1), kbPerSec: +(b / 1024 * 1000 / ms).toFixed(0) };
}
try {
  const t0 = Date.now();
  await page.goto(url, { waitUntil: 'domcontentloaded' });
  out.isolated = await page.evaluate(() => self.crossOriginIsolated);
  await page.waitForFunction(() => window._tessTerm, null, { timeout: 60000 });
  await page.evaluate(() => {
    const t = window._tessTerm, w = t.write.bind(t);
    window._frames = 0; window._bytes = 0;
    t.write = (s, cb) => { if (typeof s === 'string') { if (s.includes('\x1b[?2026h')) window._frames++; window._bytes += s.length; } return w(s, cb); };
  });
  await page.waitForFunction(() => window._frames > 5, null, { timeout: 60000 });
  out.bootMs = Date.now() - t0;
  out.grid = await page.evaluate(() => [window._tessTerm.cols, window._tessTerm.rows, window._tessTerm.options.fontSize, window._tessRenderer]);
  await page.waitForTimeout(2000);
  out.title = await fps();
  out.titleText = (await screenText()).split('\n').filter(l => l.trim()).slice(0, 4).map(l => l.trim().slice(0, 80));
  await page.screenshot({ path: `${prefix}-title.png` });
  for (let i = 0; i < presses; i++) { await page.keyboard.press('Space'); await page.waitForTimeout(i === presses - 1 ? 3000 : 2500); }
  out.play = await fps();
  await page.screenshot({ path: `${prefix}-play.png` });
  out.playText = (await screenText()).split('\n').filter(l => /score|spark|lives|fps|FIRST|1\.1/i.test(l)).slice(0, 4).map(l => l.trim().slice(0, 100));
} catch (e) { out.error = String(e).split('\n')[0]; await page.screenshot({ path: `${prefix}-error.png` }).catch(() => {}); }
out.errors = [...new Set(errs)].slice(0, 5);
out.logs = [...new Set(logs)].slice(0, 5);
console.log(JSON.stringify(out, null, 2));
await browser.close();
