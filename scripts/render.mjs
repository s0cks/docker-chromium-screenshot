#!/usr/bin/env node
// Build the dossiers with Astro, then screenshot every slide with Playwright.
//
//   node scripts/render.mjs [options] [dossier-id ...]
//
//   --theme <light|dark|both|auto>  auto (default) uses each dossier's own theme
//   --out <dir>                     output directory (default: out, or $OUT_DIR)
//   --scale <n>                     device scale factor (default: 2)
//   --strict                        exit 1 on overflow or word-budget warnings
//   --pdf                           also write one PDF per dossier and theme
//   --skip-build                    reuse an existing dist/
//   --no-live                       skip capturing JSON/jsonnet cards
//
// Environment: CARDS_DIR, OUT_DIR, OUT_OWNER (uid:gid), CHROMIUM_EXECUTABLE,
// SERVICE_BIN.
//
// Every card is one directory under CARDS_DIR (default "cards") — its own
// card.md or layout.json/yaml/jsonnet, its own logo.*, theme.css and
// meta.json/yaml/jsonnet, all self-contained; cards/_shared/ holds anything
// shared across cards (a jsonnet library, a repo-wide theme.css). See
// docs/build-a-card.md.
//
// Serving: if the Go service binary is available (SERVICE_BIN, or
// service/bin/dossier-service next to this repo), it serves dist/ plus
// /meta, /layout and /cards/* — the same server the Docker image runs.
// Without it, a minimal built-in static server serves dist/ and /cards/*,
// but not /meta or /layout, so JSON/jsonnet cards are skipped (Markdown
// cards are unaffected either way).

import { spawn, spawnSync } from 'node:child_process';
import fs from 'node:fs';
import http from 'node:http';
import net from 'node:net';
import path from 'node:path';
import { parseArgs } from 'node:util';
import { chromium } from 'playwright';

const { values: opts, positionals: only } = parseArgs({
  allowPositionals: true,
  options: {
    theme: { type: 'string', default: 'auto' },
    out: { type: 'string', default: process.env.OUT_DIR ?? 'out' },
    scale: { type: 'string', default: '2' },
    strict: { type: 'boolean', default: false },
    pdf: { type: 'boolean', default: false },
    'skip-build': { type: 'boolean', default: false },
    live: { type: 'boolean', default: true },
    help: { type: 'boolean', short: 'h', default: false },
  },
});

if (opts.help) {
  console.log(fs.readFileSync(new URL(import.meta.url), 'utf8').split('\n').slice(1, 20).map((l) => l.replace(/^\/\/ ?/, '')).join('\n'));
  process.exit(0);
}
if (!['auto', 'light', 'dark', 'both'].includes(opts.theme)) fail(`--theme must be auto, light, dark or both (got "${opts.theme}")`);

const root = process.cwd();
const dist = path.join(root, 'dist');
const out = path.resolve(root, opts.out);
const cardsDir = process.env.CARDS_DIR ? path.resolve(process.env.CARDS_DIR) : path.join(root, 'cards');

function fail(msg) {
  console.error(`error: ${msg}`);
  process.exit(1);
}

async function freePort() {
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.listen(0, '127.0.0.1', () => {
      const { port } = srv.address();
      srv.close((err) => (err ? reject(err) : resolve(port)));
    });
  });
}

// ---------- 1. build ----------
if (!opts['skip-build']) {
  const astro = path.join(root, 'node_modules', 'astro', 'bin', 'astro.mjs');
  const bin = fs.existsSync(astro) ? [process.execPath, astro] : ['npx', '--no-install', 'astro'];
  const res = spawnSync(bin[0], [...bin.slice(1), 'build'], {
    stdio: 'inherit',
    env: { ...process.env, ASTRO_TELEMETRY_DISABLED: '1' },
  });
  if (res.status !== 0)
    fail('astro build failed');
}

const manifest = JSON.parse(fs.readFileSync(path.join(dist, 'manifest.json'), 'utf8'));
const dossiers = manifest.dossiers.filter((d) => !only.length || only.includes(d.id));
if (!dossiers.length && !only.length)
  fail('no dossiers found');

// ---------- 2. serve dist/ (+ mounts) on an ephemeral port ----------
async function startServer() {
  const port = await freePort();
  const addr = `127.0.0.1:${port}`;
  const serviceBin = process.env.SERVICE_BIN
    ?? [path.join(root, 'service', 'bin', 'dossier-service'), path.join(root, '..', 'service', 'bin', 'dossier-service')].find((p) => fs.existsSync(p));

  if (serviceBin) {
    const child = spawn(serviceBin, [], {
      env: { ...process.env, ADDR: addr, DIST_DIR: dist, CARDS_DIR: cardsDir },
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    let log = '';
    child.stdout.on('data', (d) => (log += d));
    child.stderr.on('data', (d) => (log += d));
    const base = `http://${addr}`;
    for (let i = 0; i < 50; i++) {
      try {
        const res = await fetch(`${base}/healthz`);
        if (res.ok)
          return { base, kind: 'go', hasApi: true, stop: () => child.kill() };
      } catch { /* not up yet */ }

      await new Promise((r) => setTimeout(r, 100));
    }

    child.kill();
    fail(`dossier-service did not become healthy on ${addr}:\n${log}`);
  }

  const MIME = { '.html': 'text/html; charset=utf-8', '.css': 'text/css', '.js': 'text/javascript', '.json': 'application/json', '.woff2': 'font/woff2', '.woff': 'font/woff', '.png': 'image/png', '.jpg': 'image/jpeg', '.svg': 'image/svg+xml' };
  const serveFrom = (dir, urlPath, res) => {
    const file = path.join(dir, decodeURIComponent(urlPath));
    if (!file.startsWith(dir) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) return false;
    res.writeHead(200, { 'content-type': MIME[path.extname(file)] ?? 'application/octet-stream' });
    fs.createReadStream(file).pipe(res);
    return true;
  };

  const server = http.createServer((req, res) => {
    const url = new URL(req.url, 'http://localhost');
    if (url.pathname.startsWith('/cards/') && serveFrom(cardsDir, url.pathname.slice('/cards/'.length), res))
      return;

    let file = path.join(dist, decodeURIComponent(url.pathname));
    if (!file.startsWith(dist))
      return void res.writeHead(403).end();

    if (fs.existsSync(file) && fs.statSync(file).isDirectory())
      file = path.join(file, 'index.html');

    if (!fs.existsSync(file))
      return void res.writeHead(404).end('not found');

    res.writeHead(200, { 'content-type': MIME[path.extname(file)] ?? 'application/octet-stream' });
    fs.createReadStream(file).pipe(res);
  });

  await new Promise((resolve) => server.listen(port, '127.0.0.1', resolve));
  return { base: `http://${addr}`, kind: 'fallback', hasApi: false, stop: () => server.close() };
}

const server = await startServer();
const { base } = server;
if (!server.hasApi) {
  console.warn('  ! no dossier-service binary was found (build service/ first) — JSON/jsonnet cards will be skipped');
}

// ---------- 3. capture ----------
const browser = await chromium.launch({
  headless: true,
  executablePath: process.env.CHROMIUM_EXECUTABLE || undefined,
  args: ['--no-sandbox', '--force-color-profile=srgb', '--font-render-hinting=none', '--hide-scrollbars'],
});

const report = { generated: new Date().toISOString(), files: [], problems: [] };
let problems = 0;
const problem = (msg) => {
  problems++;
  report.problems.push(msg);
  console.warn(`  ! ${msg}`);
};

const pad = (n) => String(n).padStart(2, '0');

async function captureSlide(context, url, outFile, { dossierId, index, title }) {
  const page = await context.newPage();
  await page.goto(url, { waitUntil: 'networkidle' });
  const fonts = await page.evaluate(async () => {
    await document.fonts.ready;
    const need = ['700 1em Newsreader', '400 1em "IBM Plex Sans"', '600 1em "IBM Plex Mono"'];
    return { ok: need.every((n) => document.fonts.check(n)) };
  });
  if (!fonts.ok)
    problem(`${dossierId} #${index}: bundled fonts failed to load`);

  const overflow = await page.evaluate(() =>
    [...document.querySelectorAll('[data-col]')]
      .filter((el) => el.scrollHeight > el.clientHeight + 1)
      .map((el) => `${el.dataset.col} column overflows by ${el.scrollHeight - el.clientHeight}px`),
  );
  overflow.forEach((o) => problem(`${dossierId} #${index} (${title}): ${o}`));

  await page.locator('#slide').screenshot({ path: outFile, type: 'png', animations: 'disabled' });
  await page.close();
  return overflow.length > 0;
}

try {
  for (const dossier of dossiers) {
    const themes = opts.theme === 'both' ? ['light', 'dark'] : [opts.theme === 'auto' ? dossier.theme : opts.theme];
    const { width, height } = dossier.size;
    console.log(`${dossier.id}: ${dossier.slides.length} slides, ${width}x${height}, ${themes.join(' + ')}`);
    dossier.warnings.forEach((w) => problem(`${dossier.id}: ${w}`));
    fs.mkdirSync(path.join(out, dossier.id), { recursive: true });

    for (const theme of themes) {
      const context = await browser.newContext({ viewport: { width, height }, deviceScaleFactor: Number(opts.scale), colorScheme: theme, reducedMotion: 'reduce' });
      for (const slide of dossier.slides) {
        const file = path.join(out, dossier.id, `${pad(slide.index)}.${theme}.png`);
        const overflowed = await captureSlide(context, `${base}${slide.path}?theme=${theme}`, file, { dossierId: dossier.id, index: slide.index, title: slide.title });
        report.files.push({ dossier: dossier.id, slide: slide.index, theme, title: slide.title, words: slide.words, file: path.relative(root, file), width: width * Number(opts.scale), height: height * Number(opts.scale) });
        console.log(`  ${path.relative(root, file)}${overflowed ? '  (overflow)' : ''}`);
      }

      if (opts.pdf) {
        const pdfPath = path.join(out, dossier.id, `${dossier.id}.${theme}.pdf`);
        const page = await context.newPage();
        await page.goto(`${base}/${dossier.id}/print/?theme=${theme}`, { waitUntil: 'networkidle' });
        await page.evaluate(() => document.fonts.ready);
        await page.pdf({ path: pdfPath, preferCSSPageSize: true, printBackground: true });
        await page.close();
        report.files.push({ dossier: dossier.id, theme, file: path.relative(root, pdfPath), kind: 'pdf' });
        console.log(`  ${path.relative(root, pdfPath)}`);
      }

      await context.close();
    }
  }

  if (opts.live && server.hasApi) {
    const { layouts } = await fetch(`${base}/layouts`).then((r) => r.json());
    const ids = layouts.filter((id) => !only.length || only.includes(id));
    for (const id of ids) {
      const layout = await fetch(`${base}/layout/${id}`).then((r) => r.json());
      if (!layout?.slides?.length) {
        problem(`${id}: layout has no slides`);
        continue;
      }

      const themeDefault = layout.meta?.theme === 'dark' ? 'dark' : 'light';
      const themes = opts.theme === 'both' ? ['light', 'dark'] : [opts.theme === 'auto' ? themeDefault : opts.theme];
      const [width, height] = sizeOf(layout.meta?.size);
      console.log(`${id} (live): ${layout.slides.length} slides, ${width}x${height}, ${themes.join(' + ')}`);
      fs.mkdirSync(path.join(out, id), { recursive: true });

      for (const theme of themes) {
        const context = await browser.newContext({ viewport: { width, height }, deviceScaleFactor: Number(opts.scale), colorScheme: theme, reducedMotion: 'reduce' });
        for (let i = 0; i < layout.slides.length; i++) {
          const file = path.join(out, id, `${pad(i + 1)}.${theme}.png`);
          const url = `${base}/live/?dossier=${encodeURIComponent(id)}&slide=${i + 1}&theme=${theme}`;
          const page = await context.newPage();
          await page.goto(url, { waitUntil: 'networkidle' });
          await page.waitForSelector('#slide[data-ready="true"]', { timeout: 10_000 }).catch(() => problem(`${id} #${i + 1}: client render did not signal ready`));
          const overflow = await page.evaluate(() =>
            [...document.querySelectorAll('[data-col]')].filter((el) => el.scrollHeight > el.clientHeight + 1).map((el) => `${el.dataset.col} column overflows by ${el.scrollHeight - el.clientHeight}px`),
          );
          overflow.forEach((o) => problem(`${id} #${i + 1} (${layout.slides[i].title}): ${o}`));
          await page.locator('#slide').screenshot({ path: file, type: 'png', animations: 'disabled' });
          await page.close();
          report.files.push({ dossier: id, slide: i + 1, theme, title: layout.slides[i].title, file: path.relative(root, file), width: width * Number(opts.scale), height: height * Number(opts.scale), source: 'live' });
          console.log(`  ${path.relative(root, file)}${overflow.length ? '  (overflow)' : ''}`);
        }

        await context.close();
      }
    }
  }
} finally {
  await browser.close();
  server.stop();
}

fs.writeFileSync(path.join(out, 'manifest.json'), JSON.stringify(report, null, 2));
if (process.env.OUT_OWNER)
  spawnSync('chown', ['-R', process.env.OUT_OWNER, out]);
else
  spawnSync('chmod', ['-R', 'a+rwX', out]);

console.log(`\n${report.files.length} files -> ${path.relative(root, out) || '.'}${problems ? `, ${problems} problem(s)` : ''}`);
process.exit(opts.strict && problems ? 1 : 0);

function sizeOf(size = 'landscape') {
  const presets = { landscape: [1600, 1000], wide: [1600, 900], og: [1200, 630], square: [1080, 1080], portrait: [1080, 1350], story: [1080, 1920] };
  if (Array.isArray(size))
    return size.map(Number);

  const key = String(size).toLowerCase();
  if (presets[key])
    return presets[key];

  const m = key.match(/^(\d{3,5})\s*x\s*(\d{3,5})$/);
  return m ? [Number(m[1]), Number(m[2])] : presets.landscape;
}
