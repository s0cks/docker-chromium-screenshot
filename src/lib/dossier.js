// Dossier parser: Markdown + fenced component blocks -> slides.
//
//   ---                     frontmatter (YAML): series, subject, theme, size ...
//   ---
//   ## Slide title          first h1/h2 becomes the slide title
//   Lead paragraph...       everything before "+++" is the lead column
//   ```callout tone=blue    fenced blocks are components (see BLOCKS)
//   ...
//   ```
//   +++                     column break: what follows goes in the main column
//   ### Section heading
//   ```stats                YAML-bodied component
//   ...
//   ```
//   ---                     slide break
//
// Anything that is not a registered block is plain Markdown (GFM).

import fs from 'node:fs';
import path from 'node:path';
import YAML from 'yaml';

import { md, mdi } from './markdown.js';
export { md, mdi };

/** Registered blocks and how their fenced body is parsed. */
export const BLOCKS = {
  callout: 'md',
  quote: 'md',
  stats: 'yaml',
  findings: 'yaml',
  threads: 'yaml',
  timeline: 'yaml',
  chart: 'yaml',
  profile: 'yaml',
  sources: 'yaml',
  qr: 'yaml',
};

export const SIZE_PRESETS = {
  landscape: [1600, 1000],
  wide: [1600, 900],
  og: [1200, 630],
  square: [1080, 1080],
  portrait: [1080, 1350],
  story: [1080, 1920],
};

export const DEFAULT_WORD_BUDGET = 200;

export function parseSize(size = 'landscape') {
  if (Array.isArray(size)) return size.map(Number);
  const key = String(size).toLowerCase();
  if (SIZE_PRESETS[key]) return SIZE_PRESETS[key];
  const m = key.match(/^(\d{3,5})\s*x\s*(\d{3,5})$/);
  if (!m) throw new Error(`invalid size "${size}" (use WxH or one of: ${Object.keys(SIZE_PRESETS).join(', ')})`);
  return [Number(m[1]), Number(m[2])];
}

/** `tone=red mono w=half title="Two words"` -> { tone: 'red', mono: true, ... } */
export function parseAttrs(str = '') {
  const attrs = {};
  const re = /([\w-]+)(?:=("[^"]*"|'[^']*'|\S+))?/g;
  let m;
  while ((m = re.exec(str))) {
    let v = m[2];
    if (v === undefined) v = true;
    else if (/^["']/.test(v)) v = v.slice(1, -1);
    else if (v === 'true') v = true;
    else if (v === 'false') v = false;
    attrs[m[1]] = v;
  }
  return attrs;
}

function splitFrontmatter(src) {
  const m = src.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?/);
  if (!m) return { meta: {}, body: src };
  return { meta: YAML.parse(m[1]) ?? {}, body: src.slice(m[0].length) };
}

/** Replace `@include name` lines with frontmatter `snippets`. */
function expandSnippets(body, snippets = {}, where) {
  let out = body;
  for (let depth = 0; depth < 5; depth++) {
    let hit = false;
    out = out.replace(/^@include\s+([\w-]+)\s*$/gm, (_, name) => {
      if (!(name in snippets)) throw new Error(`${where}: unknown snippet "${name}"`);
      hit = true;
      return String(snippets[name]).trim();
    });
    if (!hit) break;
  }
  return out;
}

const FENCE = /^(`{3,})\s*([\w-]*)\s*(.*)$/;

/** Split on `---` lines that sit outside code fences. */
function splitSlides(body) {
  const slides = [[]];
  let fence = null;
  for (const line of body.split(/\r?\n/)) {
    const f = line.match(FENCE);
    if (f) {
      if (!fence) fence = f[1];
      else if (f[1].length >= fence.length && !f[2] && !f[3]) fence = null;
    }
    if (!fence && /^---\s*$/.test(line)) slides.push([]);
    else slides.at(-1).push(line);
  }
  return slides.map((l) => l.join('\n')).filter((s) => s.trim());
}

function collectText(value, out = []) {
  if (typeof value === 'string') out.push(value);
  else if (Array.isArray(value)) value.forEach((v) => collectText(v, out));
  else if (value && typeof value === 'object') Object.values(value).forEach((v) => collectText(v, out));
  return out;
}

const countWords = (text) => (text.replace(/<[^>]+>/g, ' ').match(/[\p{L}\p{N}][\p{L}\p{N}'’.,%$-]*/gu) ?? []).length;

export function parseSlide(source, { where = 'slide', index = 0 } = {}) {
  const slide = {
    index,
    title: null,
    config: {},
    lead: [],
    main: [],
    hasBreak: false,
    words: 0,
  };
  let region = 'lead';
  let mdLines = [];
  let words = 0;

  const flush = () => {
    const text = mdLines.join('\n').trim();
    mdLines = [];
    if (!text) return;
    words += countWords(text);
    slide[region].push({ kind: 'md', html: md(text) });
  };

  const lines = source.split(/\r?\n/);
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const f = line.match(FENCE);

    if (f) {
      const [, ticks, name, rest] = f;
      const kind = name === 'slide' ? 'slide' : name in BLOCKS ? BLOCKS[name] : null;
      // collect the fenced body (until a closing fence at least as long)
      let j = i + 1;
      const body = [];
      while (j < lines.length) {
        const c = lines[j].match(FENCE);
        if (c && c[1].length >= ticks.length && !c[2] && !c[3]) break;
        body.push(lines[j]);
        j++;
      }
      if (!kind) {
        // ordinary code fence: leave it to Markdown untouched
        mdLines.push(...lines.slice(i, j + 1));
        i = j;
        continue;
      }
      flush();
      const raw = body.join('\n');
      const ctx = `${where}, \`${name}\` block at slide line ${i + 1}`;
      if (kind === 'slide') {
        slide.config = { ...slide.config, ...parseYaml(raw, ctx) };
      } else {
        const node = { kind: 'block', name, attrs: parseAttrs(rest) };
        if (kind === 'yaml') {
          node.data = parseYaml(raw, ctx);
          words += countWords(collectText(node.data).join(' '));
        } else {
          node.md = raw.trim();
          words += countWords(node.md);
        }
        slide[region].push(node);
      }
      i = j;
      continue;
    }

    if (/^\+\+\+\s*$/.test(line)) {
      flush();
      region = 'main';
      slide.hasBreak = true;
      continue;
    }

    const h = line.match(/^#{1,2}\s+(.+?)\s*#*\s*$/);
    if (h && !slide.title) {
      slide.title = h[1];
      continue;
    }
    mdLines.push(line);
  }
  flush();

  slide.words = words;
  slide.layout = slide.config.layout ?? (slide.hasBreak ? 'split' : 'full');
  slide.kicker = slide.config.kicker ?? slide.title ?? '';
  return slide;
}

function parseYaml(raw, ctx) {
  try {
    return YAML.parse(raw) ?? {};
  } catch (err) {
    throw new Error(`${ctx}: invalid YAML: ${err.message}`);
  }
}

export function parseDossier(src, { id = 'dossier', file = id } = {}) {
  const { meta, body } = splitFrontmatter(src);
  const [width, height] = parseSize(meta.size);
  const budget = meta.budget ?? DEFAULT_WORD_BUDGET;
  const expanded = expandSnippets(body, meta.snippets, file);
  const chunks = splitSlides(expanded);
  const warnings = [];

  const slides = chunks.map((chunk, index) => {
    const slide = parseSlide(chunk, { where: `${file}, slide ${index + 1}`, index });
    if (slide.words > budget) {
      warnings.push(`slide ${index + 1}: ${slide.words} words exceeds the budget of ${budget}; split it or cut copy`);
    }
    if (!slide.title) warnings.push(`slide ${index + 1}: no title (add a "## Title" line)`);
    return slide;
  });
  if (!slides.length) throw new Error(`${file}: no slides found`);

  return {
    id,
    file,
    meta: {
      series: meta.series ?? 'Dossier',
      subject: meta.subject ?? '',
      part: meta.part ?? '',
      author: meta.author ?? '',
      ref: meta.ref ?? '',
      license: meta.license ?? '',
      link: meta.link ?? '',
      logo: meta.logo ?? null,
      theme: meta.theme === 'dark' ? 'dark' : 'light',
      accent: meta.accent ?? 'purple',
      provenance: Boolean(meta.provenance),
      title: meta.title ?? ([meta.subject, meta.part].filter(Boolean).join(' - ') || id),
    },
    size: { width, height },
    slides,
    warnings,
  };
}

export function cardsDir() {
  return path.resolve(process.env.CARDS_DIR ?? 'cards');
}

const LOGO_RE = /^logo\.(png|jpe?g|svg|webp)$/i;

/**
 * List card directories under `dir`: every immediate subdirectory whose name
 * doesn't start with `_` (that prefix is reserved for `_shared/`, holding a
 * jsonnet library, a repo-wide theme.css, etc. — see docs/build-a-card.md).
 * Does not check what kind of card each one is; loadDossiers() does that by
 * looking for card.md.
 */
export function listCardDirs(dir = cardsDir()) {
  if (!fs.existsSync(dir)) return [];
  return fs
    .readdirSync(dir, { withFileTypes: true })
    .filter((e) => e.isDirectory() && !e.name.startsWith('_'))
    .map((e) => e.name)
    .sort();
}

/** The card's own logo.{png,jpg,svg,webp}, as a /cards/<id>/... URL, or null. */
function detectLogo(id, cardDir) {
  const file = fs.readdirSync(cardDir).find((f) => LOGO_RE.test(f));
  return file ? `/cards/${id}/${file}` : null;
}

/**
 * Load every card that is a Markdown dossier: a cards/<id>/card.md file.
 * Each card is one composable directory — its own logo.*, theme.css and
 * meta.json/yaml/jsonnet live alongside it and need no separate mount. A
 * card without card.md is assumed to be a JSON/jsonnet card (see
 * service/internal/cards) and is skipped here, not an error.
 */
export function loadDossiers(dir = cardsDir()) {
  return listCardDirs(dir)
    .filter((id) => fs.existsSync(path.join(dir, id, 'card.md')))
    .map((id) => {
      const cardDir = path.join(dir, id);
      const dossier = parseDossier(fs.readFileSync(path.join(cardDir, 'card.md'), 'utf8'), { id, file: `${id}/card.md` });
      if (!dossier.meta.logo) dossier.meta.logo = detectLogo(id, cardDir);
      return dossier;
    });
}

export const pad = (n) => String(n).padStart(2, '0');
