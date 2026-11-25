# council — project presets over the canonical council

`run.sh` is a thin wrapper over the canonical
`~/wrk/common_claude/minions/council` (agents `agy`, `opencode`, `codex`,
`mirror`; read-only; see `council --help`). The only local part is the framing
in `presets/*.md`, prepended to the task.

```bash
R=minions/council/run.sh

$R decision -p "Should this service use SQLite or Postgres?" -f docs/architecture.md
$R review -d main...HEAD --diff-path internal/ -p "Review this change"
$R review -r HEAD -p "Find coupling problems in internal/services"   # committed tree only
$R docs -f README.md -f docs/DEPLOYMENT.md -p "Do the docs contradict each other?"
echo "Critique this reply" | $R docs -
$R review --dry-run -d HEAD -p "..."                                 # build prompt.md only
```

First argument: preset `free` (default), `review`, `design`, `docs`, `decision`.
Everything else goes to the canonical council unchanged (`-a`, `-t`, `-o`, `-d`,
`--diff-path`, `-f`, `-r`, `--retries`, `--strict`, `--dry-run`, `-C`).
Output: `tmp/council/<UTC>/` (`prompt.md`, `<agent>.md`, `summary.md`).
Unadjudicated: agreement between agents is a prior, not proof.

Set `COUNCIL=/path/to/council` to use another copy of the canonical script.
