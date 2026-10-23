import fs from "node:fs";
import path from "node:path";
import { defineConfig } from "astro/config";
import { goMarkdownPlugin } from "./plugin/vite-plugin-compiler";

const cardsDir = path.resolve(process.env.CARDS_DIR ?? "cards");

// Full-reload the dev server when a card changes: cards/ lives outside src/,
// so Vite would not notice edits to it on its own.
const reloadOnCardChange = {
  name: "card-reload",
  configureServer(server) {
    server.watcher.add(`${cardsDir}/**/*`);
    const reload = (file) => {
      if (file.startsWith(cardsDir)) server.ws.send({ type: "full-reload" });
    };

    server.watcher.on("change", reload);
    server.watcher.on("add", reload);
  },
};

export default defineConfig({
  output: "static",
  devToolbar: { enabled: false },
  vite: {
    plugins: [
      goMarkdownPlugin({
        compilerPath: "./compiler/bin/compiler",
      }),
      reloadOnCardChange,
    ],
  },
});
