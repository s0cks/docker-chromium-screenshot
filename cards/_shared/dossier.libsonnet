// Shared helpers for hand-built (JSON/jsonnet) layouts, mirroring the block
// shapes the Markdown pipeline produces (src/lib/dossier.js's BLOCKS) so the
// live SPA renders either source identically. See docs/authoring.md.
{
  // -- slide-level --
  slide(title, kicker=null)::
    { title: title, kicker: if kicker != null then kicker else title },

  // -- blocks: each returns a { kind: 'block', name, attrs, data|md } node --
  block(name, data, attrs={}):: { kind: 'block', name: name, attrs: attrs, data: data },
  md(text):: { kind: 'md', md: text },

  callout(text, tone='ink', attrs={})::
    { kind: 'block', name: 'callout', attrs: attrs { tone: tone }, md: text },

  quote(text, by=null, source=null, tone='accent')::
    { kind: 'block', name: 'quote', attrs: { tone: tone, by: by, source: source }, md: text },

  stat(value, label, tone='ink', note=null, delta=null, vs=null, trend=null)::
    { value: value, label: label, tone: tone }
    + (if note != null then { note: note } else {})
    + (if delta != null then { delta: delta, vs: vs, trend: trend } else {}),
  stats(items, cols=2):: self.block('stats', items, { cols: cols }),

  finding(text, head=null, source=null, tone=null)::
    { text: text }
    + (if head != null then { head: head } else {})
    + (if source != null then { source: source } else {})
    + (if tone != null then { tone: tone } else {}),
  findings(items):: self.block('findings', items),

  thread(title, status=null, text=null, source=null)::
    { title: title }
    + (if status != null then { status: status } else {})
    + (if text != null then { text: text } else {})
    + (if source != null then { source: source } else {}),
  threads(items, labels=false):: self.block('threads', items, { labels: labels }),

  event(date, title, text=null, tone='accent')::
    { date: date, title: title, tone: tone } + (if text != null then { text: text } else {}),
  timeline(events):: self.block('timeline', { events: events }),

  profile(rows, title=null)::
    self.block('profile', if title != null then { title: title, rows: rows } else rows),

  sources(items, title='Sources'):: self.block('sources', { title: title, items: items }),

  chart(type, opts={}, attrs={}):: self.block('chart', { type: type } + opts, attrs),
}
