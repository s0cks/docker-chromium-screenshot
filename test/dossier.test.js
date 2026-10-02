import assert from 'node:assert/strict';
import { test } from 'node:test';
import { parseAttrs, parseDossier, parseSize } from '../src/lib/dossier.js';

const doc = (body, meta = '') => parseDossier(`---\nseries: Test\n${meta}---\n${body}`, { id: 'test' });

test('parseAttrs handles flags, bare values and quoted strings', () => {
  assert.deepEqual(parseAttrs('tone=red mono w=half title="Two words"'), { tone: 'red', mono: true, w: 'half', title: 'Two words' });
  assert.deepEqual(parseAttrs('cols=3 labels=false'), { cols: '3', labels: false });
});

test('parseSize accepts presets and WxH', () => {
  assert.deepEqual(parseSize(), [1600, 1000]);
  assert.deepEqual(parseSize('square'), [1080, 1080]);
  assert.deepEqual(parseSize('1200x630'), [1200, 630]);
  assert.throws(() => parseSize('huge'), /invalid size/);
});

test('slides split on --- but not inside code fences', () => {
  const d = doc('## One\n\n```\n---\n```\n\n---\n\n## Two\n');
  assert.equal(d.slides.length, 2);
  assert.equal(d.slides[0].title, 'One');
  assert.equal(d.slides[1].title, 'Two');
});

test('+++ splits lead and main; layout follows', () => {
  const split = doc('## T\nlead\n\n+++\n\n### S\nmain\n');
  assert.equal(split.slides[0].layout, 'split');
  assert.equal(split.slides[0].lead.length, 1);
  assert.equal(split.slides[0].main.length, 1);
  assert.equal(doc('## T\nbody\n').slides[0].layout, 'full');
});

test('the first h1/h2 becomes the title and is removed from the body', () => {
  const s = doc('## The Title\n\nBody text.\n').slides[0];
  assert.equal(s.title, 'The Title');
  assert.ok(!s.lead.concat(s.main).some((n) => n.kind === 'md' && n.html.includes('The Title')));
  assert.equal(s.kicker, 'The Title');
});

test('yaml blocks parse to data, md blocks keep their body, attrs are read', () => {
  const s = doc('## T\n```stats cols=3\n- value: 1\n  label: a\n```\n```callout tone=red mono\n**hi**\n```\n').slides[0];
  const [stats, callout] = s.lead;
  assert.equal(stats.name, 'stats');
  assert.equal(stats.attrs.cols, '3');
  assert.deepEqual(stats.data, [{ value: 1, label: 'a' }]);
  assert.equal(callout.md, '**hi**');
  assert.equal(callout.attrs.mono, true);
});

test('unregistered fences stay ordinary Markdown code', () => {
  const s = doc('## T\n```js\nconst a = 1;\n```\n').slides[0];
  assert.equal(s.lead.length, 1);
  assert.equal(s.lead[0].kind, 'md');
  assert.match(s.lead[0].html, /<code/);
});

test('the slide fence sets per-slide config', () => {
  const s = doc('```slide\nkicker: Custom\nlayout: split\nlead: 30\n```\n## T\n').slides[0];
  assert.equal(s.kicker, 'Custom');
  assert.equal(s.layout, 'split');
  assert.equal(s.config.lead, 30);
});

test('invalid YAML names the file, slide and block', () => {
  assert.throws(() => doc('## T\n```stats\n- value: [unclosed\n```\n'), /test.*slide 1.*stats.*invalid YAML/s);
});

test('@include expands snippets and rejects unknown ones', () => {
  const d = doc('## T\n@include note\n', 'snippets:\n  note: Shared note\n');
  assert.match(d.slides[0].lead[0].html, /Shared note/);
  assert.throws(() => doc('## T\n@include nope\n'), /unknown snippet "nope"/);
});

test('word budget and missing titles produce warnings', () => {
  const long = Array.from({ length: 40 }, () => 'word').join(' ');
  const d = doc(`## T\n${long}\n`, 'budget: 10\n');
  assert.equal(d.warnings.length, 1);
  assert.match(d.warnings[0], /40 words exceeds the budget of 10/);
  assert.match(doc('no heading here\n').warnings.join(), /no title/);
});

test('an empty dossier is an error', () => {
  assert.throws(() => doc('\n'), /no slides/);
});
