// Client-side timeline rendering with d3-selection, sharing layout geometry
// with the server-rendered version (src/lib/timeline.js + Timeline.astro) so
// a live JSON dossier's timeline — vertical or horizontal — matches a
// Markdown one exactly. The reusable component is the shared geometry in
// src/lib/timeline.js; this and Timeline.astro are its two renderers.
import { select } from 'd3-selection';
import { horizontalLayout } from '../lib/timeline.js';
import { mdi } from '../lib/markdown.js';

/** Render a `timeline` block's data into `container` (a DOM element). */
export function renderTimeline(container, data, attrs = {}) {
  const events = Array.isArray(data) ? data : (data.events ?? data.items ?? []);
  const orientation = attrs.orientation ?? data.orientation ?? 'vertical';
  const scale = attrs.scale ?? data.scale ?? 'even';
  const height = attrs.h ?? data.h ?? 17;

  const root = select(container);
  if (data.title) root.append('div').attr('class', 'tl-title').style('margin-bottom', '0.9rem').text(data.title);

  if (orientation === 'horizontal') {
    const laid_out = horizontalLayout(events, { scale });
    const tl = root.append('div').attr('class', 'tl-h').style('--th', `${height}rem`);
    tl.append('div').attr('class', 'axis');
    for (const e of laid_out) {
      tl.append('i').attr('class', 'dot').attr('data-tone', e.tone ?? 'accent').style('--p', `${e.p}%`);
      const ev = tl.append('div')
        .attr('class', 'ev')
        .attr('data-tone', e.tone ?? 'accent')
        .attr('data-side', e.side)
        .style('--p', `${e.p}%`)
        .style('--ew', `${e.width}%`);
      ev.append('time').text(e.date);
      ev.append('b').html(mdi(e.title ?? ''));
      if (e.text) ev.append('span').html(mdi(e.text));
    }
  } else {
    const ol = root.append('ol').attr('class', 'tl-v');
    for (const e of events) {
      const li = ol.append('li').attr('data-tone', e.tone ?? 'accent');
      li.append('time').text(e.date);
      li.append('span').attr('class', 'rail');
      const div = li.append('div').attr('class', 'e');
      div.append('b').html(mdi(e.title ?? ''));
      if (e.text) div.append('span').html(mdi(e.text));
    }
  }
}
