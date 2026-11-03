## Agent skills

### Issue tracker

Issues live as GitHub issues in `Mirai3103/agent-skills-manager`, driven with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles mapped 1:1 to `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Engineering

### UI components

Add UI primitives with the CLI: `npx shadcn@latest add <component>`. It writes into `frontend/src/components/ui/` against this project's `base-lyra` preset and base-ui base, so the files stay diffable against the registry (`--dry-run`, `--diff` for updates). Hand-edit a registered component only when the generated output breaks a rule, and name the rule in the change.

### Dependencies

Install a library when it is the right tool, with the project's package manager, and keep going. Preference order: already installed, then standard library, then new dependency.

### Go domain code

The domain package takes its base directory as an argument; `$HOME` is resolved once at the app edge. That single seam is what makes the filesystem testable without mocks.

### Tests

Tests build their fixtures in `t.TempDir()` and assert on resulting filesystem state. The real agent homes under `$HOME` are never read or written by a test.

### Generated files

`frontend/bindings/**` is generated. After changing a Go service the frontend calls, run `task generate:bindings`.
