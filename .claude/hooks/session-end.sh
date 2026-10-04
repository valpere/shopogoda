#!/bin/bash
# Stop hook: appends/updates today's entry in .claude/session-log.md.
#
# Behaviour:
#   - Skip if /session-end skill wrote the file within the last 2 hours
#   - Generate summary: agy (Gemini) → opencode (cloud) → raw excerpt
#   - If session-log.md already has an entry for today, replace it
#   - Otherwise append; rotate to keep last 10 entries
#
# Output: .claude/session-log.md (gitignored, per-project)

set -uo pipefail

# shellcheck source=_lib/hook-common.sh
source "$(dirname "$0")/_lib/hook-common.sh"
hook_setup_logging "session-end.sh"

INPUT=$(cat)
echo "[$(date -Iseconds)] session-end invoked" >> "$LOG_FILE"

LOG="$(dirname "$0")/../session-log.md"
TODAY=$(date '+%Y-%m-%d')

# Skip if /session-end skill already ran this session (file modified < 2h ago)
if [[ -f "$LOG" ]]; then
  AGE=$(( $(date +%s) - $(stat -c %Y "$LOG" 2>/dev/null || echo 0) ))
  if [[ $AGE -lt 7200 ]]; then
    echo "[$(date -Iseconds)] session-end: skill already ran (${AGE}s ago), skipping" >> "$LOG_FILE"
    exit 0
  fi
fi

# Locate session transcript
TRANSCRIPT=$(echo "$INPUT" | python3 -c \
  "import json,sys; d=json.load(sys.stdin); print(d.get('transcript_path',''))" 2>/dev/null || echo "")

if [[ -z "$TRANSCRIPT" || ! -f "$TRANSCRIPT" ]]; then
  PROJECT_HASH=$(pwd | sed 's|/|-|g')
  TRANSCRIPT=$(ls -t "$HOME/.claude/projects/$PROJECT_HASH"/*.jsonl 2>/dev/null | head -1 || echo "")
fi

if [[ -z "$TRANSCRIPT" || ! -f "$TRANSCRIPT" ]]; then
  echo "[$(date -Iseconds)] session-end: no transcript found, skipping" >> "$LOG_FILE"
  exit 0
fi

# Extract last 30 exchanges from JSONL
EXCERPT=$(python3 - "$TRANSCRIPT" <<'PYEOF'
import json, sys

messages = []
with open(sys.argv[1]) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            entry = json.loads(line)
            msg = entry.get('message', {})
            role = msg.get('role', '')
            content = msg.get('content', '')
            if role not in ('user', 'assistant'):
                continue
            text = ''
            if isinstance(content, list):
                for block in content:
                    if isinstance(block, dict) and block.get('type') == 'text':
                        text += block.get('text', '')
            elif isinstance(content, str):
                text = content
            text = text.strip()
            if text:
                messages.append(f"[{role}]: {text[:600]}")
        except Exception:
            pass

print('\n\n'.join(messages[-30:]))
PYEOF
)

if [[ -z "$EXCERPT" ]]; then
  echo "[$(date -Iseconds)] session-end: empty transcript, skipping" >> "$LOG_FILE"
  exit 0
fi

PROMPT="Write a session summary. Use exactly this structure (no preamble, start directly with the header):

## $TODAY

### Що зробили
- completed items

### Поточний стан
- current branch / open PR / what works / what is broken

### Відкриті питання
- unresolved questions (omit this section entirely if none)

### Наступні кроки
- what to pick up next session, in priority order

Rules: 10-20 bullets total, Ukrainian for content, English for code/file names/identifiers.

SESSION TRANSCRIPT:
$EXCERPT"

# --- LLM call helpers ---

# is_valid_summary rejects anything that doesn't start with the expected
# "## $TODAY" header — guards against a model ignoring the prompt and
# returning generic chit-chat (observed: agy silently dropping the prompt
# when passed via stdin, e.g. "I am currently running on Gemini 3.5
# Flash..."). A non-empty result is not enough on its own; it must also
# look like the summary we asked for, or the next tier should be tried.
is_valid_summary() {
  [[ "$1" == "## $TODAY"* ]]
}

try_agy() {
  # Refreshed 2026-09-04 — the 3.5 series is retired (live "invalid model
  # selection" error, confirmed against agy's own --model error listing,
  # which starts at 3.6). Cheap-to-capable ordering preserved.
  local models=(
    "Gemini 3.8 Flash (Low)"
    "Gemini 3.8 Flash (Medium)"
    "Gemini 3.8 Flash (High)"
    "Gemini 3.1 Pro (Low)"
    "Gemini 3.1 Pro (High)"
  )
  command -v agy &>/dev/null || return 1
  for model in "${models[@]}"; do
    echo "[$(date -Iseconds)] session-end: trying agy model: $model" >> "$LOG_FILE"
    local result
    # Prompt as a positional arg, not stdin — `agy -p` reads the prompt
    # from its argument; piping via `<<<` leaves it unset and agy falls
    # back to a generic interactive-style greeting instead of erroring.
    result=$(timeout 45 agy -p "$1" --model "$model" 2>>"$LOG_FILE") && \
      is_valid_summary "$result" && { echo "$result"; return 0; }
    echo "[$(date -Iseconds)] session-end: agy model $model returned no usable summary" >> "$LOG_FILE"
  done
  return 1
}

# Write excerpt to temp file for opencode --file
EXCERPT_TMP=$(mktemp /tmp/session-end-excerpt.XXXXXX)
echo "$EXCERPT" > "$EXCERPT_TMP"
trap 'rm -f "$EXCERPT_TMP"' EXIT

OPENCODE_MSG="Summarize the session transcript in the attached file. Use exactly this structure (no preamble):

## $TODAY

### Що зробили
- completed items

### Поточний стан
- current branch / open PR / what works / what is broken

### Відкриті питання
- unresolved questions (omit section if none)

### Наступні кроки
- what to pick up next session

Rules: 10-20 bullets total, Ukrainian for content, English for code/file names."

try_opencode() {
  # 2026-09-27: qwen3.5:cloud retired 2026-09-25 (announced 2026-09-17
  # alongside glm-5.1/deepseek-v4-flash:0731, both already swapped in
  # fix-review/session-maintain by a765ee7 — this file was missed in
  # that pass). Canonical 1:1 replacement per
  # dreaming/ollama-model-retirement-instructions.md: glm-5.3-flash or
  # deepseek-v4.1-flash; picked deepseek-v4.1-flash to avoid a second
  # glm entry alongside glm-5.2 above.
  local models=(
    "ollama/glm-5.2:cloud"
    "ollama/kimi-k2.6:cloud"
    "ollama/minimax-m2.7:cloud"
    "ollama/deepseek-v4.1-flash:cloud"
  )
  command -v opencode &>/dev/null || return 1
  for model in "${models[@]}"; do
    echo "[$(date -Iseconds)] session-end: trying opencode model: $model" >> "$LOG_FILE"
    local result
    result=$(timeout 45 opencode run "$OPENCODE_MSG" \
      --file "$EXCERPT_TMP" --model "$model" --format json 2>>"$LOG_FILE" | \
      python3 -c "
import json,sys
parts=[]
for line in sys.stdin:
    line=line.strip()
    if not line: continue
    try:
        obj=json.loads(line)
        if obj.get('type')=='text':
            parts.append(obj['part']['text'])
    except: pass
print(''.join(parts))
") && is_valid_summary "$result" && { echo "$result"; return 0; }
    echo "[$(date -Iseconds)] session-end: opencode model $model returned no usable summary" >> "$LOG_FILE"
  done
  return 1
}

# --- Generate summary ---

SUMMARY=""

SUMMARY=$(try_agy "$PROMPT" 2>>"$LOG_FILE") || true

if [[ -z "$SUMMARY" ]]; then
  SUMMARY=$(try_opencode 2>>"$LOG_FILE") || true
fi

if [[ -z "$SUMMARY" ]]; then
  echo "[$(date -Iseconds)] session-end: all LLMs failed, using raw excerpt" >> "$LOG_FILE"
  SUMMARY="## $TODAY

### Transcript excerpt (auto-summary unavailable)

\`\`\`
$(echo "$EXCERPT" | head -60)
\`\`\`"
fi

# --- Append/update/rotate session-log.md ---

python3 - "$LOG" "$TODAY" "$SUMMARY" <<'PYEOF'
import sys, re, os

log_file = sys.argv[1]
today    = sys.argv[2]
entry    = sys.argv[3].strip()
max_keep = 10

if not os.path.exists(log_file):
    with open(log_file, 'w') as f:
        f.write(entry + '\n')
    sys.exit(0)

with open(log_file) as f:
    content = f.read()

# Split on ## YYYY-MM-DD headers; keep each header with its content
parts = re.split(r'(?m)(?=^## \d{4}-\d{2}-\d{2})', content)
entries = [p.strip() for p in parts if p.strip()]

today_header = f'## {today}'
today_idx = next((i for i, e in enumerate(entries) if e.startswith(today_header)), -1)
if today_idx >= 0:
    entries.pop(today_idx)
    entries.append(entry)           # replace today's entry, moved to end
else:
    entries.append(entry)           # new day

entries = entries[-max_keep:]    # rotate — freshest write can never be trimmed away

with open(log_file, 'w') as f:
    f.write('\n\n'.join(entries) + '\n')
PYEOF

echo "[$(date -Iseconds)] session-end: wrote $(wc -l < "$LOG") lines to $LOG" >> "$LOG_FILE"
