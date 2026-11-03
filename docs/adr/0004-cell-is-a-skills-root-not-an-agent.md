# 0004 — The cell is a skills root, not an agent

Status: accepted (2026-11-03)
Applies to: ADR-0001, ADR-0003
Supersedes: ADR-0003 decision 3 (one column per agent home)

## Context

Investigation of each agent's documented and source-level skill discovery (`local://agent-skill-discovery.md`) contradicts the assumption ADR-0003 was written on. Agents do not own disjoint roots — they overlap heavily:

| Skills root | Read by |
|---|---|
| `~/.agents/skills` | Codex, Gemini, OpenCode |
| `~/.claude/skills` | Claude Code, OpenCode |
| `~/.gemini/skills` | Gemini |
| `~/.codex/skills` | Codex |
| `~/.config/opencode/skills` | OpenCode |
| `~/.claude/skills/synced/<account>` | Claude Code (account-synced, 21 skills on this machine) |

So "enable this skill for Gemini" is not a thing the filesystem can express. Enabling in `~/.agents/skills` hands the skill to Codex, Gemini *and* OpenCode at once. A column per agent would have shown three independent toggles that are physically one link — the app would be lying about what a click does.

The overlap also has consequences that are mechanical, not aesthetic:

- **Codex** deduplicates by canonical path, not by name, so the same skill linked into both `~/.codex/skills` and `~/.agents/skills` loads **twice** into the catalog.
- **OpenCode** overwrites duplicate names while scanning with unbounded concurrency, so which copy wins is non-deterministic between runs.
- **Gemini** ranks the `.agents` alias above `~/.gemini/skills` within the user tier, so a link in the "wrong" root is silently shadowed.
- **Claude Code** reads `SKILL.md` from the link target and loads the skill once even when several locations point at the same target — the only one that tolerates redundancy.

## Decision

1. **The toggleable unit is (Skill, Skills root).** The service API takes a root, not an agent, for enable/disable. Each column in the matrix is one root.
2. **Every column states its *Readers*.** The header shows the root and which agents read it, because that is what a click on that column actually affects.
3. **Adapters declare roots plus readers, not the reverse.** An adapter says: this agent reads these roots. The reader list per column is derived by inverting that map, so the many-to-many relationship exists in exactly one place.
4. **Double visibility is a first-class warning.** Enabling a skill in a second root that is read by an agent already seeing it is warned about before it happens, naming the agent and the roots. It is not blocked in general — Claude Code tolerates it, and some users will want the redundancy — but it is never silent.
5. **Roots carry a linkable flag.** Claude's account-synced store is scanned and shown but never written: the app does not modify what another tool manages. Unlinkable roots render as read-only columns.
6. **The `~/.agents` root is not privileged.** It is a root like any other, read by three agents. The app does not push every skill there.

## Consequences

- The matrix has fewer columns than agents (five roots vs four agents on this machine), and one column can serve three agents. Correct, and it removes the "why does Gemini show two of the same skill" confusion by explaining it in the header instead of hiding it.
- "Enable for Codex only" is only expressible when Codex's own root is linkable — true for `~/.codex/skills`. For agents without a private root (none today except shared-only setups), the user must accept serving the co-readers too.
- The reader map is a maintained fact. An agent that adds or drops a root changes what a column means, so the citations in the adapter definitions are load-bearing, not decoration.
- Double-visibility warnings will fire often on this machine, because `skill-creator` already lives in two roots read by overlapping agents. That is the actual complaint the tool exists to answer, not noise to be suppressed.

## Rejected alternatives

- **Column per agent, toggle writes the "preferred" root** — invents a preference model the agents do not have, and hides which physical link changed.
- **One canonical root for all skills (`~/.agents`)** — forces every skill onto the three agents that read it, breaking per-agent control, the feature being built.
- **Blocking all double visibility** — Claude Code deduplicates by target, so the rule would forbid states that work fine.
