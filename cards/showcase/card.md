---
series: Sample Dossier
subject: Halden Freight
part: Q3 2026 review
handle: "@dossier_demo_2026"
ref: DOS-2026-014
provenance: true
theme: dark
size: 1600x1000
snippets:
  invented: |
    ```callout tone=plain
    Invented data. Every name, figure and date in this sample is made up to show the components.
    ```
---

## Halden at a glance

Halden Freight moves one in nine containers through the northern corridor. It missed its on-time target for a third quarter running, and the board has asked for an independent review.

```callout tone=blue
**Bottom line.** Delays trace to two terminals, not the network. Fix Rotterdam-North and Gdansk and on-time delivery returns to target by Q1.
```

@include invented

+++

### Entity record

```profile
Legal name: Halden Freight Group plc
Founded: 1994, Hamburg
Fleet: 212 vessels, 38 owned
Terminals: 14, of which 2 flagged
Chair: I. Solheim
Status: "**Under board review**"
```

### Q3 vitals

```stats cols=3
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

## How it unfolded

Three shocks in ten weeks: a port strike removed capacity, a software cutover removed visibility, and a late customs rule change turned a backlog into a queue.

```callout tone=plain
Events above the line were outside Halden's control. Events below are its own decisions.
```

+++

### Timeline

```timeline orientation=horizontal scale=time h=19
events:
  - date: 2026-04
    title: Rotterdam strike
    text: 11 days, 3 berths lost
    tone: red
    side: up
  - date: 2026-05
    title: Yard system cutover
    text: Planned for 2 days, ran 9
    tone: orange
    side: down
  - date: 2026-06
    title: Customs rule change
    text: New pre-clearance filing
    tone: yellow
    side: up
  - date: 2026-07
    title: Gdansk backlog peaks
    text: 7,400 boxes waiting
    tone: red
    side: up
  - date: 2026-08
    title: Board review ordered
    text: Independent panel named
    tone: purple
    side: down
  - date: 2026-09
    title: Interim fix live
    text: Manual gate windows
    tone: green
    side: down
```

### What the timeline shows

```findings
- head: The cutover was the avoidable delay.
  text: The strike was external, but the yard system change arrived two weeks after it, with no rollback plan.
  source: Board paper 14 · Ops incident log
- head: Customs was the multiplier.
  text: Pre-clearance added about a day per box just as Gdansk hit its busiest month.
  source: Customs notice 2026/11 · Terminal KPIs
```

---

## The numbers

On-time delivery fell 11 points in five months. Two terminals account for nearly two thirds of the cost, and one cause, terminal congestion, drives more than half of delays.

```quote by="Board review panel" source="Interim note, 3 Sep"
The network is not slow. Two terminals are.
```

+++

```chart w=full h=275
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
format: { suffix: '%', compact: false }
min: 70
max: 100
annotations:
  - x: May
    text: Cutover
source: Terminal KPIs · Peer benchmark, 9 carriers
```

```chart w=half h=230
type: hbar
title: Delay cost by terminal, €M
items:
  - { label: Rotterdam-North, value: 14.2, tone: red }
  - { label: Gdansk, value: 11.3, tone: red }
  - { label: Antwerp, value: 5.1, tone: blue }
  - { label: Felixstowe, value: 4.4, tone: blue }
  - { label: Other 10, value: 6.0, tone: muted }
format: { prefix: '€', suffix: 'M', decimals: 1 }
```

```chart w=half h=230
type: donut
title: Delay cause, share of hours
center: { value: 54%, label: Congestion }
items:
  - { label: Congestion, value: 54, tone: red }
  - { label: Customs, value: 21, tone: yellow }
  - { label: Systems, value: 15, tone: orange }
  - { label: Weather, value: 10, tone: cyan }
format: { suffix: '%', compact: false }
```

---

## What we assess

The evidence supports a narrow fix. Address the two terminals first, and treat the systems programme as the biggest governance question.

```qr
link: https://github.com/s0cks/docker-chromium-screenshot
label: Source and build notes
size: 6
```

+++

### Open questions

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

### Recommended actions

| Action | Owner | By | Cost |
| --- | --- | --- | ---: |
| Add gate windows at Gdansk | Terminals | Oct | €0.8M |
| Rebuild Rotterdam-North yard plan | Ops | Nov | €2.4M |
| Independent audit of yard system | Board | Dec | €0.5M |

```sources
- Board paper 14, 22 Aug 2026
- Terminal KPI export, Apr to Sep 2026
- Customs notice 2026/11
```
