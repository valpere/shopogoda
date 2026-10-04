#!/usr/bin/env bash
# minions/council/run.sh — general-purpose council: fan ONE task out to several
# independent third-party coding-agent CLIs in parallel and collect their reports.
#
# Works for any task you can phrase as a prompt: code review, design critique,
# doc/draft review, "which option?" decisions, free-form questions. It is NOT
# tied to a repository — run it from anywhere.
#
# Isolation: every agent runs with cwd = OUTDIR/work, a directory that holds
# only what you attached (inlined into prompt.md). Nothing else from your
# machine is in its working directory. With -w the cwd is instead a disposable
# detached git worktree of HEAD (for tasks that need to browse the repo).
# Each agent is started in the CLI's read-only / plan mode where one exists and
# the prompt forbids edits. That is best effort, not a sandbox guarantee.
#
# Usage:
#   council/run.sh [options] [TASK TEXT...]
#   echo "question" | council/run.sh [options]
#
#   -p, --prompt TEXT        the task (alternatively positional words or stdin)
#   -f, --prompt-file FILE   the task, read from a file
#   -c, --context PATH       attach a file or directory (text files only); repeatable
#   -g, --git-range RANGE    attach `git log` + `git diff` for RANGE (e.g. main..HEAD)
#   -P, --pathspec "PATHS"   limit --git-range to these paths (space-separated)
#   -s, --preset NAME        framing: free (default) | review | design | docs | decision
#   -a, --agents LIST        comma list (default: opencode,kilo,kiro-cli,agy)
#   -t, --timeout SECS       per-agent timeout (default: 600, env COUNCIL_TIMEOUT)
#   -o, --out DIR            output dir (default: ./tmp/council/<UTC timestamp>)
#   -m, --max-bytes N        cap per attachment, bytes (default: 200000)
#   -w, --worktree           run agents inside a disposable git worktree of HEAD
#       --list               list known agents and presets
#   -h, --help
#
# Agents: opencode, kilo, kiro-cli, agy  (default set)
#         cursor-agent (weekly usage cap), codex (needs working bubblewrap),
#         omp (needs OpenRouter credit), dry (prints the prompt; self-test)
#
# Output: OUTDIR/prompt.md, OUTDIR/<agent>.md (stdout), <agent>.stderr.log,
#         <agent>.exit, OUTDIR/summary.md. Findings are a prior, not a verdict —
#         read, de-duplicate and adjudicate yourself.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PRESET_DIR="$HERE/presets"

TASK=""
TASK_FILE=""
CONTEXTS=()
GIT_RANGE=""
PATHSPEC=""
PRESET="free"
AGENTS="opencode,kilo,kiro-cli,agy"
TIMEOUT="${COUNCIL_TIMEOUT:-600}"
OUTDIR=""
MAX_BYTES=200000
MAX_TOTAL=600000 # cap for one attached directory
USE_WORKTREE=0

usage() { sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'; }
die() { echo "council: $*" >&2; exit 2; }

list_all() {
	echo "agents: opencode kilo kiro-cli agy cursor-agent codex omp dry"
	echo "default agents: opencode,kilo,kiro-cli,agy"
	printf 'presets:'
	for f in "$PRESET_DIR"/*.md; do printf ' %s' "$(basename "$f" .md)"; done
	echo
}

while [ $# -gt 0 ]; do
	case "$1" in
	-p | --prompt) [ $# -ge 2 ] || die "$1 needs a value"; TASK="$2"; shift 2 ;;
	-f | --prompt-file) [ $# -ge 2 ] || die "$1 needs a value"; TASK_FILE="$2"; shift 2 ;;
	-c | --context) [ $# -ge 2 ] || die "$1 needs a value"; CONTEXTS+=("$2"); shift 2 ;;
	-g | --git-range) [ $# -ge 2 ] || die "$1 needs a value"; GIT_RANGE="$2"; shift 2 ;;
	-P | --pathspec) [ $# -ge 2 ] || die "$1 needs a value"; PATHSPEC="$2"; shift 2 ;;
	-s | --preset) [ $# -ge 2 ] || die "$1 needs a value"; PRESET="$2"; shift 2 ;;
	-a | --agents) [ $# -ge 2 ] || die "$1 needs a value"; AGENTS="$2"; shift 2 ;;
	-t | --timeout) [ $# -ge 2 ] || die "$1 needs a value"; TIMEOUT="$2"; shift 2 ;;
	-o | --out) [ $# -ge 2 ] || die "$1 needs a value"; OUTDIR="$2"; shift 2 ;;
	-m | --max-bytes) [ $# -ge 2 ] || die "$1 needs a value"; MAX_BYTES="$2"; shift 2 ;;
	-w | --worktree) USE_WORKTREE=1; shift ;;
	--list) list_all; exit 0 ;;
	-h | --help) usage; exit 0 ;;
	--) shift; TASK="${TASK:+$TASK }$*"; break ;;
	-*) die "unknown option $1 (see --help)" ;;
	*) TASK="${TASK:+$TASK }$1"; shift ;;
	esac
done

[[ "$TIMEOUT" =~ ^[0-9]+$ ]] || die "timeout must be a number of seconds"
[[ "$MAX_BYTES" =~ ^[0-9]+$ ]] || die "max-bytes must be a number"
case "$GIT_RANGE" in -*) die "git range must not start with -" ;; esac
[ -f "$PRESET_DIR/$PRESET.md" ] || die "unknown preset '$PRESET' (see --list)"

if [ -n "$TASK_FILE" ]; then
	[ -r "$TASK_FILE" ] || die "cannot read $TASK_FILE"
	TASK="${TASK:+$TASK$'\n\n'}$(cat "$TASK_FILE")"
fi
if [ -z "$TASK" ] && { [ -p /dev/stdin ] || [ -f /dev/stdin ]; }; then
	TASK="$(cat)"
fi
[ -n "$TASK" ] || die "no task given (use -p, -f, positional text or stdin)"

[ -n "$OUTDIR" ] || OUTDIR="tmp/council/$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUTDIR/work"
OUTDIR="$(cd "$OUTDIR" && pwd)"
WORKDIR="$OUTDIR/work"

# --- build the single prompt file ------------------------------------------

# cap_file <path>: print a file, truncated to MAX_BYTES with a notice.
cap_file() {
	local size
	size="$(wc -c <"$1")"
	if [ "$size" -gt "$MAX_BYTES" ]; then
		head -c "$MAX_BYTES" "$1"
		printf '\n[... truncated: %s of %s bytes shown ...]\n' "$MAX_BYTES" "$size"
	else
		cat "$1"
	fi
}

attach_file() {
	local f="$1"
	if ! grep -Iq . "$f" 2>/dev/null; then
		echo "=== FILE: $f (skipped: binary or empty) ==="
		return
	fi
	echo "=== FILE: $f ==="
	cap_file "$f"
	echo
}

PROMPT_FILE="$OUTDIR/prompt.md"
{
	cat "$PRESET_DIR/$PRESET.md"
	echo
	echo "## TASK"
	echo
	printf '%s\n' "$TASK"

	if [ ${#CONTEXTS[@]} -gt 0 ] || [ -n "$GIT_RANGE" ]; then
		echo
		echo "## MATERIAL"
	fi

	for p in "${CONTEXTS[@]}"; do
		echo
		if [ -d "$p" ]; then
			# Skip VCS/dependency dirs and anything that looks like a secret; cap the total.
			total=0
			while IFS= read -r f; do
				sz="$(wc -c <"$f")"
				if [ $((total + sz)) -gt "$MAX_TOTAL" ]; then
					echo "=== (directory $p: remaining files skipped, total cap $MAX_TOTAL bytes reached) ==="
					break
				fi
				total=$((total + sz))
				attach_file "$f"
			done < <(find "$p" -type f \
				-not -path '*/.git/*' -not -path '*/node_modules/*' -not -path '*/vendor/*' -not -path '*/tmp/council/*' \
				-not -name '.env' -not -name '.env.*' -not -name '*.pem' -not -name '*.key' -not -name 'id_rsa*' \
				-not -name 'id_ed25519*' -not -name '*.p12' -not -name 'credentials*' -not -name '*.kdbx' \
				| LC_ALL=C sort)
		elif [ -f "$p" ]; then
			attach_file "$p"
		else
			echo "=== MISSING: $p ==="
			echo "council: context '$p' not found" >&2
		fi
	done

	if [ -n "$GIT_RANGE" ]; then
		echo
		echo "=== GIT RANGE: $GIT_RANGE ${PATHSPEC:+(paths: $PATHSPEC)} ==="
		# shellcheck disable=SC2086
		{
			git log --oneline "$GIT_RANGE" ${PATHSPEC:+-- $PATHSPEC}
			echo
			git diff "$GIT_RANGE" ${PATHSPEC:+-- $PATHSPEC}
		} >"$OUTDIR/.range.txt" 2>"$OUTDIR/.range.err" || die "git range failed: $(cat "$OUTDIR/.range.err")"
		cap_file "$OUTDIR/.range.txt"
		rm -f "$OUTDIR/.range.txt" "$OUTDIR/.range.err"
	fi
} >"$PROMPT_FILE"

# Agents see a copy of the prompt inside their cwd, so file-read tools work
# without reaching anything else.
cp "$PROMPT_FILE" "$WORKDIR/prompt.md"

# --- optional disposable worktree ------------------------------------------

AGENT_CWD="$WORKDIR"
if [ "$USE_WORKTREE" -eq 1 ]; then
	REPO_ROOT="$(git rev-parse --show-toplevel)" || die "--worktree needs a git repository"
	WT="$(mktemp -d /tmp/council-wt.XXXXXX)"
	git -C "$REPO_ROOT" worktree add --detach "$WT" HEAD >/dev/null
	cp "$PROMPT_FILE" "$WT/.council-prompt.md"
	trap 'git -C "$REPO_ROOT" worktree remove --force "$WT" >/dev/null 2>&1 || true' EXIT
	AGENT_CWD="$WT"
	PROMPT_IN_CWD="$WT/.council-prompt.md"
else
	PROMPT_IN_CWD="$WORKDIR/prompt.md"
fi

GO="Read the file $PROMPT_IN_CWD IN FULL — it is your complete instructions plus the task and material. Then produce the report it asks for. Your final chat message IS the report. Do not modify any file."

# --- agents -----------------------------------------------------------------

agent_opencode() { opencode run --agent plan --model opencode/nemotron-3-ultra-free "$GO"; }
agent_kilo() { kilo run --agent plan --model kilo/kilo-auto/free "$GO"; }
agent_cursor_agent() { cursor-agent --print --trust --mode plan --model auto "$GO"; }
agent_kiro_cli() { kiro-cli chat --no-interactive --trust-tools=fs_read "$GO"; }
agent_codex() { codex exec --sandbox read-only --skip-git-repo-check "$GO"; }
agent_omp() { omp --print --model auto "$GO"; }
# agy in headless mode cannot approve file reads: it gets the prompt inline.
# Linux caps a single argv string at 128 KiB, so inline at most 100000 bytes.
agent_agy() {
	local inline size
	size="$(wc -c <"$PROMPT_FILE")"
	inline="$(head -c 100000 "$PROMPT_FILE")"
	if [ "$size" -gt 100000 ]; then
		inline+=$'\n\n[NOTE: this prompt was truncated to its first 100000 of '"$size"$' bytes for this agent; your review covers only part of the material. Say so in your report.]'
	fi
	agy --mode plan -p "$inline"
}
# Self-test: proves the plumbing (cwd, prompt, timeout, summary) without any model.
agent_dry() { echo "dry run in $(pwd)"; head -n 20 "$PROMPT_IN_CWD"; }

export PROMPT_FILE PROMPT_IN_CWD GO

# --- fan-out ------------------------------------------------------------------

# Paths under our own output dir are expected changes, so they are filtered out.
status_snapshot() {
	local top rel
	top="$(git rev-parse --show-toplevel 2>/dev/null)" || return 0
	rel="${OUTDIR#"$top"/}"
	if [ -z "$rel" ] || [ "$rel" = "$OUTDIR" ]; then
		git status --porcelain 2>/dev/null || true # OUTDIR outside the repo or equal to it: nothing to exclude
	else
		git status --porcelain 2>/dev/null | awk -v r="$rel" 'index(substr($0, 4), r) != 1' || true
	fi
}
BEFORE="$(status_snapshot)"

echo "council: preset=$PRESET agents=$AGENTS timeout=${TIMEOUT}s out=$OUTDIR${USE_WORKTREE:+ (worktree)}"

IFS=',' read -ra LIST <<<"$AGENTS"
pids=()
names=()
for name in "${LIST[@]}"; do
	fn="agent_${name//-/_}"
	if ! declare -F "$fn" >/dev/null; then
		echo "council: unknown agent '$name' — skipping (see --list)" >&2
		continue
	fi
	# shellcheck disable=SC2163
	export -f "$fn"
	(
		set +e
		cd "$AGENT_CWD"
		start=$(date +%s)
		timeout -k 10 "$TIMEOUT" bash -c "$fn" </dev/null \
			>"$OUTDIR/$name.md" 2>"$OUTDIR/$name.stderr.log"
		echo $? >"$OUTDIR/$name.exit"
		echo $(($(date +%s) - start)) >"$OUTDIR/$name.secs"
	) &
	pids+=($!)
	names+=("$name")
	echo "council: launched $name (pid $!)"
done

for pid in "${pids[@]}"; do
	wait "$pid" || true
done

# Some CLIs (kiro-cli) emit ANSI colour codes on stdout; strip them so reports are plain text.
for name in "${names[@]}"; do
	[ -f "$OUTDIR/$name.md" ] && sed -i 's/\x1b\[[0-9;?]*[a-zA-Z]//g' "$OUTDIR/$name.md"
done

# --- summary ------------------------------------------------------------------

{
	echo "# Council summary"
	echo
	echo "- **Preset:** \`$PRESET\`"
	echo "- **Agents:** $AGENTS"
	echo "- **Prompt:** \`prompt.md\`"
	echo
	echo "| Agent | Exit | Secs | Lines | Report |"
	echo "|-------|------|------|-------|--------|"
	for name in "${names[@]}"; do
		code="$(cat "$OUTDIR/$name.exit" 2>/dev/null || echo '?')"
		secs="$(cat "$OUTDIR/$name.secs" 2>/dev/null || echo '?')"
		lines="$(wc -l <"$OUTDIR/$name.md" 2>/dev/null || echo 0)"
		note=""
		[ "$code" = "124" ] && note=" (timeout)"
		echo "| \`$name\` | $code$note | $secs | $lines | \`$name.md\` |"
	done
	echo
	echo "Unadjudicated fan-out: agreement between agents is a prior, not proof —"
	echo "re-derive each claim before acting. Exit 124 = timeout; check \`<agent>.stderr.log\`"
	echo "for rate limits, auth errors or sandbox problems."
} >"$OUTDIR/summary.md"

echo
cat "$OUTDIR/summary.md"

if git rev-parse --show-toplevel >/dev/null 2>&1; then
	AFTER="$(status_snapshot)"
	if [ "$BEFORE" != "$AFTER" ]; then
		echo
		echo "council: WARNING — 'git status' changed during the run (an agent may have written files):"
		diff <(echo "$BEFORE") <(echo "$AFTER") || true
	fi
fi
