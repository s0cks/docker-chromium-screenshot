// Client-side chart rendering with d3-selection, sharing scale/shape/format
// logic with the server-rendered version (src/lib/chart.js) so a live JSON
// dossier and a Markdown one draw identical charts.
import { select } from 'd3-selection';
import { areaPath, donutArcs, fmt, linePath, niceAxis, toneColor } from '../lib/chart.js';
import { el, svg } from './dom.js';

/** Render a `chart` block's data into `container` (a DOM element). */
export function renderChart(container, data, attrs = {}) {
  const type = attrs.type ?? data.type ?? 'bar';
  const half = attrs.w === 'half';
  const vw = half ? 430 : 880;
  const vh = Number(attrs.h ?? data.h ?? (half ? 250 : 300));
  const format = data.format ?? {};
  const showLabels = data.labels ?? true;
  const xs = data.x ?? [];
  const series = (data.series ?? []).map((s, i) => ({ ...s, color: toneColor(s.tone, i) }));
  const items = (data.items ?? []).map((s, i) => ({ ...s, color: toneColor(s.tone, i) }));

  const figure = select(container).append('figure').attr('class', 'chart').style('margin', '0').style('--kw', half ? '40%' : '28%');
  if (data.title) figure.append('div').attr('class', 'chart-title').text(data.title);

  const legend = data.legend ?? series.length > 1;
  if (legend && type !== 'donut' && type !== 'hbar') {
    const l = figure.append('div').attr('class', 'legend');
    for (const s of series) {
      const span = l.append('span');
      span.append('i').style('--c', s.color);
      span.append(() => document.createTextNode(s.name));
    }
  }

  if (type === 'bar' || type === 'line' || type === 'area') {
    const isLine = type !== 'bar';
    const m = { l: 46, r: 14, t: 26, b: 30 };
    const pw = vw - m.l - m.r;
    const ph = vh - m.t - m.b;
    const stacked = Boolean(data.stacked);
    const dataMax = stacked && series.length
      ? Math.max(...xs.map((_, i) => series.reduce((a, s) => a + (s.values[i] ?? 0), 0)))
      : Math.max(0, ...series.flatMap((s) => s.values));
    const axis = niceAxis(data.max ?? dataMax * (showLabels && !isLine ? 1.08 : 1), 4, { min: isLine ? (data.min ?? 0) : 0, max: data.max });
    axis.scale.range([m.t + ph, m.t]);
    const y = (v) => axis.scale(v);
    const band = xs.length ? pw / xs.length : pw;
    const cx = (i) => m.l + band * i + band / 2;
    const lx = (i) => (xs.length > 1 ? m.l + 14 + ((pw - 28) * i) / (xs.length - 1) : m.l + pw / 2);
    const annotations = (data.annotations ?? [])
      .map((a) => ({ ...a, i: typeof a.x === 'number' && !xs.includes(a.x) ? a.x : xs.findIndex((v) => String(v) === String(a.x)) }))
      .filter((a) => a.i >= 0);

    const root = svg('svg', { viewBox: `0 0 ${vw} ${vh}`, role: 'img', 'aria-label': data.title ?? `${type} chart` });
    for (const t of axis.ticks) {
      root.append(svg('line', { class: 'grid-line', x1: m.l, x2: vw - m.r, y1: y(t), y2: y(t) }));
      root.append(svg('text', { class: 'ax', x: m.l - 8, y: y(t) + 4, 'text-anchor': 'end' }, [document.createTextNode(fmt(t, format))]));
    }

    if (type === 'bar') {
      xs.forEach((label, i) => {
        const n = series.length;
        const groupW = Math.min(band * 0.74, 64 * n);
        const w = stacked ? groupW : groupW / n;
        let acc = 0;
        series.forEach((s, k) => {
          const v = s.values[i] ?? 0;
          const x = stacked ? cx(i) - groupW / 2 : cx(i) - groupW / 2 + k * w;
          const y0 = stacked ? acc : 0;
          if (stacked) acc += v;
          const bw = Math.max(2, w - (stacked ? 0 : 2));
          const top = y(y0 + v);
          const bottom = y(y0);
          root.append(svg('rect', { x, y: top, width: bw, height: Math.max(0, bottom - top), fill: s.color }));
          if (showLabels && !stacked) root.append(svg('text', { class: 'val', x: x + bw / 2, y: top - 6, 'text-anchor': 'middle' }, [document.createTextNode(fmt(v, format))]));
        });
        root.append(svg('text', { class: 'ax', x: cx(i), y: vh - 8, 'text-anchor': 'middle' }, [document.createTextNode(label)]));
      });
    } else {
      xs.forEach((label, i) => root.append(svg('text', { class: 'ax', x: lx(i), y: vh - 8, 'text-anchor': 'middle' }, [document.createTextNode(label)])));
      const lastVals = series.map((s) => s.values.at(-1));
      series.forEach((s, si) => {
        if (type === 'area') root.append(svg('path', { d: areaPath(s.values, lx, y, axis.min), fill: s.color, 'fill-opacity': 0.16 }));
        root.append(svg('path', { d: linePath(s.values, lx, y), fill: 'none', stroke: s.color, 'stroke-width': 3, 'stroke-linejoin': 'round', 'stroke-linecap': 'round' }));
        s.values.forEach((v, i) => root.append(svg('circle', { class: 'bg-dot', cx: lx(i), cy: y(v), r: 4.5, fill: s.color })));
        if (showLabels) {
          const below = lastVals[si] < Math.max(...lastVals);
          root.append(svg('text', { class: 'val', x: lx(s.values.length - 1), y: y(s.values.at(-1)) + (below ? 22 : -11), 'text-anchor': 'end', style: `fill:${s.color}` }, [document.createTextNode(fmt(s.values.at(-1), format))]));
        }
      });
    }
    root.append(svg('line', { class: 'base-line', x1: m.l, x2: vw - m.r, y1: y(axis.min), y2: y(axis.min) }));
    for (const a of annotations) {
      const anchor = a.i > xs.length / 2 ? 'end' : 'start';
      root.append(svg('line', { class: 'ann-line', x1: cx ? cx(a.i) : lx(a.i), x2: cx ? cx(a.i) : lx(a.i), y1: m.t - 8, y2: y(axis.min) }));
      root.append(svg('text', { class: 'ann', x: (cx ? cx(a.i) : lx(a.i)) + (anchor === 'end' ? -6 : 6), y: m.t - 2, 'text-anchor': anchor }, [document.createTextNode(a.text)]));
    }
    container.append(root);
  } else if (type === 'hbar') {
    const hmax = Math.max(0, ...items.map((s) => s.value));
    const wrap = el('div', { class: 'hbars' });
    for (const s of items) {
      const row = el('div', { class: 'hbar', 'data-tone': s.tone });
      row.append(el('span', { class: 'k', text: s.label }));
      const track = el('div', { class: 'track' });
      track.append(el('span', { class: 'fill', style: { display: 'block', '--pct': `${(s.value / (hmax || 1)) * 100}%`, background: s.color } }));
      row.append(track);
      row.append(el('span', { class: 'num', text: fmt(s.value, format) }));
      wrap.append(row);
    }
    figure.node().append(wrap);
  } else if (type === 'donut') {
    const R = 78;
    const total = items.reduce((a, s) => a + s.value, 0) || 1;
    const arcs = items.length ? donutArcs(items.map((s) => s.value), { innerRadius: R - 30, outerRadius: R }) : [];
    const wrap = el('div', { class: 'donut', 'data-size': half ? 'sm' : 'md' });
    const root = svg('svg', { viewBox: `${-R} ${-R} ${R * 2} ${R * 2}`, role: 'img', 'aria-label': data.title ?? 'donut chart' });
    arcs.forEach((a, i) => root.append(svg('path', { d: a.path, fill: items[i].color })));
    if (data.center) root.append(svg('text', { class: 'donut-c', x: 0, y: 4, 'text-anchor': 'middle' }, [document.createTextNode(data.center.value)]));
    if (data.center?.label) root.append(svg('text', { class: 'donut-l', x: 0, y: 24, 'text-anchor': 'middle' }, [document.createTextNode(String(data.center.label).toUpperCase())]));
    wrap.append(root);
    const ul = el('ul');
    for (const s of items) {
      const li = el('li');
      li.append(el('i', { style: { '--c': s.color } }));
      li.append(el('span', { text: s.label }));
      li.append(el('span', { class: 'pct', text: data.show === 'value' ? fmt(s.value, format) : `${Math.round((s.value / total) * 100)}%` }));
      ul.append(li);
    }
    wrap.append(ul);
    figure.node().append(wrap);
  }

  if (data.source) {
    figure.append('div').attr('class', 'chart-note').text(Array.isArray(data.source) ? data.source.join(' \u00b7 ') : data.source);
  }
}
