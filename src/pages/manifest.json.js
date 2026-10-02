import { loadDossiers, pad } from '../lib/dossier.js';

// Consumed by scripts/render.mjs to know what to capture.
export function GET() {
  const dossiers = loadDossiers().map((d) => ({
    id: d.id,
    title: d.meta.title,
    theme: d.meta.theme,
    size: d.size,
    warnings: d.warnings,
    slides: d.slides.map((s) => ({ index: s.index + 1, title: s.title, words: s.words, path: `/${d.id}/${pad(s.index + 1)}/` })),
  }));

  return new Response(JSON.stringify({ dossiers }, null, 2), { headers: { 'content-type': 'application/json' } });
}
