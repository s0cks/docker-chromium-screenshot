// Timeline geometry, built on d3-scale (d3-array for its date extent) —
// the reusable component behind both the server-rendered Timeline.astro and
// the client-side src/client/timeline-d3.js, so a horizontal or vertical
// timeline lays out identically whichever pipeline draws it.
import { extent as d3extent } from 'd3-array';
import { scaleTime } from 'd3-scale';

/** Parse "2019", "2019-04", "2019-04-12" to a Date, else null. */
export function parseWhen(s) {
  const m = String(s).trim().match(/^(\d{4})(?:-(\d{2}))?(?:-(\d{2}))?$/);
  if (!m) return null;
  return new Date(Date.UTC(Number(m[1]), Number(m[2] ?? 1) - 1, Number(m[3] ?? 1)));
}

/**
 * Horizontal positions (0-100) for timeline events. Even spacing by default;
 * `scale: time` uses a d3 time scale, then enforces a minimum gap so labels
 * never collide (d3 spaces by date, not by legibility).
 */
export function timelinePositions(events, { scale = 'even', lo = 7, hi = 93 } = {}) {
  const n = events.length;
  if (n === 1) return [50];
  const even = events.map((_, i) => lo + ((hi - lo) * i) / (n - 1));
  if (scale !== 'time') return even;
  const dates = events.map((e) => parseWhen(e.date));
  if (dates.some((d) => d === null)) return even;
  const time = scaleTime().domain(d3extent(dates)).range([lo, hi]);
  const pos = dates.map((d) => time(d));
  const gap = Math.min(9, (hi - lo) / (n - 1));
  for (let i = 1; i < n; i++) pos[i] = Math.max(pos[i], pos[i - 1] + gap);
  // a forward pass to enforce gaps can overshoot the axis: compress back into range
  if (pos[n - 1] > hi) {
    const k = (hi - lo) / (pos[n - 1] - lo);
    for (let i = 1; i < n; i++) pos[i] = lo + (pos[i] - lo) * k;
  }
  return pos;
}

/**
 * Full layout for a horizontal timeline: each event annotated with its `p`
 * (0-100 position from timelinePositions), `side` ('up'|'down' — alternating
 * unless the event sets its own), and a shared `width` (label width as a
 * percent, sized to the gap between same-side neighbours so labels never
 * overlap). Shared by Timeline.astro and src/client/timeline-d3.js so an
 * "orientation: horizontal" timeline is one component, not two
 * implementations that could drift.
 */
export function horizontalLayout(events, { scale = 'even', lo = 7, hi = 93 } = {}) {
  const pos = timelinePositions(events, { scale, lo, hi });
  const sides = events.map((e, i) => (e.side === 'up' || e.side === 'down' ? e.side : i % 2 === 0 ? 'up' : 'down'));
  let minGap = 100;
  for (const side of ['up', 'down']) {
    const p = pos.filter((_, i) => sides[i] === side);
    for (let i = 1; i < p.length; i++) minGap = Math.min(minGap, p[i] - p[i - 1]);
  }
  const width = Math.max(13, Math.min(28, minGap * 0.92));
  return events.map((e, i) => ({ ...e, p: pos[i], side: sides[i], width }));
}
