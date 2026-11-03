# Glossary

Terms this project uses for its domain. Agents and humans both read this; use these words, not synonyms.

## Skill

A directory containing a `SKILL.md` (plus optional assets) that a coding agent loads on demand. The unit this app manages.

## Agent home

A per-agent data directory that the agent itself owns and reads, e.g. `~/.claude`, `~/.agents`, `~/.gemini`, `~/.codex`. **Avoid**: "config dir", "profile".

## Skills root

The `skills/` directory inside an agent home (`~/.claude/skills`). Every agent home on this machine has its own, which is why the same skill exists in several places.

## Skill store

The one app-owned directory holding a single canonical copy of each managed skill. Not inside any agent home. **Avoid**: "library", "vault", "registry", "repo".

## Link

A symlink from `<skills root>/<name>` to `<store>/<name>`. Targets are **relative**, so moving the store or the home does not break them. This is what puts a skill in front of an agent.

## Enabled / Disabled

**Enabled** = a link exists in that agent's skills root. **Disabled** = no link. The filesystem is the state; the app has no separate toggle to drift from it.

## Loose skill

A skill directory that exists in an agent home but has no copy in the store. Everything starts loose.

## Adopted skill

A skill whose canonical copy lives in the store and which agents reach through a link.

## Duplicate

Two or more skills in different roots whose `SKILL.md` byte hash is equal. Duplicates are safe to collapse.

## Variant

Two or more skills sharing a folder name but differing in `SKILL.md` hash. Not safe to collapse — a human decides which one is canonical.

## Adapter

Built-in knowledge of one agent: which *Skills roots* it reads, and whether each is *Linkable*. Adding support for an agent means adding an adapter, and its root claims carry a cited source.

## Reader

An agent that reads a given *Skills root*. Roots have readers, not the other way round — `~/.agents/skills` has three. Inverting the adapters' root claims is the only place this map exists.

## Shared root

A *Skills root* with more than one *Reader*. One link in a shared root serves every reader at once.

## Read-only root

A *Skills root* the app scans but never writes, because another tool owns it — Claude Code's account-synced store is the one on this machine. Unlinkable.

## Linkable

Said of a *Skills root* the app is allowed to write links into. Every root is linkable unless it is *read-only*.

## Double visibility

One agent seeing the same *Skill* through two roots. Warned before it happens: Codex loads it twice, OpenCode's winner is non-deterministic, Gemini silently shadows one copy, and only Claude Code deduplicates by link target.

## Root override

A skills root the user adds by hand through `~/.agent-skills/config.json`, because no adapter knows about it. The only thing the app persists.

## Orphan link

A link whose store entry no longer exists. Reported, never silently deleted.

## Scan

A read-only traversal of the detected skills roots and the *Skill store* that produces the inventory. Scanning never mutates anything.

## Non-goals (v1)

MCP servers and per-project skill management are out of scope for v1; the link model must not make them harder later.
