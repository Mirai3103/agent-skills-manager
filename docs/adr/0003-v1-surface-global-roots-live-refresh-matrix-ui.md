# 0003 — v1 surface: global roots only, live refresh, matrix UI

Status: accepted (2026-11-03)
Applies to: ADR-0001, ADR-0002

## Context

Three surfaces were still open after ADR-0002: which *Skills roots* v1 reads, how the inventory stays current, and the shape of the main view. All three leak into the backend contract (which roots the adapter list exposes, whether the app pushes update events, whether enable/disable acts per agent), so they get decided before the spec, not during it.

## Decision

1. **Global roots only.** v1 scans the skills roots inside *Agent homes* and nothing else. Project-local roots (a repo's `.agents/skills`) are out of scope: no evidence yet on how an agent ranks a project skill against a global one, and a link written into a repo would fight whatever installed the skill there (`skills-lock.json` in this very repo).
2. **Live refresh, with a manual fallback.** A watcher on each detected *Skills root* and on the store, debounced, pushes an updated inventory to the UI. A visible Rescan action exists and is the whole story if the watcher fails — the app must never depend on the watcher for correctness.
3. **Matrix as the main view.** One row per *Skill*, one column per detected *Skills root*, the cell is the enable state at that root. *(Amended by ADR-0004: the original said one column per agent home; roots are shared by several agents, so a column per agent would have shown independent toggles for one physical link.)* Each column header names the root and its *Readers*. Cells encode three states by shape, not colour alone: enabled (*Link* present), loose (real directory, not adopted), absent. Row headers and column headers are semantic table headers, and every toggle carries an accessible name naming both the skill and the root.
4. **Status is a column, not a set of tabs.** *Loose*, *Duplicate*, *Variant*, *Orphan link* appear as a per-row status. Filtering is a query, not navigation; nothing hides a skill the user might be looking for.

## Consequences

- Adding project support later adds roots to the scan, not a second model — the *Link* primitive is unchanged.
- A watcher that misses an event shows a stale matrix. Acceptable because every write path re-scans its own result before reporting success; the matrix is a view of the last known state, never the authority.
- Density 8 with a matrix caps usability at roughly six root columns before they crowd. Past that, the same data renders as a per-skill detail list; the backend contract does not change.
- Every cell being independently addressable means enable/disable must be a per-(skill, root) operation in the service API from day one — batching it later would be a breaking change.

## Rejected alternatives

- **List + detail pane** — requires a click per skill to answer "who sees this", which is the question the app exists to answer.
- **Tabs per status** — hides rows, and the interesting rows are the conflicting ones.
- **Polling instead of watching** — makes every edit feel stale for one interval; the user's complaint is about a messy filesystem they cannot see.
