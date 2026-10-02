// Isomorphic (server + browser) Markdown helpers. Kept separate from
// dossier.js, which imports node:fs/node:path and can't be bundled for the
// client renderer (src/client/).
import { marked } from 'marked';

export const md = (s = '') => marked.parse(String(s), { async: false, gfm: true });
export const mdi = (s = '') => marked.parseInline(String(s), { async: false, gfm: true });
