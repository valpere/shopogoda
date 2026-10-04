# Anti-Patterns — shopogoda

Approaches that keep failing or are explicitly banned. When about to try one — stop.

## From CLAUDE.md
- ❌ Commit or push directly to `main` — branch-protected; always branch + PR
- ❌ Write outside this repository (e.g. ~/wrk/common) without explicit permission
- ❌ Hardcode credentials / commit `.env` — configuration comes from environment variables
- ❌ Reintroduce PostgreSQL/Redis — state is one SQLite file + in-process cache (single instance)
- ❌ Merge before review + green required CI — `gh pr merge --auto` ignores non-required checks

## Learned (auto-promoted)

---
