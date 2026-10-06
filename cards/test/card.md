---
series: Shadow Sovereignty
author: "@s0cks"
link: https://google.com
license: MIT

provenance: true

theme: dark
size: 1600x1000
---

## Quotes

```quote by="Board review panel" source="Interim note, 3 Sep"
The network is not slow. Two terminals are.
```

```quote by="Test" source="Test"
Test
```

---

## Callouts

```callout
This is a normal callout
```

```callout tone="accent"
This is a callout with a tone of 'accent'
```

```callout tone="red"
This is a callout with a 'red' tone
```

---

## Stats

```stats cols=2
- value: 81.4%
  label: On-time delivery
  note: Target 92%
  delta: -4.1 pts
  vs: vs Q2
  trend: bad
  tone: red
- value: 6.2 days
  label: Dwell time, flagged terminals
  note: Network median 2.1
  delta: +1.8 days
  vs: vs Q2
  trend: bad
- value: €41M
  label: Delay-related cost
  note: 62% at two terminals
  delta: +€9M
  vs: vs Q2
  trend: bad
```

---

## Findings

```findings
- head: The cutover was the avoidable delay.
  text: The strike was external, but the yard system change arrived two weeks after it, with no rollback plan.
  source: Board paper 14 · Ops incident log
- head: Customs was the multiplier.
  text: Pre-clearance added about a day per box just as Gdansk hit its busiest month.
  source:
  - Customs notice 2026/11 · Terminal KPIs
  - Test
```

---

## Profiles

```profile
title: Hello World
Legal name: Halden Freight Group plc
Founded: 1994, Hamburg
Fleet: 212 vessels, 38 owned
Terminals: 14, of which 2 flagged
Chair: I. Solheim
Status: "**Under board review**"
```

---

## Threads

Threads with no labels:

```threads
- status: confirmed
  title: Two terminals drive 62% of delay cost.
  text: Cross-checked against finance and terminal logs.
  source: Terminal KPIs · Finance ledger
- status: alleged
  title: Cutover was rushed to meet a bonus date.
  text: Two interviewees say so; no document yet.
- status: open
  title: Was a rollback plan ever written?
  text: Requested from the CIO on 9 Sep.
- status: watch
  title: Customs filing rule review
  text: Regulator decision due Q4.
```

Threads with labels:

```threads labels=true
- status: confirmed
  title: Two terminals drive 62% of delay cost.
  text: Cross-checked against finance and terminal logs.
  source: Terminal KPIs · Finance ledger
- status: alleged
  title: Cutover was rushed to meet a bonus date.
  text: Two interviewees say so; no document yet.
- status: open
  title: Was a rollback plan ever written?
  text: Requested from the CIO on 9 Sep.
- status: watch
  title: Customs filing rule review
  text: Regulator decision due Q4.
```

---

## Timeline (vertical)

```timeline
events:
  - date: 2026-04
    title: Rotterdam strike
    text: 11 days, 3 berths lost
    tone: red
  - date: 2026-05
    title: Yard system cutover
    text: Planned for 2 days, ran 9
    tone: orange
  - date: 2026-09
    title: Interim fix live
    text: Manual gate windows
    tone: green
```

---

## Timeline (horizontal)

```timeline orientation=horizontal scale=time
events:
  - date: 2026-04
    title: Rotterdam strike
    text: 11 days, 3 berths lost
    tone: red
  - date: 2026-05
    title: Yard system cutover
    text: Planned for 2 days, ran 9
    tone: orange
  - date: 2026-07
    title: Gdansk backlog peaks
    text: 7,400 boxes waiting
    tone: red
  - date: 2026-09
    title: Interim fix live
    text: Manual gate windows
    tone: green
```

---

## Charts: bar and line

```chart w=half
type: bar
title: Delay cost by terminal, €M
x: [Rotterdam, Gdansk, Antwerp, Felixstowe]
series:
  - name: Q3
    values: [14.2, 11.3, 5.1, 4.4]
    tone: red
```

```chart w=half
type: line
title: On-time delivery, % of boxes
x: [Apr, May, Jun, Jul, Aug, Sep]
series:
  - name: Halden
    values: [92, 88, 84, 78, 80, 81.4]
    tone: blue
  - name: Peer median
    values: [91, 91, 90, 91, 90, 91]
    tone: muted
```

---

## Charts: area and hbar

```chart w=half
type: area
title: Backlog, boxes waiting
x: [Jun, Jul, Aug, Sep]
series:
  - name: Gdansk
    values: [2100, 7400, 5200, 1800]
    tone: orange
```

```chart w=half
type: hbar
title: Delay cost by terminal, €M
items:
  - label: Rotterdam
    value: 14.2
    tone: red
  - label: Gdansk
    value: 11.3
    tone: orange
  - label: Antwerp
    value: 5.1
    tone: blue
```

---

## Charts: donut

```chart
type: donut
title: Delay cause, share of hours
center:
  value: 54%
  label: Congestion
items:
  - label: Congestion
    value: 54
    tone: red
  - label: Customs
    value: 21
    tone: yellow
  - label: Systems
    value: 15
    tone: orange
  - label: Weather
    value: 10
    tone: cyan
```

---

## Sources and QR

```sources
title: Sources
items:
  - text: Board paper 14, 22 Aug 2026
  - text: Terminal KPI export, Apr to Sep 2026
  - text: Customs notice 2026/11
```

```qr
link: https://github.com/s0cks/docker-chromium-screenshot
label: Source and build notes
```
