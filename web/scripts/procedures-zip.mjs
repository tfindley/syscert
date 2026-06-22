// Postbuild: bundle the operating procedures into dist/downloads/syscert-procedures.zip.
//
// One Markdown file per procedure, named by its Procedure ID (SC-OPS-00X-<slug>.md),
// with the Astro YAML frontmatter stripped and a standalone H1 title prepended — so the
// files drop cleanly into a business's own operational documentation. The zip is a build
// artifact written into dist/ (served by nginx at /downloads/…); it is never committed.
//
// Reads the vendored SOP sub-pages at src/content/docs/procedures/ (present in both local
// and the Docker build context); the index page (src/content/docs/procedures.md) is excluded.
import { readFileSync, readdirSync, mkdirSync, existsSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import AdmZip from "adm-zip";

const here = dirname(fileURLToPath(import.meta.url));
const srcDir = join(here, "..", "src", "content", "docs", "procedures");
const outDir = join(here, "..", "dist", "downloads");
const outZip = join(outDir, "syscert-procedures.zip");

if (!existsSync(srcDir)) {
  console.warn("procedures-zip: no procedures dir — skipping");
  process.exit(0);
}

const zip = new AdmZip();
let count = 0;
for (const file of readdirSync(srcDir).filter((f) => f.endsWith(".md")).sort()) {
  const raw = readFileSync(join(srcDir, file), "utf8");
  const fm = raw.match(/^---\n([\s\S]*?)\n---\n?/); // leading YAML frontmatter block
  if (!fm) {
    console.warn(`procedures-zip: ${file} has no frontmatter — skipping`);
    continue;
  }
  const body = raw.slice(fm[0].length).replace(/^\s+/, "");
  const titleMatch = fm[1].match(/^title:\s*["']?(.+?)["']?\s*$/m);
  const title = titleMatch ? titleMatch[1] : file.replace(/\.md$/, "");
  const id = (title.match(/SC-OPS-\d+/) || [])[0]; // e.g. SC-OPS-001
  const slug = file.replace(/\.md$/, "");
  const name = id ? `${id}-${slug}.md` : `${slug}.md`;
  zip.addFile(name, Buffer.from(`# ${title}\n\n${body}`, "utf8"));
  count++;
}

mkdirSync(outDir, { recursive: true });
zip.writeZip(outZip);
console.log(`procedures-zip: wrote ${count} procedure(s) → dist/downloads/syscert-procedures.zip`);
