package skills

// Claim is one agent's statement that it reads a skills root. The per-column
// reader list is derived by inverting these claims — see readers in Scan.
type Claim struct {
	// Root is a path relative to the user's home.
	Root string
	// Depth is how far below Root a skill directory sits: 1 for
	// <root>/<skill>, 2 for <root>/<owner>/<skill> (Claude's account-synced store).
	Depth int
	// Linkable reports whether the app may write links into this root.
	Linkable bool
	// Source is where the claim comes from. It is load-bearing: an agent that
	// changes its roots changes what a column means.
	Source string
}

// Adapter is one agent and the roots it reads.
type Adapter struct {
	Agent  string
	Claims []Claim
}

// DefaultAdapters returns the verified root claims for the agents on a Linux box.
// Evidence: local://agent-skill-discovery.md (2026-11-03).
func DefaultAdapters() []Adapter {
	const (
		claudeDocs = "https://code.claude.com/docs/en/skills"
		codexSrc   = "https://github.com/openai/codex/blob/main/codex-rs/ext/skills/src/host_roots.rs"
		geminiDocs = "https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/skills.md"
		opencode   = "https://opencode.ai/docs/skills"
		agentsSpec = "https://agentskills.io/specification"
	)
	return []Adapter{
		{Agent: "Claude Code", Claims: []Claim{
			{Root: ".claude/skills", Depth: 1, Linkable: true, Source: claudeDocs},
			// Account-synced store: 19 skills behind a per-account directory, and
			// another tool owns the content, so the app never writes here.
			{Root: ".claude/skills/synced", Depth: 2, Linkable: false, Source: claudeDocs},
		}},
		{Agent: "Codex CLI", Claims: []Claim{
			{Root: ".codex/skills", Depth: 1, Linkable: true, Source: codexSrc},
			{Root: ".agents/skills", Depth: 1, Linkable: true, Source: codexSrc},
		}},
		{Agent: "Gemini CLI", Claims: []Claim{
			{Root: ".gemini/skills", Depth: 1, Linkable: true, Source: geminiDocs},
			// The .agents alias outranks ~/.gemini/skills within the user tier.
			{Root: ".agents/skills", Depth: 1, Linkable: true, Source: geminiDocs + "#precedence-and-aliases"},
		}},
		{Agent: "OpenCode", Claims: []Claim{
			{Root: ".config/opencode/skills", Depth: 1, Linkable: true, Source: opencode},
			// OpenCode aggregates other tools' roots by default.
			{Root: ".claude/skills", Depth: 1, Linkable: true, Source: opencode},
			{Root: ".agents/skills", Depth: 1, Linkable: true, Source: opencode + " (external skills)"},
		}},
		{Agent: "Agent Skills spec", Claims: []Claim{
			{Root: ".agents/skills", Depth: 1, Linkable: true, Source: agentsSpec},
		}},
	}
}
