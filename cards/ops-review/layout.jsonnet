// Sample JSON-layout dossier: the same "Halden Freight" scenario as
// dossiers/showcase.md, built with jsonnet instead of Markdown, and served
// by the Go service at GET /layout/ops-review for the live SPA
// (/live/?dossier=ops-review&slide=1). See docs/authoring.md#json-layouts.
local d = import 'dossier.libsonnet';

{
  meta: {
    series: 'Ops Review',
    subject: 'Halden Freight',
    part: 'Q3 2026, jsonnet edition',
    author: '@dossier_demo_2026',
    theme: 'dark',
    accent: 'purple',
    size: 'landscape',
  },
  slides: [
    d.slide('Halden at a glance') {
      lead: [
        d.md(|||
          Halden Freight moves one in nine containers through the northern
          corridor. This slide is generated straight from jsonnet — no
          Markdown file involved — and served by the Go layout endpoint.
        |||),
        d.callout('**Bottom line.** Same content model as Markdown dossiers: this jsonnet file builds the same block JSON the parser would.', tone='accent'),
      ],
      main: [
        d.md('### Q3 vitals'),
        d.stats([
          d.stat('81.4%', 'On-time delivery', tone='red', note='Target 92%', delta='-4.1 pts', vs='vs Q2', trend='bad'),
          d.stat('$41M', 'Delay-related cost', tone='ink', delta='+$9M', vs='vs Q2', trend='bad'),
        ], cols=2),
        d.md('### Delay cause'),
        d.chart('donut', {
          center: { value: '54%', label: 'Congestion' },
          items: [
            { label: 'Congestion', value: 54, tone: 'red' },
            { label: 'Customs', value: 21, tone: 'yellow' },
            { label: 'Systems', value: 15, tone: 'orange' },
            { label: 'Weather', value: 10, tone: 'cyan' },
          ],
        }),
      ],
    },
    d.slide('How it unfolded') {
      lead: [
        d.md('Three shocks in ten weeks — the same horizontal timeline component as the Markdown pipeline, laid out here from jsonnet and rendered client-side with D3.'),
      ],
      main: [
        d.md('### Timeline'),
        d.block('timeline', {
          orientation: 'horizontal',
          scale: 'time',
          h: 19,
          events: [
            d.event('2026-04', 'Rotterdam strike', text='11 days, 3 berths lost', tone='red') { side: 'up' },
            d.event('2026-05', 'Yard system cutover', text='Planned for 2 days, ran 9', tone='orange') { side: 'down' },
            d.event('2026-07', 'Gdansk backlog peaks', text='7,400 boxes waiting', tone='red') { side: 'up' },
            d.event('2026-09', 'Interim fix live', text='Manual gate windows', tone='green') { side: 'down' },
          ],
        }),
      ],
    },
  ],
}
