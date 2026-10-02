import assert from 'node:assert/strict';
import { test } from 'node:test';
import { horizontalLayout, parseWhen, timelinePositions } from '../src/lib/timeline.js';

test('parseWhen understands year, month and day precision', () => {
  assert.equal(parseWhen('2026').getTime(), Date.UTC(2026, 0, 1));
  assert.equal(parseWhen('2026-04').getTime(), Date.UTC(2026, 3, 1));
  assert.equal(parseWhen('Q3 2026'), null);
});

test('timelinePositions stay in range, ordered, and separated', () => {
  const ev = ['2020', '2020-02', '2020-03', '2026'].map((date) => ({ date }));
  const pos = timelinePositions(ev, { scale: 'time' });
  assert.ok(pos.every((p, i) => p >= 7 - 1e-9 && p <= 93 + 1e-9 && (i === 0 || p > pos[i - 1])));
  assert.deepEqual(timelinePositions([{ date: 'a' }, { date: 'b' }], { scale: 'time' }), [7, 93]);
  assert.deepEqual(timelinePositions([{ date: 'x' }]), [50]);
});

test('horizontalLayout alternates sides and sizes labels to the tightest gap', () => {
  const events = [{ date: '2020' }, { date: '2021' }, { date: '2022' }, { date: '2023' }];
  const laid = horizontalLayout(events, { scale: 'even' });
  assert.deepEqual(laid.map((e) => e.side), ['up', 'down', 'up', 'down']);
  assert.ok(laid.every((e) => e.width === laid[0].width));
  assert.ok(laid[0].width > 0 && laid[0].width <= 28);
});

test('horizontalLayout honours an explicit side per event', () => {
  const events = [{ date: '2020', side: 'up' }, { date: '2021', side: 'up' }, { date: '2022', side: 'down' }];
  const laid = horizontalLayout(events);
  assert.deepEqual(laid.map((e) => e.side), ['up', 'up', 'down']);
});
