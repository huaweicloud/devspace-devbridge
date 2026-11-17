#!/usr/bin/env bash
# Static check for issue #51: cli/install.ps1 -Help must print usage and
# return early so no install actions (download/extract/install/PATH) run.
#
# Runs on Linux CI (no Pester/Windows runner needed). Wire it into a workflow
# step, e.g.:
#   - name: Static check (install.ps1 -Help)
#     run: cli/scripts/check-install-ps1-help.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PS1_FILE="$SCRIPT_DIR/../install.ps1"

python3 - "$PS1_FILE" <<'EOF'
import re, sys

src = open(sys.argv[1], encoding='utf-8-sig').read()
m = re.search(r'if \(\$Help\)\s*\{([^{}]*)\}', src, re.S)
if not m:
    sys.exit('FAIL: if ($Help) block not found in cli/install.ps1')
body = m.group(1)
if 'Show-Usage' not in body:
    sys.exit('FAIL: Show-Usage missing from the -Help block')
if not re.search(r'\breturn\b', body):
    sys.exit('FAIL: -Help block must return so no install actions run')
if re.search(r'\bexit\b', body):
    sys.exit('FAIL: -Help block must not use exit (irm | iex would close the caller session)')
print('OK: install.ps1 -Help prints usage and returns early')
EOF
