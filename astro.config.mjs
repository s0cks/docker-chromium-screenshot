import fs from 'node:fs';
import path from 'node:path';
import { defineConfig } from 'astro/config';

const cardsDir = path.resolve(process.env.CARDS_DIR ?? 'cards');

// Full-reload the dev server when a card changes: cards/ lives outside src/,
// so Vite would not notice edits to it on its own.
const reloadOnCardChange = {
  name: 'card-reload',
  configureServer(server) {
    server.watcher.add(cardsDir);
    const reload = (file) => {
      if (file.startsWith(cardsDir)) server.ws.send({ type: 'full-reload' });
    };
    server.watcher.on('change', reload);
    server.watcher.on('add', reload);
  },
};

// Serve cards/ itself at /cards/* during `astro dev` — a card's logo.*,
// theme.css and any other file it carries — matching what the Go service
// (and render.mjs's fallback server) serve in every other environment. See
// docs/build-a-card.md.
const mountCards = {
  name: 'mount-cards',
  configureServer(server) {
    server.middlewares.use((req, res, next) => {
      if (!req.url?.startsWith('/cards/')) return next();
      const file = path.join(cardsDir, req.url.slice('/cards/'.length).split('?')[0]);
      if (file.startsWith(cardsDir) && fs.existsSync(file) && fs.statSync(file).isFile()) {
        return void fs.createReadStream(file).pipe(res);
      }
      next();
    });
  },
};

export default defineConfig({
  output: 'static',
  devToolbar: { enabled: false },
  vite: { plugins: [reloadOnCardChange, mountCards] },
});
