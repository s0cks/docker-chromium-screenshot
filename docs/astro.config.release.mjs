import { defineConfig } from "astro/config";
import shared from "./astro.config.shared.mjs";

export default defineConfig({
  base: "/docker-chromium-screenshot",
  ...shared,
});
