import assert from 'node:assert/strict';
import { donutArcs, fmt, niceAxis } from '../src/lib/chart.js';
import { test } from 'node:test';

test('niceAxis returns a d3 scale "niced" to round ticks that cover the data', () => {
  const a = niceAxis(87);
  assert.equal(a.min, 0);
  assert.ok(a.max >= 87);
  assert.equal(typeof a.scale, 'function');
  assert.ok(a.ticks.every((t, i) => i === 0 || t > a.ticks[i - 1]));
  assert.deepEqual(niceAxis(100, 4, { min: 70, max: 100 }).ticks, [70, 80, 90, 100]);
  assert.equal(niceAxis(0).max, 1);
});

test('fmt compacts and decorates numbers', () => {
  assert.equal(fmt(12400), '12.4k');
  assert.equal(fmt(3_400_000, { prefix: '$' }), '$3.4M');
  assert.equal(fmt(81.4, { suffix: '%', compact: false }), '81.4%');
  assert.equal(fmt(14.2, { prefix: '€', suffix: 'M', decimals: 1 }), '€14.2M');
  assert.equal(fmt(undefined), '');
});

test('donutArcs produces one arc path per value via d3-shape', () => {
  const arcs = donutArcs([54, 21, 15, 10], { innerRadius: 40, outerRadius: 78 });
  assert.equal(arcs.length, 4);
  for (const a of arcs) {
    assert.equal(typeof a.path, 'string');
    assert.match(a.path, /^M/);
  }
  // ascending value order in the input keeps arcs in the same angular order (sort: null)
  assert.ok(arcs[0].midAngle < arcs[3].midAngle);
});
