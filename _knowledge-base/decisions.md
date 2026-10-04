# Architecture Decision Records

| Date | Decision | Context | Alternatives Considered |
|------|----------|---------|------------------------|
| 2026-10-04 | Replace PostgreSQL+Redis with SQLite + in-process cache | One binary + one file on a small VM; no production data to keep | Keep Postgres/Redis (hosted free tiers) |
