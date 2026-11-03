package skills

// State is what one root holds for a skill.
type State string

const (
	// Enabled means a link exists at the root.
	Enabled State = "enabled"
	// Loose means a real directory exists at the root.
	Loose State = "loose"
	// Absent means the root does not have the skill.
	Absent State = "absent"
	// Orphan means a link exists at the root but its target is gone.
	Orphan State = "orphan"
)

// Status summarises a row across roots.
type Status string

const (
	StatusSingle    Status = ""
	StatusDuplicate Status = "duplicate" // identical hash in more than one root
	StatusVariant   Status = "variant"   // same name, different hashes
	StatusOrphan    Status = "orphan"    // a link whose target is gone
	// StatusConflict means a column still holds a copy that differs from the
	// store copy: the user has to deal with that path themselves.
	StatusConflict Status = "conflict"
)

// Root is a skills root: the toggleable unit, not the agent (ADR-0004).
type Root struct {
	Path     string   `json:"path"`
	Label    string   `json:"label"`
	Readers  []string `json:"readers"`
	Linkable bool     `json:"linkable"`
	// depth is how far below Path a skill directory sits: 1 for
	// <root>/<skill>, 2 for <root>/<account>/<skill>. Unexported because the
	// UI never needs it; a write does, to refuse a layout it cannot address.
	depth int
}

// Cell is one skill's state at one root.
type Cell struct {
	Root  string `json:"root"`
	State State  `json:"state"`
	Path  string `json:"path"`
	// Conflict marks a copy whose SKILL.md differs from the store copy, so the
	// UI can show the path the user has to deal with by hand.
	Conflict bool `json:"conflict"`
}

// Skill is one row of the matrix.
type Skill struct {
	Name   string `json:"name"`
	Hash   string `json:"hash"`
	Status Status `json:"status"`
	// Stored reports whether a copy lives in the Skill store. A store-only
	// skill keeps its row with every cell Absent: installed, disabled everywhere.
	Stored  bool   `json:"stored"`
	Warning string `json:"warning"`
	Cells   []Cell `json:"cells"`
}

// RootError reports one unreadable root (or the store) without failing the
// whole scan.
type RootError struct {
	Root string `json:"root"`
	Err  string `json:"err"`
}

// Inventory is the whole read-only picture of the machine.
type Inventory struct {
	Roots  []Root      `json:"roots"`
	Skills []Skill     `json:"skills"`
	Errors []RootError `json:"errors"`
}
