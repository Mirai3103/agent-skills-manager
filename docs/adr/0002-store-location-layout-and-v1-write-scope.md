# 0002 — Store location, layout, and what v1 may write

Status: accepted (2026-11-03)
Applies to: ADR-0001

## Context

ADR-0001 fixed the model (store + relative links, link presence = enabled). It left open where the store lives, how a skill is named inside it, and what the app is allowed to write in v1. Those three decide whether the first version can be trusted with a user's existing skills.

## Decision

1. **Location.** `~/.agent-skills/store/<name>/`. Visible and inspectable on purpose: this is a developer tool, and a user who wants to add or edit a skill by hand should be able to `cd` there. Not XDG, not inside an agent home.
2. **Layout.** Flat, one directory per skill. No source- or agent-based nesting — an agent has to see `<skills root>/<name>`, so any hierarchy inside the store would have to be flattened again at link time.
3. **Name.** The directory name is the *Skill*'s name, and it is the same as the folder name the agents already see. If `SKILL.md` frontmatter `name` disagrees, the folder name still wins and the mismatch is surfaced as a warning — renaming behind the user's back would silently move the skill between agent namespaces.
4. **Adopt is the only v1 write besides links.** Copy the loose skill into the store, verify the copy's `SKILL.md` hash matches the source, then replace each loose original with a relative *Link*. Per-skill, user-triggered, never automatic during a *Scan*.
5. **A *Variant* blocks adopt.** If another root holds a same-named skill with a different hash, adopt refuses and asks the user to pick the canonical copy first. This is why there is no `.variants` staging directory — the unresolved case never enters the store. After the pick, the losing copies are left exactly where they are: the app never renames or `.disabled`-suffixes them, because a hidden directory is still one the agent may read. A column still holding a differing copy keeps a conflict status and refuses enable until the user deals with that path themselves.
6. **v1 deletes nothing.** No store GC, no removal of *Orphan links*' targets, no cleanup of loose skills. Disabling removes a link and nothing else. A user's skill content is only ever moved by an explicit adopt.
7. **No database.** The filesystem is the state; a *Scan* re-derives the inventory and hashes `SKILL.md` in place. The only persisted file is `~/.agent-skills/config.json`, holding extra *Skills roots* the adapter list does not know about.

## Consequences

- Someone who edits `SKILL.md` by hand changes every agent at once — intended, but the UI should show the hash so a surprise change is visible.
- Adopt is not atomic across several roots: a crash between two link replacements leaves some roots linked and some not. A *Scan* still reports the truth, so recovery is re-running adopt, not a transaction log. A crash between moving an original aside and linking it leaves `<name>.adopt-old` beside the absent skill: rename it back, then adopt again.
- Refusing to adopt while a Variant exists means the user sometimes has to resolve a conflict before getting any value out of the tool. Cheaper than a staging concept nobody asked for.
- Hashing on every scan is O(skill bytes). Revisit with a cache only if a real inventory feels slow.

## Rejected alternatives

- **Nested `store/<source>/<name>`** — forces dedupe decisions at link time, which is exactly where they are hardest to undo.
- **SQLite inventory** — a second source of truth that can disagree with disk; the whole point is that it cannot.
- **Auto-adopt on first scan** — irreversibly rearranges the user's home before they have seen the inventory.
