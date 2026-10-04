# council — general-purpose multi-agent fan-out

Send one task to several independent coding-agent CLIs (opencode, kilo,
kiro-cli, agy by default) in parallel and collect their reports side by side.
Not tied to a repo or to code review: any task you can phrase as a prompt.
Unadjudicated by design — you are the arbiter; agreement is a prior, not proof.

```bash
R=minions/council/run.sh

$R -p "Should this service use SQLite or Postgres?" -s decision -c docs/architecture.md
$R -s review -g main..HEAD -P "internal/" -p "Review this change"
$R -s docs -c README.md -c docs/DEPLOYMENT.md -p "Do the docs contradict each other?"
echo "Critique this reply to the client" | $R -s docs -c chat.md
$R -s design -w -p "Find coupling problems in internal/services"   # agents may browse a worktree of HEAD
$R --list
```

| Option | Meaning |
|---|---|
| `-p/-f`, positional, stdin | the task |
| `-c PATH` (repeatable) | attach file or directory; text only, inlined into `prompt.md` and sent to third-party model endpoints, so attach only what you are willing to share. Capped by `-m` (200000 B per file, 600000 B per directory); `.git`, `node_modules`, `vendor`, `.env*`, keys and credential files are skipped in directories (not when you name a file explicitly) |
| `-g RANGE` `[-P "paths"]` | attach `git log` + `git diff` for a range |
| `-s PRESET` | framing in `presets/`: `free` (default), `review`, `design`, `docs`, `decision` |
| `-a LIST` | agents, default `opencode,kilo,kiro-cli,agy`; opt-in `cursor-agent`, `codex`, `omp`; `dry` = self-test without a model |
| `-t SECS` | per-agent timeout (default 600) |
| `-o DIR` | output dir (default `./tmp/council/<UTC>`) |
| `-w` | agents run in a disposable detached worktree of **HEAD** (uncommitted changes are not in it) instead of a bare sandbox |

## Isolation model

Each agent's cwd is `OUTDIR/work`, which contains only a copy of `prompt.md`
(everything you attached is inlined in it). Nothing else from your machine is
in its working directory. `-w` swaps that for a throwaway `git worktree` of
HEAD, removed on exit. Every CLI is started in plan/read-only mode where one
exists and the preset forbids edits — best effort, not a sandbox guarantee.
After the run, if you are inside a git repo, the script warns when
`git status` changed outside the output dir. This is a smoke alarm, not a guarantee:
it cannot see writes to git-ignored paths (including `tmp/`, where the agents' cwd lives).

## Output

`OUTDIR/prompt.md`, `<agent>.md` (ANSI stripped), `<agent>.stderr.log`,
`<agent>.exit`, `<agent>.secs`, and `summary.md` (status table). Exit 124 =
timeout; stderr usually explains rate limits or auth problems.

## Agent notes (from earlier per-project councils)

- `agy` cannot approve file reads headless, so it gets the prompt inline
  (first 100000 bytes — Linux caps one argv string at 128 KiB; a truncation note is appended to its prompt).
- `codex` needs a working bubblewrap sandbox (`--sandbox read-only`); it can
  fail inside nested sandboxes. Off by default.
- `omp` needs OpenRouter credit; `cursor-agent` has a weekly usage cap. Off by default.
- Add an agent: define `agent_<name>()` (dashes → underscores) in `run.sh`,
  using `"$GO"` (short instruction pointing at the prompt file) or `$PROMPT_FILE`.
