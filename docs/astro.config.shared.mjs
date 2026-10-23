import { defineConfig } from "astro/config";
import react from "@astrojs/react";
import starlight from "@astrojs/starlight";

import mdx from "@astrojs/mdx";

export default {
  markdown: {
    shikiConfig: {
      theme: "nord",
      wrap: true,
    },
  },
  integrations: [
    react(),
    starlight({
      title: "docker-chromium-screenshot",
      sidebar: [
        {
          label: "Guides",
          items: [
            {
              label: "Introduction",
              link: "/docs/introduction",
            },
            {
              label: "Getting Started",
              link: "/docs/getting-started",
            },
            {
              label: "Frontmatter",
              link: "/docs/frontmatter",
            },
            {
              label: "Callouts",
              link: "/docs/components/callouts",
            },
            {
              label: "Charts",
              link: "/docs/components/charts",
            },
            {
              label: "Findings",
              link: "/docs/components/findings",
            },
            {
              label: "Network Graphs",
              link: "/docs/components/network-graphs",
            },
            {
              label: "Profiles",
              link: "/docs/components/profiles",
            },
            {
              label: "Prose",
              link: "/docs/components/prose",
            },
            {
              label: "QRs",
              link: "/docs/components/qrs",
            },
            {
              label: "Quotes",
              link: "/docs/components/quotes",
            },
            {
              label: "Sources",
              link: "/docs/components/sources",
            },
            {
              label: "Stats",
              link: "/docs/components/stats",
            },
            {
              label: "Timelines",
              link: "/docs/components/timelines",
            },
            {
              label: "Threads",
              link: "/docs/components/threads",
            },
          ],
        },
      ],
    }),
    mdx(),
  ],
  vite: {
    css: {
      preprocessorOptions: {
        scss: {
          api: "modern-compiler",
        },
      },
    },
  },
};
