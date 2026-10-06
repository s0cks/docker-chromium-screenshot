// Entry point for src/pages/live.astro: fetch a compiled layout (served by
// the Go service from that card's layout.json/yaml/jsonnet, see
// docs/authoring.md#json-and-jsonnet-cards) and render one slide of it client-side.
//
//   /live/?dossier=<id>&slide=<n>&theme=light|dark
//
// Sets #slide[data-ready="true"] once rendering (including web fonts) is
// settled, which render.mjs's Playwright capture waits on, same as it waits
// for networkidle on the server-rendered pages.
import { el, pad } from './dom.js';
import { renderNode } from './blocks.js';

const SIZE_PRESETS = {
  landscape: [1600, 1000], wide: [1600, 900], og: [1200, 630],
  square: [1080, 1080], portrait: [1080, 1350], story: [1080, 1920],
};

function parseSize(size = 'landscape') {
  if (Array.isArray(size)) return size.map(Number);
  const key = String(size).toLowerCase();
  if (SIZE_PRESETS[key]) return SIZE_PRESETS[key];
  const m = key.match(/^(\d{3,5})\s*x\s*(\d{3,5})$/);
  return m ? [Number(m[1]), Number(m[2])] : SIZE_PRESETS.landscape;
}

async function fetchJSON(url) {
  const res = await fetch(url, { cache: 'no-store' });
  if (!res.ok) throw new Error(`${url}: ${res.status} ${res.statusText}`);
  return res.json();
}

function applyTheme(root, { theme, accent }) {
  root.dataset.theme = theme;
  // A real (if injected) stylesheet, not inline style properties: inline
  // styles would outrank the /cards/_shared/theme.css link regardless
  // of specificity, defeating theme mounts for live-rendered dossiers the
  // same way an inline style="" on <html> would (see Base.astro).
  let style = document.getElementById('live-vars');
  if (!style) {
    style = document.createElement('style');
    style.id = 'live-vars';
    const overrides = document.querySelector('link[href="/cards/_shared/theme.css"]');
    document.head.insertBefore(style, overrides ?? document.head.firstChild);
  }
  // No baked pixel size here either — same fluid vmin/cqi scale as the
  // server-rendered path (main.scss, chrome.scss); only theme vars travel.
  const vars = [
    `--accent: var(--${accent})`,
    `--accent-fill: var(--${accent}-fill)`,
    `--accent-2: var(--${accent}-2)`,
    `--accent-on: var(--${accent === 'yellow' ? 'yellow-on' : 'on-fill'})`,
  ];
  style.textContent = `:root {\n  ${vars.join(';\n  ')};\n}`;
}

function buildMasthead({ series, logo }, index, total) {
  const header = el('header', { class: 'masthead' });
  const brand = el('div', { class: 'brand' });
  if (logo) brand.append(el('img', { class: 'logo', src: logo, alt: '' }));
  brand.append(el('span', { class: 'series', text: series }));
  header.append(brand);
  header.append(el('span', { class: 'pager', text: `${pad(index + 1)} / ${pad(total)}` }));
  return header;
}

function buildKicker(parts) {
  const bar = el('div', { class: 'kicker' });
  for (const p of parts.filter(Boolean)) bar.append(el('span', { text: p }));
  return bar;
}

function buildColophon({ series, author, provenanceText }) {
  const footer = el('footer', { class: 'colophon' });
  footer.append(el('span', { text: series }));
  if (provenanceText) footer.append(el('span', { class: 'prov', text: provenanceText }));
  if (author) {
    const m = author.match(/^(.*?)(\d+)$/);
    const span = el('span', { class: 'author' });
    if (m) {
      span.append(document.createTextNode(m[1]));
      span.append(el('b', { text: m[2] }));
    } else span.textContent = author;
    footer.append(span);
  }
  return footer;
}

function buildColumn(nodes, { withTitle, title, grid }) {
  const inner = el('div', { class: grid ? 'col-inner grid-2' : 'col-inner' });
  if (withTitle && title) inner.append(el('h2', { class: 'title', text: title }));
  for (const node of nodes) renderNode(inner, node);
  return inner;
}

async function main() {
  const params = new URLSearchParams(location.search);
  const dossierId = params.get('dossier');
  const slideIndex = Math.max(1, Number(params.get('slide') ?? '1')) - 1;
  const themeOverride = params.get('theme');
  const mount = document.getElementById('slide-root');

  if (!dossierId) {
    mount.textContent = 'Usage: /live/?dossier=<id>&slide=<n>&theme=light|dark';
    return;
  }

  let dossier;
  try {
    dossier = await fetchJSON(`/layout/${encodeURIComponent(dossierId)}`);
  } catch (err) {
    mount.textContent = `Failed to load layout "${dossierId}": ${err.message}`;
    console.error(err);
    return;
  }

  const meta = {
    series: 'Dossier', accent: 'purple', theme: 'light', provenance: false,
    ...dossier.meta,
  };
  const slide = dossier.slides?.[slideIndex];
  if (!slide) {
    mount.textContent = `Layout "${dossierId}" has no slide ${slideIndex + 1} (it has ${dossier.slides?.length ?? 0}).`;
    return;
  }

  const [width, height] = parseSize(meta.size);
  applyTheme(document.documentElement, { theme: themeOverride ?? meta.theme, accent: meta.accent, width, height });
  if (!document.querySelector(`link[href="/cards/${dossierId}/theme.css"]`)) {
    const cardTheme = document.createElement('link');
    cardTheme.rel = 'stylesheet';
    cardTheme.href = `/cards/${encodeURIComponent(dossierId)}/theme.css`;
    document.head.appendChild(cardTheme);
  }
  document.title = `${meta.title ?? meta.series} ${pad(slideIndex + 1)}/${pad(dossier.slides.length)}`;

  const article = el('article', { class: 'slide', id: 'slide', 'data-dossier': dossierId, 'data-slide': slideIndex + 1 });
  article.append(buildMasthead(meta, slideIndex, dossier.slides.length));
  article.append(buildKicker([meta.subject, meta.part, slide.kicker ?? slide.title]));

  const layout = slide.layout ?? (slide.lead?.length ? 'split' : 'full');
  const body = el('div', {
    class: 'body',
    'data-layout': layout,
    'data-valign': slide.valign ?? 'center',
    style: slide.lead_width ? { '--lead': `${slide.lead_width}%` } : {},
  });
  if (layout === 'full') {
    body.append(el('div', { class: 'col', 'data-col': 'main' }, [buildColumn(slide.main ?? [], { withTitle: true, title: slide.title, grid: true })]));
  } else {
    body.append(el('div', { class: 'col lead', 'data-col': 'lead' }, [buildColumn(slide.lead ?? [], { withTitle: true, title: slide.title, grid: false })]));
    body.append(el('div', { class: 'col main', 'data-col': 'main' }, [buildColumn(slide.main ?? [], { withTitle: false, grid: true })]));
  }
  article.append(body);

  const provenanceText = meta.provenance
    ? [meta.ref, dossier.commit?.slice(0, 7), dossier.builtAt?.slice(0, 10)].filter(Boolean).join('  \u00b7  ')
    : '';
  article.append(buildColophon({ series: meta.series, handle: meta.handle, provenanceText }));

  mount.replaceChildren(article);

  await document.fonts.ready;
  article.dataset.ready = 'true';
}

main().catch((err) => {
  console.error(err);
  document.getElementById('slide-root').textContent = `Render error: ${err.message}`;
});
