#!/usr/bin/env bash
# minions/council/run.sh — thin wrapper over the canonical council
# (~/wrk/common_claude/minions/council: agy, opencode, codex, mirror; read-only).
# The only local part is the framing presets in presets/*.md.
#
# Usage:
#   minions/council/run.sh [PRESET] [council options] (-p "task" | TASK-FILE | -)
#
#   PRESET   free (default) | review | design | docs | decision — prepended to the task
#   options  passed through unchanged: -C -a -t -o -d --diff-limit --diff-path
#            -f -r --retries --strict --dry-run (see `council --help`)
set -euo pipefail

COUNCIL="${COUNCIL:-$HOME/wrk/common_claude/minions/council}"
PRESET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/presets"
die() { echo "council/run.sh: $*" >&2; exit 2; }

[[ -x "$COUNCIL" ]] || die "canonical council not found: $COUNCIL"
case "${1:-}" in -h|--help) sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'; exec "$COUNCIL" --help ;; esac

preset=free
if [[ $# -gt 0 && "$1" != -* && -f "$PRESET_DIR/$1.md" ]]; then preset="$1"; shift; fi

pass=() task="" src=""
while (($#)); do
	case "$1" in
	-p|--prompt) task="${2:?}"; shift 2 ;;
	-C|--cwd|-a|--agents|-t|--timeout|-o|--out|-d|--diff|--diff-limit|--diff-path|-f|--file|-r|--ref|--retries)
		pass+=("$1" "${2:?}"); shift 2 ;;
	-) src="-"; shift ;;
	-*) pass+=("$1"); shift ;;
	*) src="$1"; shift ;;
	esac
done
if [[ -z "$task" ]]; then
	[[ -n "$src" ]] || die "no task (-p TEXT, TASK-FILE or -)"
	if [[ "$src" == "-" ]]; then task="$(cat)"; else task="$(cat -- "$src")"; fi
fi

{ cat "$PRESET_DIR/$preset.md"; printf '\n## TASK\n\n%s\n' "$task"; } | exec "$COUNCIL" "${pass[@]}" -
