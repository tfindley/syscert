#!/usr/bin/env bash
#
# scripts/prerelease.sh — audit release readiness, decide the next version from
# Conventional Commits, scaffold the CHANGELOG entry, and emit RELEASE-READINESS.md.
#
# Safe to run repeatedly: it builds/tests/scans and writes RELEASE-READINESS.md +
# a CHANGELOG.md draft section. It NEVER tags or pushes — scripts/release.sh does.
#
#   scripts/prerelease.sh                 # infer the next version from commits
#   scripts/prerelease.sh minor           # force a bump level
#   scripts/prerelease.sh 0.2.0           # force an explicit version
#   scripts/prerelease.sh --attest-security   # mark the manual security review done
#   scripts/prerelease.sh --skip-scan     # skip govulncheck/gosec/syft (faster)
#
# gate() always returns 0, so the `cmd && gate PASS || gate FAIL` idiom throughout
# is a safe if-then-else (the FAIL branch only runs when cmd fails), not the SC2015
# trap — disable that one check file-wide.
# shellcheck disable=SC2015
set -uo pipefail   # NB: not -e — we run every gate and aggregate results.

cd "$(git rev-parse --show-toplevel)" || exit 1

# ── args ────────────────────────────────────────────────────────────────────
BUMP_ARG="" ATTEST_SEC=no SKIP_SCAN=no
for a in "$@"; do
  case "$a" in
    --attest-security) ATTEST_SEC=yes ;;
    --skip-scan)       SKIP_SCAN=yes ;;
    -h|--help)         sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    -*)                echo "unknown flag: $a" >&2; exit 2 ;;
    *)                 BUMP_ARG="$a" ;;
  esac
done

# ── gate framework ──────────────────────────────────────────────────────────
PASS=0 FAIL=0 WARN=0
declare -a ROWS
gate() { # <PASS|WARN|FAIL> <name> [detail]
  local s=$1 n=$2 d=${3:-} c
  case $s in PASS) PASS=$((PASS+1)) c=32 ;; WARN) WARN=$((WARN+1)) c=33 ;; FAIL) FAIL=$((FAIL+1)) c=31 ;; esac
  ROWS+=("$s|$n|$d")
  printf "\033[1;%sm%-4s\033[0m %-22s %s\n" "$c" "$s" "$n" "$d"
}
have() { command -v "$1" >/dev/null 2>&1; }
run()  { "$@" >/tmp/pre.out 2>&1; }   # capture last command output for detail

echo "── SysCert prerelease audit ──"

# ── version from Conventional Commits ───────────────────────────────────────
LAST_TAG=$(git tag --list 'v[0-9]*' --sort=-v:refname | head -n1)
RANGE=${LAST_TAG:+$LAST_TAG..}HEAD
mapfile -t SUBJECTS < <(git log --no-merges --format='%s' "$RANGE" 2>/dev/null)
BODIES=$(git log --no-merges --format='%B' "$RANGE" 2>/dev/null)

# Conventional Commit types (keep in step with .githooks/commit-msg).
TYPES="feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert"

nfeat=0 nfix=0 nbreak=0 nother=0
for s in "${SUBJECTS[@]}"; do
  # web/site/ci-scoped commits ship in the website image or the release tooling,
  # not the binary, so they must NOT move the binary release version. Skip them
  # from the bump tally (a feat(web) is still a real feature, just not a binary one).
  printf '%s' "$s" | grep -Eq '^(feat|fix|perf|refactor)\((web|site|ci)\)!?:' && continue
  case $s in *"!:"*) nbreak=$((nbreak+1)) ;; esac
  if   printf '%s' "$s" | grep -Eq '^feat(\(.+\))?!?:'; then nfeat=$((nfeat+1))
  elif printf '%s' "$s" | grep -Eq '^fix(\(.+\))?!?:';  then nfix=$((nfix+1))
  elif printf '%s' "$s" | grep -Eq "^($TYPES)(\(.+\))?!?:"; then :
  else nother=$((nother+1)); fi
done
printf '%s' "$BODIES" | grep -q 'BREAKING CHANGE' && nbreak=$((nbreak+1))

base=${LAST_TAG#v}; base=${base:-0.0.0}
IFS=. read -r MA MI PA <<<"$base"
explicit=""
case $BUMP_ARG in
  major|minor|patch) BUMP="$BUMP_ARG" ;;
  v[0-9]*|[0-9]*)    explicit=${BUMP_ARG#v} ;;
  "")                if   [ "$nbreak" -gt 0 ]; then BUMP="major"
                     elif [ "$nfeat"  -gt 0 ]; then BUMP="minor"
                     else BUMP="patch"; fi ;;
  *) echo "error: bad version arg '$BUMP_ARG'" >&2; exit 2 ;;
esac
if [ -n "$explicit" ]; then NEW=$explicit
else case $BUMP in
  major) NEW="$((MA+1)).0.0" ;;
  minor) NEW="$MA.$((MI+1)).0" ;;
  patch) NEW="$MA.$MI.$((PA+1))" ;;
esac; fi
TAG="v$NEW"
ncommits=${#SUBJECTS[@]}
echo "  since ${LAST_TAG:-<root>}: $ncommits commits — ${nfeat} feat, ${nfix} fix, ${nbreak} breaking"
echo "  proposed version: $TAG"
[ "${MA}" = 0 ] && [ "${nbreak}" -gt 0 ] && [ -z "$explicit" ] && \
  echo "  note: pre-1.0 + breaking change → defaulted to major ($TAG); pass an explicit version to override"
echo

# ── gates: build / test / lint ──────────────────────────────────────────────
run go build -o /tmp/syscert.pre ./cmd/syscert && gate PASS "go build" || gate FAIL "go build" "$(tail -1 /tmp/pre.out)"
run go test ./...   && gate PASS "go test"  || gate FAIL "go test" "$(tail -1 /tmp/pre.out)"
run go vet ./...    && gate PASS "go vet"   || gate FAIL "go vet" "$(tail -1 /tmp/pre.out)"
fmt=$(git ls-files '*.go' | xargs -r gofmt -l 2>/dev/null); [ -z "$fmt" ] && gate PASS "gofmt" || gate FAIL "gofmt" "unformatted: $(echo "$fmt" | tr '\n' ' ')"

if have shellcheck; then
  run shellcheck packaging/*.sh scripts/*.sh && gate PASS "shellcheck" || gate FAIL "shellcheck" "$(grep -c '\^' /tmp/pre.out 2>/dev/null) issue(s)"
else gate WARN "shellcheck" "not installed"; fi

if have npm; then
  ( cd web && { [ -d node_modules ] || npm ci --silent; } && npm run build ) >/tmp/pre.out 2>&1 \
    && gate PASS "web build" || gate FAIL "web build" "$(tail -1 /tmp/pre.out)"
else gate WARN "web build" "npm not installed"; fi

# ── gates: tool ⇄ docs parity ───────────────────────────────────────────────
if [ -x /tmp/syscert.pre ]; then
  cmds=$(/tmp/syscert.pre --help 2>&1 | awk '/^Commands:/{f=1;next} f&&/^$/{f=0} f&&/^  [a-z]/{print $1}')
  miss=""
  # documented = mentioned in the canonical user docs (README or any docs page).
  # Recursive: docs has sub-pages (advanced-install/, procedures/, containerisation/),
  # and a flag or command documented only there was previously invisible here.
  # shellcheck disable=SC2046
  docfiles="README.md $(find docs -name '*.md' -not -path 'docs/internal/*' | tr '\n' ' ')"
  # shellcheck disable=SC2086
  for c in $cmds; do
    grep -q "$c" $docfiles 2>/dev/null || miss="$miss $c"
  done
  [ -z "$miss" ] && gate PASS "cmd↔docs parity" "$(echo "$cmds" | wc -w | tr -d ' ') commands" \
                 || gate FAIL "cmd↔docs parity" "undocumented:$miss"

  # backward parity: a command *named in an invocation* in the docs (`syscert <word> -…`)
  # must actually exist — catches a documented-but-nonexistent command, e.g. the old
  # `syscert ensure --config …` (ensure is the bare default, not a subcommand). Scans doc
  # subdirs too (that bug lived in docs/advanced-install/cron.md); skips internal notes.
  allowed=" $(echo "$cmds" | tr '\n' ' ')version help syscert "  # space-join $cmds (awk emits one per line) so the case-match below works
  bogus=""
  # Word-splitting is intentional: $(find …) is a file list passed to grep, and the
  # loop iterates over command words.
  # shellcheck disable=SC2013,SC2046
  for n in $(grep -hoE 'syscert [a-z][a-z0-9-]+ +-' README.md $(find docs -name '*.md' -not -path 'docs/internal/*') 2>/dev/null | awk '{print $2}' | sort -u); do
    case "$allowed" in *" $n "*) ;; *) bogus="$bogus $n" ;; esac
  done
  [ -z "$bogus" ] && gate PASS "docs→cmd parity (backward)" \
                 || gate FAIL "docs→cmd parity (backward)" "docs name non-commands:$bogus"

  fmiss=""
  # shellcheck disable=SC2086
  for f in --config --staging --force --config-only --env-file --interval --keep-account --ca-file --write --dirs; do
    grep -q -- "$f" $docfiles 2>/dev/null || fmiss="$fmiss $f"
  done
  [ -z "$fmiss" ] && gate PASS "flag↔docs parity" || gate WARN "flag↔docs parity" "check:$fmiss"
else
  gate WARN "cmd↔docs parity" "binary not built"
fi

# ── gate: example configs validate (host-FQDN failures are WARN, not FAIL) ───
exfail="" exwarn=""
if [ -x /tmp/syscert.pre ]; then
  for f in examples/*.toml; do
    if ! out=$(/tmp/syscert.pre dry-run --config-only --config "$f" 2>&1); then
      echo "$out" | grep -q "is not a FQDN" && exwarn="$exwarn ${f##*/}" || exfail="$exfail ${f##*/}"
    fi
  done
  if   [ -n "$exfail" ]; then gate FAIL "examples validate" "broken:$exfail"
  elif [ -n "$exwarn" ]; then gate WARN "examples validate" "host has no FQDN:$exwarn"
  else gate PASS "examples validate"; fi
else gate WARN "examples validate" "binary not built"; fi

# ── gate: version stamping works for the new tag ────────────────────────────
if go build -ldflags "-X main.version=$TAG" -o /tmp/syscert.stamp ./cmd/syscert 2>/dev/null \
   && /tmp/syscert.stamp version 2>&1 | grep -q "$TAG"; then
  gate PASS "version stamp" "$TAG"
else gate FAIL "version stamp" "binary does not report $TAG"; fi

# ── gate: release asset names match what docs reference ─────────────────────
assets="syscert-linux-amd64 syscert-linux-arm64 sha256sums.txt"
amiss=""
for a in $assets; do
  grep -q "$a" .github/workflows/release.yml || amiss="$amiss $a(workflow)"
  grep -rq "$a" README.md docs/ 2>/dev/null || amiss="$amiss $a(docs)" # -r: docs has sub-pages now
done
[ -z "$amiss" ] && gate PASS "asset names" || gate WARN "asset names" "mismatch:$amiss"

# ── gates: security / supply chain ──────────────────────────────────────────
if [ "$SKIP_SCAN" = yes ]; then
  gate WARN "govulncheck" "skipped (--skip-scan)"
  gate WARN "gosec" "skipped (--skip-scan)"
  gate WARN "SBOM (syft)" "skipped (--skip-scan)"
else
  if have govulncheck; then VC=(govulncheck ./...); else VC=(go run golang.org/x/vuln/cmd/govulncheck@latest ./...); fi
  if run "${VC[@]}"; then gate PASS "govulncheck"
  elif grep -qi 'vulnerab' /tmp/pre.out; then gate FAIL "govulncheck" "vulnerabilities found"
  else gate WARN "govulncheck" "could not run (network/tool)"; fi

  if have gosec; then run gosec -quiet -exclude-dir=.venv ./... && gate PASS "gosec" || gate FAIL "gosec" "findings"; else gate WARN "gosec" "not installed"; fi

  if have syft; then syft packages dir:. -o spdx-json=dist/sbom.spdx.json >/dev/null 2>&1 && gate PASS "SBOM (syft)" "dist/sbom.spdx.json" || gate WARN "SBOM (syft)" "generation failed"
  else gate WARN "SBOM (syft)" "not installed"; fi
fi

if have npm; then
  ( cd web && npm audit --audit-level=high ) >/tmp/pre.out 2>&1 \
    && gate PASS "npm audit" || gate WARN "npm audit" "$(grep -Eo '[0-9]+ (high|critical)' /tmp/pre.out | head -1)"
else gate WARN "npm audit" "npm not installed"; fi

if have systemd-analyze; then
  run systemd-analyze verify packaging/systemd/syscert.service packaging/systemd/syscert.timer \
    && gate PASS "systemd verify" || gate WARN "systemd verify" "review: $(tail -1 /tmp/pre.out)"
else gate WARN "systemd verify" "systemd-analyze not present"; fi

# ── gates: community health ─────────────────────────────────────────────────
miss=""
for f in LICENSE SECURITY.md README.md; do [ -f "$f" ] || miss="$miss $f"; done
[ -z "$miss" ] && gate PASS "repo health" || gate FAIL "repo health" "missing:$miss"

# ── changelog scaffold (preserve human edits) ───────────────────────────────
PLACEHOLDER="_TODO: summarise the security impact + risk of this release._"
have_section() { grep -q "^## \[$TAG\]" CHANGELOG.md 2>/dev/null; }
if ! have_section; then
  commitlog=$(git log --no-merges --format='%s (%h)' "$RANGE")
  {
    printf '## [%s] — %s\n' "$TAG" "$(date +%Y-%m-%d)"
    section() { # title regex
      local out; out=$(printf '%s\n' "$commitlog" | grep -E "^$2")
      [ -n "$out" ] && printf '\n### %s\n\n%s\n' "$1" "$out"
    }
    section "Features"      'feat(\(.+\))?!?:'
    section "Bug Fixes"     'fix(\(.+\))?!?:'
    section "Performance"   'perf(\(.+\))?!?:'
    section "Refactors"     'refactor(\(.+\))?!?:'
    section "Documentation" 'docs(\(.+\))?!?:'
    other=$(printf '%s\n' "$commitlog" | grep -vE "^($TYPES)(\(.+\))?!?:")
    [ -n "$other" ] && printf '\n### Other\n\n%s\n' "$other"
    printf '\n### Risk & Security\n\n%s\n\n' "$PLACEHOLDER"
  } > /tmp/pre.section
  # insert after the <!-- next-release --> marker
  awk -v f=/tmp/pre.section '1;/<!-- next-release -->/{while((getline l < f)>0) print l; print ""}' \
    CHANGELOG.md > /tmp/pre.changelog && mv /tmp/pre.changelog CHANGELOG.md
  echo "  wrote CHANGELOG.md draft section for $TAG"
fi

# ── gates: risk note + security review attestation ──────────────────────────
# extract the TAG section (heading → next "## " heading); index() avoids regex/bracket escaping
section_text=$(awk -v tag="$TAG" '
  index($0,"## ")==1 { if (seen) exit; if (index($0,tag)>0) seen=1 }
  seen' CHANGELOG.md)
if printf '%s' "$section_text" | grep -qF "$PLACEHOLDER"; then
  gate FAIL "risk analysis" "fill the Risk & Security note in CHANGELOG.md ($TAG)"
elif printf '%s' "$section_text" | grep -qiE 'risk|security'; then
  gate PASS "risk analysis"
else gate WARN "risk analysis" "no Risk/Security note found for $TAG"; fi

if [ "$ATTEST_SEC" = yes ]; then gate PASS "security review" "attested (/security-review run on $RANGE)"
else gate FAIL "security review" "run /security-review on $RANGE, then re-run with --attest-security"; fi

# ── report ──────────────────────────────────────────────────────────────────
verdict="READY"; [ "$FAIL" -gt 0 ] && verdict="NOT READY"
{
  echo "# Release readiness — $TAG"
  echo
  echo "_Generated $(date -u +%Y-%m-%dT%H:%MZ) · range \`${RANGE}\` · $ncommits commits._"
  echo
  echo "**Verdict: $verdict** — $PASS passed, $WARN warnings, $FAIL failed."
  echo
  echo "| | Gate | Detail |"
  echo "|---|---|---|"
  for r in "${ROWS[@]}"; do
    IFS='|' read -r s n d <<< "$r"
    case $s in PASS) i="✅";; WARN) i="⚠️";; FAIL) i="❌";; esac
    echo "| $i | $n | ${d:-} |"
  done
  echo
  echo "## Next steps"
  echo
  echo "1. Resolve every ❌ above (warnings are advisory)."
  echo "2. Fill the **Risk & Security** note in \`CHANGELOG.md\` for $TAG."
  echo "3. Run \`/security-review\` on \`$RANGE\`, then re-run \`scripts/prerelease.sh --attest-security\`."
  echo "4. Commit the prep, then: \`scripts/release.sh $TAG\`"
} > RELEASE-READINESS.md

echo
echo "── $verdict ──  $PASS passed · $WARN warn · $FAIL failed"
echo "report: RELEASE-READINESS.md   ·   next: scripts/release.sh $TAG"
[ "$FAIL" -gt 0 ] && exit 1 || exit 0
