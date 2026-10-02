// Chart geometry, built on D3 (d3-scale, d3-shape, d3-array) rather than
// hand-rolled math, so both the server-rendered SVG (Chart.astro) and the
// client-side live renderer (src/client/chart-d3.js) compute axes, curves
// and arcs the same way.
import { scaleLinear } from 'd3-scale';
import { arc as d3arc, area as d3area, line as d3line, pie as d3pie } from 'd3-shape';

export const SERIES_TONES = ['blue', 'orange', 'green', 'purple', 'cyan', 'magenta', 'yellow', 'red'];

/** CSS colour for a tone name, falling back to the n-th series colour. */
export function toneColor(tone, i = 0) {
  if (!tone) return `var(--series-${(i % SERIES_TONES.length) + 1})`;
  if (tone === 'accent') return 'var(--accent)';
  if (tone === 'ink') return 'var(--ink)';
  if (tone === 'muted') return 'var(--ink-2)';
  return `var(--${tone})`;
}

/**
 * A d3 linear scale from [min, dataMax] (or an explicit `max`), "niced" to
 * round tick values. Returns { min, max, ticks, scale } — `scale` maps a
 * data value straight to a pixel position once you call `.range([...])` on
 * it (it's left unranged here; callers set the range once they know the
 * plot's pixel height).
 */
export function niceAxis(dataMax, count = 4, { min = 0, max } = {}) {
  const top = max ?? dataMax;
  const scale = scaleLinear()
    .domain([min, Number.isFinite(top) && top > min ? top : min + 1])
    .nice(count);
  const [lo, hi] = scale.domain();
  return { min: lo, max: hi, ticks: scale.ticks(count), scale };
}

/** 12400 -> "12.4k"; honours { prefix, suffix, decimals, compact }. */
export function fmt(v, { prefix = '', suffix = '', decimals, compact = true } = {}) {
  if (v === null || v === undefined || Number.isNaN(Number(v))) return '';
  const n = Number(v);
  const abs = Math.abs(n);
  let out;
  if (compact && abs >= 1e12) out = trim(n / 1e12, decimals) + 'T';
  else if (compact && abs >= 1e9) out = trim(n / 1e9, decimals) + 'B';
  else if (compact && abs >= 1e6) out = trim(n / 1e6, decimals) + 'M';
  else if (compact && abs >= 1e4) out = trim(n / 1e3, decimals) + 'k';
  else out = decimals === undefined ? String(Number(n.toFixed(2))) : n.toFixed(decimals);
  return `${prefix}${out}${suffix}`;
}
const trim = (n, d) => (d === undefined ? String(Number(n.toFixed(1))) : n.toFixed(d));

/** An SVG `d` path for a line series, given x(i) and y(v) pixel-mapping functions. */
export function linePath(values, x, y) {
  return d3line()
    .x((v, i) => x(i))
    .y((v) => y(v))(values);
}

/** An SVG `d` path for an area series filled down to y(baseline). */
export function areaPath(values, x, y, baseline = 0) {
  return d3area()
    .x((v, i) => x(i))
    .y0(y(baseline))
    .y1((v) => y(v))(values);
}

/** Pie/donut wedges: [{ path, midAngle }], one per value, via d3-shape. */
export function donutArcs(values, { innerRadius, outerRadius, padAngle = 0.012 }) {
  const gen = d3arc().innerRadius(innerRadius).outerRadius(outerRadius).cornerRadius(2);
  const wedges = d3pie()
    .value((v) => Math.max(0, v))
    .sort(null)
    .padAngle(padAngle)(values);
  return wedges.map((w) => ({ path: gen(w), midAngle: (w.startAngle + w.endAngle) / 2 }));
}
