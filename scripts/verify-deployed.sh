#!/usr/bin/env bash
# Retired compatibility entrypoint.
#
# Production deployment gates and their only asset baseline live in the source
# tree that produced the candidate package. Keeping a second checker/baseline
# here previously allowed one tree to validate another tree's package.
set -euo pipefail

cat >&2 <<'EOF'
This outer verify-deployed.sh entrypoint is retired.

Run the authoritative scripts from the package source tree instead:
  scripts/deploy-production.sh
  scripts/verify-deployed-pre.sh
  scripts/verify-deployed-post.sh

For the current production package source tree:
  F:/CodexTemp/reseller-restore-worktree

No production check was run and no baseline was changed.
EOF
exit 2
