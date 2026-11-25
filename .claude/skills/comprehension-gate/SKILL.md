---
name: comprehension-gate
description: "Explain the consequences of a specific risky action when the user asks. Follow project approval boundaries; no automatic quizzes or repeated confirmations."
---

# Consequence explanation — compatibility entry

The former automatic comprehension gate was retired as part of the agent
configuration repair. The name remains for existing references; the old quiz,
grading, escalation loop, and mandatory override phrase no longer apply.

When the user asks about consequences, explain the concrete effect, affected
resources, and available recovery using evidence from the actual change.
Resolve an unclear scope before acting when that ambiguity affects the action.

Approval requirements come from `CLAUDE.md` / `AGENTS.md` and actual user
decisions. Preserve those boundaries and existing authorization. A clear
instruction is not an exam of the user's knowledge. This skill adds no approval
step to an assigned audit, ordinary local checks, a permitted commit/merge, or
a deployment already authorized for that target.

For SoT changes, production writes, and destructive actions, follow the
applicable project rules. If a new decision is necessary, ask about that
decision and its consequence once; do not require the user to recite code,
pass a quiz, or say a special phrase.
