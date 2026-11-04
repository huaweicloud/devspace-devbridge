/**
 * Custom npm audit wrapper.
 *
 * Runs `npm audit --json`, then filters out advisories listed in
 * IGNORED_ADVISORIES (by GHSA ID). A vulnerability is considered
 * "ignored" only if every advisory in its dependency chain traces
 * back to an ignored ID.
 *
 * Exits non-zero if any un-ignored vulnerabilities remain.
 *
 * Rationale: some advisories have no patched version (e.g. braces
 * GHSA-vfj7-8cjw-p6xm). This lets CI pass while still catching
 * everything else. When a fix becomes available, remove the ID.
 */

import { execFileSync } from "node:child_process";

/** GHSA IDs we knowingly accept (no patched version available). */
const IGNORED_ADVISORIES = new Set([
  "GHSA-vfj7-8cjw-p6xm", // braces ReDoS — dev-only, no fix available
]);

const MIN_SEVERITY = "moderate";

/** Extract GHSA ID from advisory URL like https://github.com/advisories/GHSA-xxxx */
function extractGhsa(url) {
  if (!url) return null;
  const match = url.match(/(GHSA-[a-z0-9-]+)/i);
  return match ? match[1] : null;
}

function runAudit() {
  try {
    const raw = execFileSync(
      "npm",
      ["audit", "--json", `--audit-level=${MIN_SEVERITY}`],
      { encoding: "utf8", maxBuffer: 10 * 1024 * 1024 },
    );
    return JSON.parse(raw);
  } catch (err) {
    // npm audit exits non-zero when vulnerabilities are found,
    // but stdout still contains the JSON payload.
    if (err.stdout) {
      return JSON.parse(err.stdout);
    }
    throw err;
  }
}

const report = runAudit();
const vulns = report.vulnerabilities ?? {};

/**
 * Recursively resolve all advisory GHSA IDs for a package,
 * following `via` string references up the dependency chain.
 * Returns a Set of GHSA IDs (or null entries for unknown advisories).
 */
function resolveAdvisories(name, visited = new Set()) {
  if (visited.has(name)) return new Set(); // cycle guard
  visited.add(name);

  const info = vulns[name];
  if (!info) return new Set();

  const ids = new Set();
  for (const adv of info.via ?? []) {
    if (typeof adv === "object" && adv !== null) {
      // Direct advisory — extract GHSA from URL or source
      const ghsa = extractGhsa(adv.url) || String(adv.source);
      if (ghsa) ids.add(ghsa);
    } else if (typeof adv === "string") {
      // Reference to another vulnerable package — recurse
      for (const id of resolveAdvisories(adv, visited)) {
        ids.add(id);
      }
    }
  }
  return ids;
}

// Classify each vulnerability
const remaining = [];
const ignoredNames = [];

for (const [name, info] of Object.entries(vulns)) {
  const advisoryIds = resolveAdvisories(name);
  const isFullyIgnored =
    advisoryIds.size > 0 &&
    [...advisoryIds].every((id) => IGNORED_ADVISORIES.has(id));

  if (isFullyIgnored) {
    ignoredNames.push(name);
  } else {
    remaining.push(name);
  }
}

if (remaining.length === 0) {
  if (ignoredNames.length > 0) {
    console.log(
      `✅ npm audit passed — ${ignoredNames.length} package(s) with ignored advisories (no fix available):`,
    );
    console.log(`   ${ignoredNames.join(", ")}`);
    console.log(`   Ignored: ${[...IGNORED_ADVISORIES].join(", ")}`);
  } else {
    console.log("✅ npm audit passed — no vulnerabilities found.");
  }
  process.exit(0);
} else {
  console.error(
    `❌ npm audit failed — ${remaining.length} vulnerabilities remain after filtering:`,
  );
  for (const name of remaining) {
    const info = vulns[name];
    const severity = info.severity ?? "unknown";
    const ids = resolveAdvisories(name);
    console.error(
      `   ${name} (${severity}) — ${[...ids].join(", ") || "see npm audit for details"}`,
    );
  }
  process.exit(1);
}
