// Vendors canonical repo sources into the site. Runs as both `predev` and
// `prebuild`; pass `--check` (CI) to fail when the committed copies have drifted.
//
// The canonical files are the single source of truth:
//   ../../packaging/net-install.sh -> public/install.sh        (served at /install.sh)
//   ../../docs/*.md                -> src/content/docs/*.md     (rendered at /docs/*)
//   ../../CHANGELOG.md             -> src/generated/changelog.md (rendered at /changelog)
//
// The vendored copies are committed (the Docker build uses context ./web and can't
// reach ../docs). `--check` re-vendors and `git diff`s the destinations this script
// owns — so adding a vendored source is a one-line manifest edit and the CI drift
// check covers it automatically. If a source is missing (e.g. the Docker build,
// which has no ../docs), we warn and keep the committed copy.

import { execFileSync } from "node:child_process";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url)); // web/scripts
const root = resolve(here, "../.."); // repo root
const web = resolve(here, ".."); // web

// The site page supplies its own heading, so drop the changelog's editing preamble
// (everything up to the marker) and render only the version sections.
const stripChangelogPreamble = (md) => {
  const marker = "<!-- next-release -->";
  const i = md.indexOf(marker);
  return i === -1 ? md : md.slice(i + marker.length).replace(/^\s+/, "");
};

// Single source of truth for what gets vendored; --check derives its paths from this.
const FILES = [
  {
    label: "install.sh",
    src: join(root, "packaging/net-install.sh"),
    dest: join(web, "public/install.sh"),
  },
  {
    label: "changelog",
    src: join(root, "CHANGELOG.md"),
    dest: join(web, "src/generated/changelog.md"),
    transform: stripChangelogPreamble,
  },
];
const DOCS = { label: "docs", src: join(root, "docs"), dest: join(web, "src/content/docs") };
const DEST_PATHS = [...FILES.map((f) => f.dest), DOCS.dest];

function vendorFile({ label, src, dest, transform }) {
  if (!existsSync(src)) {
    console.warn(`[vendor] ${label}: source missing (${src}) — keeping committed copy`);
    return;
  }
  mkdirSync(dirname(dest), { recursive: true });
  if (transform) writeFileSync(dest, transform(readFileSync(src, "utf8")));
  else copyFileSync(src, dest);
  console.log(`[vendor] ${label}: ${src} -> ${dest}`);
}

// Subdirs to never publish (internal engineering notes live in docs/internal).
const DOCS_SKIP_DIRS = new Set(["internal"]);

function vendorDocs({ label, src, dest }) {
  if (!existsSync(src)) {
    console.warn(`[vendor] ${label}: source missing (${src}) — keeping committed copies`);
    return;
  }
  // Wipe and recopy so renames/removals (incl. in subdirs) never linger.
  rmSync(dest, { recursive: true, force: true });
  mkdirSync(dest, { recursive: true });
  let n = 0;
  const walk = (rel) => {
    for (const entry of readdirSync(join(src, rel), { withFileTypes: true })) {
      const next = rel ? join(rel, entry.name) : entry.name;
      if (entry.isDirectory()) {
        if (!DOCS_SKIP_DIRS.has(entry.name)) walk(next);
      } else if (entry.name.endsWith(".md")) {
        const out = join(dest, next);
        mkdirSync(dirname(out), { recursive: true });
        copyFileSync(join(src, next), out);
        n++;
      }
    }
  };
  walk("");
  console.log(`[vendor] ${label}: ${n} file(s) -> ${dest}`);
}

for (const f of FILES) vendorFile(f);
vendorDocs(DOCS);

// CI: fail if the committed copies drifted from the canonical sources.
if (process.argv.includes("--check")) {
  try {
    execFileSync("git", ["diff", "--quiet", "--", ...DEST_PATHS], { cwd: root, stdio: "ignore" });
    console.log("[vendor] check: vendored copies are in sync");
  } catch {
    console.error(
      "[vendor] check: vendored copies are OUT OF DATE — run 'npm --prefix web run build' (or 'node web/scripts/vendor.mjs') and commit:",
    );
    try {
      execFileSync("git", ["--no-pager", "diff", "--stat", "--", ...DEST_PATHS], {
        cwd: root,
        stdio: "inherit",
      });
    } catch {
      /* diff --stat is best-effort output only */
    }
    process.exit(1);
  }
}
