import { defineCollection, z } from "astro:content";
import { glob } from "astro/loaders";

// Docs are canonical Markdown at the repo root (docs/*.md), vendored into
// src/content/docs/ at build by scripts/vendor.mjs (see RELEASING.md).
const docs = defineCollection({
  loader: glob({ pattern: "*.md", base: "./src/content/docs" }),
  schema: z.object({
    title: z.string(),
    navLabel: z.string().optional(),
    description: z.string(),
    eyebrow: z.string().optional(),
    lede: z.string().optional(),
    order: z.number().default(99),
  }),
});

export const collections = { docs };
