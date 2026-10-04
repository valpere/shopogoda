#!/usr/bin/env bash
# Stop hook — auto-logs a win/mistake from the just-ended session into
# _patterns/{wins,mistakes}.jsonl. Fails open: any error must never block
# session end. Mirrors session-end.sh's transcript extraction + model chain.
set -uo pipefail

source "$(dirname "$0")/_lib/hook-common.sh" 2>/dev/null || true
if declare -f hook_setup_logging >/dev/null 2>&1; then hook_setup_logging "self-learn-autolog.sh"; fi
LOG_FILE="${LOG_FILE:-$HOME/.cache/shopogoda/hooks.log}"
mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
INPUT=$(cat)

TRANSCRIPT=$(echo "$INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('transcript_path',''))" 2>/dev/null || echo "")
if [[ -z "$TRANSCRIPT" || ! -f "$TRANSCRIPT" ]]; then
  PROJECT_HASH=$(pwd | sed 's|/|-|g')
  TRANSCRIPT=$(ls -t "$HOME/.claude/projects/$PROJECT_HASH"/*.jsonl 2>/dev/null | head -1 || echo "")
fi
[[ -z "$TRANSCRIPT" || ! -f "$TRANSCRIPT" ]] && exit 0

EXCERPT=$(python3 - "$TRANSCRIPT" <<'PYEOF'
import json, sys
msgs = []
with open(sys.argv[1]) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            m = json.loads(line).get('message', {})
            role = m.get('role', '')
            if role not in ('user', 'assistant'):
                continue
            c = m.get('content', '')
            t = ''
            if isinstance(c, list):
                t = ''.join(b.get('text', '') for b in c if isinstance(b, dict) and b.get('type') == 'text')
            elif isinstance(c, str):
                t = c
            t = t.strip()
            if t:
                msgs.append(f"[{role}]: {t[:600]}")
        except Exception:
            pass
print('\n\n'.join(msgs[-40:]))
PYEOF
)
[[ -z "$EXCERPT" ]] && exit 0

TS=$(date -u +%FT%TZ)
PROMPT="Below is an excerpt of a coding session. Did anything match a WIN (a technique/pattern that worked well and is reusable) or a MISTAKE (a non-obvious error, caught late, worth remembering)?
If YES, emit exactly one JSON line and nothing else:
{\"ts\":\"$TS\",\"project\":\"shopogoda\",\"kind\":\"win\",\"pattern\":\"<short label>\",\"detail\":\"<1-2 sentences>\",\"context\":\"<file or area>\"}
(use \"mistake\" instead of \"win\" for a mistake). If NO, emit nothing at all — no prose, no markdown.

SESSION:
$EXCERPT"

RESULT=""
if command -v agy >/dev/null 2>&1; then
  RESULT=$(timeout 45 agy -p "$PROMPT" --model "Gemini 3.8 Flash (Low)" 2>>"$LOG_FILE" || true)
fi
if [[ -z "$RESULT" ]] && command -v opencode >/dev/null 2>&1; then
  RESULT=$(timeout 45 opencode run "$PROMPT" --model ollama/qwen3-coder-next:cloud 2>>"$LOG_FILE" || true)
fi

LINE=$(printf '%s\n' "$RESULT" | grep -o '{[^}]*}' | tail -1)
[[ -z "$LINE" ]] && exit 0

KIND=$(printf '%s' "$LINE" | python3 -c 'import sys,json
try:
    o=json.loads(sys.stdin.read())
    k=o.get("kind")
    if k in ("win","mistake") and o.get("pattern"): print(k)
except Exception: pass' 2>/dev/null || true)

case "$KIND" in
  win)     printf '%s\n' "$LINE" >> "$PROJECT_DIR/_patterns/wins.jsonl";     echo "[$TS] autolog: +win" >> "$LOG_FILE" ;;
  mistake) printf '%s\n' "$LINE" >> "$PROJECT_DIR/_patterns/mistakes.jsonl"; echo "[$TS] autolog: +mistake" >> "$LOG_FILE" ;;
esac
exit 0
