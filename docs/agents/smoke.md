# UI smoke

How to see the real matrix before claiming a UI change works. The app is a
Wails desktop app, so there is no dev server that runs alone: build the frontend,
build a server-mode binary that embeds it, then point the binary at a **sandbox
`$HOME`**. Never smoke against your real `$HOME` — adopt and enable write links
into it.

```sh
./scripts/check.sh                       # bindings, types, tests
(cd frontend && npm ci && npm run build) # dist is embedded when the Go binary builds
go build -tags server -o bin/agent-skills-manager-server .
```

Fixture home (depth-1 roots are `<home>/<root>/<skill>`):

```sh
HOME_DIR=$(mktemp -d)
mkdir -p "$HOME_DIR/.agents/skills/alpha" "$HOME_DIR/.claude/skills/alpha"
printf -- '---\nname: alpha\n---\n# Alpha\n' | tee "$HOME_DIR/"{.agents,.claude}/skills/alpha/SKILL.md
HOME="$HOME_DIR" ./bin/agent-skills-manager-server
```

Server mode listens on `http://localhost:8080`; the generated bindings talk to
it over the same origin, so a browser there is the real app, not a mock.

Verify on disk, not just on screen: `ls -l "$HOME_DIR"/.claude/skills`,
`readlink`, and `cat` inside `.agent-skills/store/`. A refusal path is easiest to
provoke with `chmod 0555 <root>` before the click.

Rebuild the frontend and the binary after every frontend change — the old
`dist` is baked into a server binary that already exists.
