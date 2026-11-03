# 0001 — Central skill store, symlinked into agent homes

Status: accepted (2026-11-03)

## Context

Skills for coding agents are scattered across `~/.agents`, `~/.claude`, `~/.gemini`, `~/.codex`. On this machine the same skill already exists in several of them — `skill-creator` in `~/.agents` and `~/.gemini`, `hf-cli` in `~/.claude` and `~/.gemini` — and some pairs have diverged in content. At least one agent reads more than one root, so duplicated skills load twice or the wrong copy wins.

We need one place to see every skill, one place to edit it, and a cheap way to turn it on or off per agent. Editing the same skill in four homes is the bug, not the workflow.

## Decision

1. **One store holds the Canonical copy.** The app owns a single directory; every managed *Skill* lives there exactly once. The store is not inside any *Agent home*.
2. **A *Link* is the only thing that makes a skill visible to an agent.** `<skills root>/<name>` is a symlink to `<store>/<name>`, with a **relative** target — the machine already does this by hand: `~/.claude/skills/hf-cli -> ../../.agents/skills/hf-cli`.
3. **Enable/disable is link presence.** Enabling creates the link, disabling removes it. No app-side flag, no rewriting agent config, no `.disabled` renames that agents can still pick up.
4. **Editing happens once.** Because agents read through the link, updating the store copy updates every agent that has it enabled. There is no sync step and no second copy to reconcile.
5. **Identity is the `SKILL.md` hash.** Equal hashes are *Duplicates* and collapse safely; equal names with different hashes are *Variants* and require a human choice.

## Consequences

- Deleting store content disables that skill for every agent at once — the UI must say so before any delete.
- Symlink support is **verified** for all four target agents (2026-11-03): Claude Code documents it and deduplicates by link target; Codex CLI sets `DirectorySymlinkPolicy::Follow`; Gemini CLI ships `gemini skills link`; OpenCode's glob sets `symlink: true`. Sources in `local://agent-skill-discovery.md`. An agent that does not follow symlinks still cannot be served by this model, so the adapter flag stays per root for future agents.
- Anything an agent writes *into* a skill directory writes through the link into the store. Acceptable for skills (they are read-only by contract), not acceptable for agent-generated state — the app never links a directory an agent mutates.
- A relative link assumes store and agent home sit under the same user; fine for a single-user desktop app.
- Adding per-project skill management later reuses the same primitive: a project-local `skills root` gets its own links. MCP management is unrelated and unaffected.

## Rejected alternatives

- **Copy instead of link** — recreates the divergence that motivated the project, on the first edit.
- **Manifest only, no store** — leaves duplicates in place; the app reads state it cannot fix.
- **Store as a git repo of shared skills** — a distribution story, not a management story; still needs the link layer.
