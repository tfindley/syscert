// Build-time release info. Resolves the latest GitHub release's tag and the
// sha256 checksums of its Linux binaries, so the site can show a REAL, verifiable
// digest instead of a hardcoded one.
//
// Rate-limit-free: uses the public `releases/latest` redirect + the release asset
// download, not the GitHub API. Fail-safe: on any error (offline build, no release
// yet, timeout) it falls back to the static SITE.version and null checksums, so the
// build never breaks and components simply omit the live bits.
//
// Freshness: a new release's sha256sums.txt only exists once CI has finished
// building, so the site is rebuilt on `release: published` (see web.yml) — that
// build is what picks up the new digest.

import { SITE, LINKS } from "../consts";

export interface ReleaseInfo {
  /** e.g. "v0.0.6" — falls back to SITE.version when the live lookup fails. */
  version: string;
  /** Per-arch sha256 of syscert-linux-<arch>, or null when unavailable. */
  sha256: { amd64?: string; arm64?: string } | null;
}

// Reuse the canonical releases URL (LINKS.releases = https://github.com/<owner>/<repo>/releases).
const RELEASES = LINKS.releases;
const TIMEOUT_MS = 6000;

let cached: Promise<ReleaseInfo> | null = null;

/** Resolve once per build; reused across every component import. */
export function getRelease(): Promise<ReleaseInfo> {
  cached ??= load();
  return cached;
}

/** First 8 + last 7 hex chars, for a compact display digest. */
export function shortSha(hex: string): string {
  return hex.length > 19 ? `${hex.slice(0, 8)}…${hex.slice(-7)}` : hex;
}

async function load(): Promise<ReleaseInfo> {
  try {
    const signal = AbortSignal.timeout(TIMEOUT_MS);
    const [version, sha256] = await Promise.all([resolveVersion(signal), fetchSums(signal)]);
    return { version, sha256 };
  } catch (err) {
    console.warn(`[release] live lookup failed, using static fallback (${SITE.version}): ${err}`);
    return { version: SITE.version, sha256: null };
  }
}

async function resolveVersion(signal: AbortSignal): Promise<string> {
  // Prefer a build-time injection (SITE_VERSION, set by web.yml). This is deterministic
  // and — because it's a Docker build-arg — busts the layer cache per release, so a
  // rebuild can't reuse a layer that baked the previous version. The live fetch below is
  // the fallback for local builds (`npm run dev`/`build` with no SITE_VERSION).
  const injected = process.env.SITE_VERSION;
  if (injected && /^v[0-9]/.test(injected)) return injected;

  // GET /releases/latest follows a redirect to /releases/tag/vX.Y.Z.
  const res = await fetch(`${RELEASES}/latest`, { redirect: "follow", signal });
  const m = res.url.match(/\/tag\/(v[0-9][^/]*)$/);
  return m ? m[1] : SITE.version;
}

async function fetchSums(signal: AbortSignal): Promise<ReleaseInfo["sha256"]> {
  const res = await fetch(`${RELEASES}/latest/download/sha256sums.txt`, { signal });
  if (!res.ok) throw new Error(`sha256sums.txt → HTTP ${res.status}`);
  const out: { amd64?: string; arm64?: string } = {};
  for (const line of (await res.text()).split("\n")) {
    const [hash, name] = line.trim().split(/\s+/);
    if (!hash || !name) continue;
    if (name === "syscert-linux-amd64") out.amd64 = hash;
    else if (name === "syscert-linux-arm64") out.arm64 = hash;
  }
  if (!out.amd64 && !out.arm64) throw new Error("no syscert-linux-* entries in sha256sums.txt");
  return out;
}
