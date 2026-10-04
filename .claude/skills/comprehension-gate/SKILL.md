---
name: comprehension-gate
description: "Before any hard-to-reverse action (commit/push/merge/force-push, deploy, migration, destructive query, infra/secret change, deleting files you didn't create) verify the human actually understands the consequence — ask 1-3 questions anchored in this project's real code/schema/callers, judge honestly, escalate on wrong answers, proceed only when every question is cleared. Never accept 'looks good' or 'I understand' as proof. No slash command — activates automatically when about to take a gated action."
---

# Skill: comprehension-gate
# Verify Understanding Before Irreversible Actions

Adapted from [hassanhabib/the-standard-skills](https://github.com/hassanhabib/the-standard-skills)
(`the-standard-comprehension-gate`) — stripped of C#/.NET framing, kept
language-agnostic.

---

## OVERVIEW

This sits between "I'm about to do something risky" and doing it. An agent can generate
or run a change faster than a human can absorb it — the human says "yes," it ships, and
nobody in the loop actually knew what would happen until it did. The 2017 S3 outage was
one engineer running a routine command with a wrong parameter: the machine did exactly
what it was told while no fully-present human stood between the command and its blast
radius.

**Governing principle: understanding must be demonstrated, not asserted.** "I
understand," "looks good," "just do it" are worth nothing — they're exactly what someone
says right before they break production. The only proof is answering a question that
cannot be answered *without* understanding.

This is stricter than a plain confirmation prompt. The system-level "ask before risky
actions" behavior gets you a yes/no. This skill gets you *evidence* the yes meant
something — 1–3 project-anchored questions the human must actually answer, not just
approve.

---

## WHEN TO GATE (scope)

Not "is this a git command" — three questions about the action itself. If any answer is
yes, gate it:

1. **Hard to reverse?** Can the human cleanly undo it in seconds, or does undoing require
   heroics (restore a backup, revert-and-redeploy, apologize to users)?
2. **Changes shared or persistent state?** Does it touch something other people or future
   runs depend on — a shared branch, a database, infra, a published artifact?
3. **Visible outside this machine?** Network calls, deploys, messages, published
   packages, spent money?

**Clearly gate:** push, merge, force-push, rebase onto shared branches, rewriting
published history, tags, releases; deploys, releases, publishing packages/images;
migrations, backfills, `DELETE`/`UPDATE`/`DROP`/`TRUNCATE`, bulk edits, cache/queue
purges; side-effecting scripts (write outside the repo, call a paid/external API, mutate
infra, send communications); secrets, env vars, IAM/permissions, CI/CD, feature flags,
DNS, IaC apply; deleting or overwriting files you did not create.

**Do not gate:** reading files; `git status`/`diff`/`log`; running tests, builds,
linters, type-checks; drafting or editing uncommitted code; local dev servers; anything
undoable with a keystroke that never left the machine. Gating safe actions isn't "extra
safe" — it trains people to click through, so the gate is gone when a dangerous action
wears the same clothes.

When two signals disagree (a "small" diff that drops a table), trust the **consequence**,
not the size. One line can be the most dangerous line.

---

## DEPTH — scale to risk

| Risk | Looks like | Questions | Posture |
|---|---|---|---|
| **Low** | Small commit on a feature branch; reversible local change | 1 | Quick gut-check. Vague-but-directionally-right can pass. |
| **Medium** | Push to a shared branch; routine deploy; a script hitting an external service | 2 | Expect specifics. Press once on a fuzzy answer. |
| **High** | Migrations, deletes, force-push, prod config/secrets, anything irreversible or user-facing | 3 | Hold the line. Vague answers don't pass; escalate until precise or abort. |

Bump the tier — not just the count — on amplifiers: production/shared environments, large
blast radius, no easy rollback, time pressure or fatigue (the "2am text" conditions), or
the human moving faster than they're reading.

---

## HOW TO ASK

A question is only worth asking if it's **impossible to answer correctly without
actually understanding this change.**

- **Anchor in this codebase (most important).** Real function/file names, the actual
  schema, the specific caller that breaks, the domain invariant this change touches. The
  test: *could a model with no access to this repo answer it?* If yes, it's too generic —
  rewrite it. A generic question is one paste away from another chat window doing the
  thinking instead of the human.
- **Aim at the blast radius, not trivia.** "What happens to the 40k existing rows…", not
  "what does force-push do."
- **Un-guessable from its own text.** No answer embedded in the phrasing; no yes/no; no
  leading. Prefer "what happens to…", "what state is X left in…", "what breaks if this
  fails halfway."
- **Worth holding for.** If a wrong answer wouldn't actually stop you, cut the question.

Ask, then **stop and wait.** Do not answer them yourself, do not hint, do not proceed.

---

## JUDGING & ESCALATION

For each answer: **correct**, **partial/vague**, or **wrong**. Grade on an accurate model
of the outcome, not eloquence — a confident wrong answer is the most dangerous signal
there is, because that misconception is the exact thing about to cause harm.

When an answer is wrong or vague:

1. Close the specific gap in a sentence or two — teach, don't scold. **But teaching does
   not clear the question** — you supplied the answer, which proves what *you* know. Pick
   the technique that matches *why* the gap exists (see TEACHING TECHNIQUE below) — don't
   default to a generic re-explanation.
2. Re-probe the **same gap** plus one adjacent consequence, at higher difficulty. Narrow
   in; don't wander to new topics.
3. Repeat — questions get harder, not easier, while answers stay wrong. The human climbs
   up to understanding; the action does not come down to meet them.

If after ~2–3 focused rounds the human still cannot answer, **that is the answer: do not
take the action.** Not shipping a misunderstood change is the gate succeeding, not
failing. Offer real off-ramps: walk through it together, shrink the change to something
they understand, or shelve it.

### TEACHING TECHNIQUE — match the gap, don't default to one

The gap is one of two kinds; each has a cheaper, sharper fix than a generic re-explanation:

- **Doesn't recognize a concept/pattern** (unfamiliar API, idiom, or mechanism — not
  specific to this codebase) → **ELI5**: one-sentence plain-language analogy, no jargon.
  ("A mutex is a bathroom key someone hands back when they're done.")
- **Doesn't see why the code is shaped this way** (a design/architecture decision specific
  to this diff) → **reverse-engineer**: walk the actual WHY (necessary cause) / PURPOSE
  (goal it serves) reasoning behind it. If the PR already has a Rationale section (from
  code-generator's Comment Discipline), reuse that text — don't re-derive it.

Both stay a sentence or two — this is still gap-closing, not a lecture. If unsure which
kind, ask a clarifying follow-up before picking a technique; don't guess.

### LOGGING — turn escalations into data, not vibes

If `.claude/skills/self-learn/` exists in this project: after any escalation (wrong/vague
answer triggered teaching), log one entry via `/self-learn log` — gate action type, gap
category (concept vs rationale), technique used, and the outcome (Proceed / Redesign /
Hold). Best-effort — skip silently if logging fails or self-learn isn't installed, never
block the gate on it.

This is what makes "does teaching help" answerable later instead of guessed at: run
`/self-learn retro` after enough escalations accumulate and look for the same gap category
recurring less often, or resolving faster, over time. Flat or noisy — drop the technique
distinction rather than layering more onto it.

---

## OUTCOMES

- **Proceed** — only when *every* question asked has landed a correct answer, from
  *them*, in their words, after any teaching. Not "most," not "the ones that mattered."
  Letting a wrong answer slide because it "didn't really matter" is the gate becoming
  theater.
- **Hold** — understanding not demonstrated. Do not act.
- **Redesign** — the best outcome. The human, now engaged, concludes the action is wrong
  as designed and changes it ("don't drop that column on a live table; add the new schema
  alongside it and roll over slowly"). Don't force them back onto the original questions —
  the redesign usually demonstrates deeper understanding than the questions would have.
  But a redesign clears the gate only on *concrete* reasoning — confident vocabulary
  ("just make it idempotent," "open–closed") is not a pass. Mirror the new plan back in
  specific terms and confirm the safety actually holds.

---

## CONDUCT

- **Do not leak the answer** — not in the question, not in a "hint," not in the framing.
- **Do not accept assertions of understanding.** Redirect warmly to the actual question.
- **Do not grade on a curve to reach "yes"** — in either direction (teaching-then-
  proceeding, or rubber-stamping confident-sounding architecture-speak).
- **Do not be a jerk.** A sharp colleague doing a genuine gut-check, not a bouncer. Keep
  it short; scale down for trivial changes.
- **Honor a deliberate override, but make it deliberate.** State plainly what goes
  unverified and the specific risk, and require explicit acceptance ("proceed without
  verifying — I accept the risk"). Friction, not a wall. An emergency is a reason to be
  *more* awake, not less.

---

## WORKED EXAMPLE

**Action:** run a migration adding `status NOT NULL` to `orders`, no default, ~40k live
rows. High risk — production data, irreversible.

> **Agent:** Two questions before I run this — it touches a table with 40k live rows:
> 1. The migration adds `status` as `NOT NULL` with no default. What happens to the
>    existing 40k rows the moment it runs?
> 2. If it fails partway through, what state is `orders` left in — can this be safely
>    re-run?
>
> **Developer:** It just adds the column, should be quick.
>
> **Agent:** That's the gap, and it's the dangerous one. A `NOT NULL` column with no
> default can't be added to a table that already has rows — there's no value to put in
> them, so it errors out and rolls back. What would need to change for those 40k rows to
> survive it?
>
> **Developer:** Give it a default, or add it nullable first, backfill, then set NOT NULL.
>
> **Agent:** Right — which do you want, and should the backfill be a separate step so a
> failure doesn't block the deploy?
>
> **Developer:** Add nullable, backfill `'pending'` in a follow-up, tighten to NOT NULL
> after. Ship it as two migrations.
>
> **Agent:** That's a plan you understand and can stand behind. Splitting it into the two
> migrations you described, showing you both before running anything.

The first answer was confidently wrong; the agent held, corrected the misconception in
one sentence, and escalated with a *harder* follow-up on the same gap — never proceeding
on its own explanation. The outcome improved: two safe migrations instead of one
destructive one. That's the **redesign** outcome, working as intended.

---

## ANTI-PATTERNS — a gate that became theater

- **Accepting assertion** — "This force-push will drop 5 commits on main, understand the
  impact?" / "Yep, all good." / *proceeds*. A yes/no invites exactly this; ask a question
  that *requires* the model, not permission.
- **Teaching, then proceeding** — agent explains what `--force-with-lease` does after the
  developer guesses wrong, then proceeds as if the developer had known it. That verifies
  the *agent's* understanding, not theirs.
- **Reclassifying a miss as "secondary"** — three questions asked, two landed, the third
  missed but "that's a minor detail, proceeding." If it was worth asking, a wrong answer
  blocks. If it wasn't, it should never have been asked.
- **Over-gating trivial actions** — three questions before committing a README typo fix.
  Trains the developer to bulldoze through the gate on reflex, so it's gone when a
  `DROP TABLE` wears the same clothes.
- **Rubber-stamping confident vocabulary** — "we should just make it idempotent and
  feature-flag it, open–closed" / "sounds right, proceeding." A redesign clears the gate
  only on concrete reasoning, not sophisticated-sounding words. Mirror the plan back in
  specifics and confirm the actual safety holds.
