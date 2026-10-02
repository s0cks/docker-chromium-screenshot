// Client-side block renderers. Each mirrors the markup its .astro twin in
// src/components/blocks/ produces, so blocks.css applies unchanged whether a
// slide was server-rendered from Markdown or fetched as live layout JSON.
import { mdi, md } from '../lib/markdown.js';
import { el } from './dom.js';
import { renderChart } from './chart-d3.js';
import { renderTimeline as renderTimelineD3 } from './timeline-d3.js';

const STATUS_TONE = { open: 'accent', confirmed: 'green', alleged: 'yellow', disputed: 'red', closed: 'muted', watch: 'purple' };
const joinSource = (s) => (Array.isArray(s) ? s.join(' \u00b7 ') : s);
const arrow = (d) => (String(d).trim().startsWith('-') || String(d).trim().startsWith('\u2212') ? '\u2193' : '\u2191');
const items = (data) => (Array.isArray(data) ? data : (data.items ?? []));

export function renderCallout(container, node) {
  container.append(el('aside', { class: 'callout', 'data-tone': node.attrs?.tone ?? 'ink', 'data-mono': node.attrs?.mono || undefined, html: md(node.md ?? '') }));
}

export function renderQuote(container, node) {
  const cite = [node.attrs?.by, node.attrs?.source].filter(Boolean).join(' \u00b7 ');
  const figure = el('figure', { class: 'quote', 'data-tone': node.attrs?.tone ?? 'accent' });
  figure.append(el('blockquote', { html: md(node.md ?? '') }));
  if (cite) figure.append(el('figcaption', { text: cite }));
  container.append(figure);
}

export function renderStats(container, node) {
  const { data, attrs = {} } = node;
  const list = items(data);
  const cols = attrs.cols ?? data.cols ?? Math.min(list.length, 2) ?? 2;
  const wrap = el('div', { class: 'stats', style: { '--cols': cols } });
  for (const s of list) {
    const filled = s.tone && s.tone !== 'outline';
    const stat = el('div', { class: 'stat', 'data-tone': filled ? s.tone : 'ink', 'data-filled': filled || undefined });
    stat.append(el('div', { class: 'v', text: s.value }));
    if (s.label) stat.append(el('div', { class: 'l', html: mdi(s.label) }));
    if (s.note) stat.append(el('div', { class: 'n', html: mdi(s.note) }));
    if (s.delta) stat.append(el('div', { class: 'd', 'data-good': s.trend, text: `${arrow(s.delta)} ${String(s.delta).replace(/^[-\u2212+]/, '')}${s.vs ? ` ${s.vs}` : ''}` }));
    wrap.append(stat);
  }
  container.append(wrap);
}

export function renderFindings(container, node) {
  const { data, attrs = {} } = node;
  const ul = el('ul', { class: 'findings' });
  for (const f of items(data)) {
    const li = el('li', { 'data-tone': f.tone ?? attrs.tone ?? 'accent' });
    if (f.head) li.append(el('strong', { html: mdi(f.head) + ' ' }));
    li.append(el('span', { html: mdi(f.text ?? '') }));
    if (f.source) li.append(el('div', { class: 'src', text: joinSource(f.source) }));
    ul.append(li);
  }
  container.append(ul);
}

export function renderThreads(container, node) {
  const { data } = node;
  const showLabels = data.labels ?? (Array.isArray(data) ? false : data.labels) ?? false;
  const wrap = el('div', { class: 'threads' });
  for (const t of items(data)) {
    const row = el('div', { class: 'thread', 'data-tone': STATUS_TONE[t.status] ?? t.tone ?? 'blue' });
    const line = el('div');
    if (showLabels && t.status) line.append(el('span', { class: 'chip', text: t.status }));
    line.append(el('span', { class: 't', html: mdi(t.title ?? '') }));
    if (t.text) line.append(el('span', { html: ' ' + mdi(t.text) }));
    row.append(line);
    if (t.source) row.append(el('div', { class: 'src', text: joinSource(t.source) }));
    wrap.append(row);
  }
  container.append(wrap);
}

export function renderProfile(container, node) {
  const { data } = node;
  const { title, rows } = data.rows ? data : { title: data.title, rows: Object.fromEntries(Object.entries(data).filter(([k]) => k !== 'title')) };
  const wrap = el('div', { class: 'profile' });
  if (title) wrap.append(el('div', { class: 'profile-title', text: title }));
  const dl = el('dl');
  for (const [k, v] of Object.entries(rows)) {
    dl.append(el('dt', { text: k }));
    dl.append(el('dd', { html: mdi(v) }));
  }
  wrap.append(dl);
  container.append(wrap);
}

export function renderSources(container, node) {
  const { data } = node;
  const list = items(data);
  const title = Array.isArray(data) ? 'Sources' : (data.title ?? 'Sources');
  const wrap = el('div', { class: 'sources' });
  wrap.append(el('h6', { text: title }));
  const ol = el('ol');
  for (const s of list) ol.append(el('li', { html: mdi(typeof s === 'string' ? s : [s.text, s.url].filter(Boolean).join(' \u00b7 ')) }));
  wrap.append(ol);
  container.append(wrap);
}


const RENDERERS = {
  callout: renderCallout,
  quote: renderQuote,
  stats: renderStats,
  findings: renderFindings,
  threads: renderThreads,
  timeline: (container, node) => renderTimelineD3(container, node.data, node.attrs ?? {}),
  profile: renderProfile,
  sources: renderSources,
  chart: (container, node) => renderChart(container, node.data, node.attrs ?? {}),
};

/** Render one parsed node ({ kind: 'md', md } or { kind: 'block', name, ... }) into container. */
export function renderNode(container, node) {
  if (node.kind === 'md') {
    container.append(el('div', { class: 'prose', html: md(node.md ?? '') }));
    return;
  }
  const render = RENDERERS[node.name];
  if (!render) {
    console.warn(`live renderer: unknown block "${node.name}"`);
    return;
  }
  const holder = node.attrs?.w === 'half' ? el('div', { 'data-w': 'half' }) : container;
  render(holder, node);
  if (holder !== container) container.append(holder);
}
