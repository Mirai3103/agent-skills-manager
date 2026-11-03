package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scan is a read-only traversal of every skills root under home that exists on
// disk, plus the store. Missing roots contribute no column; an unreadable root
// reports a RootError and leaves the rest of the inventory intact. Store skills
// add rows but no column, so a store-only skill lists with every cell Absent.
func Scan(home string, adapters []Adapter) Inventory {
	inv := Inventory{
		Roots:  []Root{},
		Skills: []Skill{},
		Errors: []RootError{},
	}

	var found []entry
	for _, spec := range rootSpecs(home, adapters) {
		info, err := os.Stat(spec.Path)
		if err != nil {
			if !os.IsNotExist(err) {
				inv.Errors = append(inv.Errors, RootError{Root: spec.Label, Err: err.Error()})
			}
			continue
		}
		if !info.IsDir() {
			continue
		}

		index := len(inv.Roots)
		inv.Roots = append(inv.Roots, spec.Root)

		entries, err := scanRoot(spec)
		if err != nil {
			inv.Errors = append(inv.Errors, RootError{Root: spec.Label, Err: err.Error()})
		}
		for i := range entries {
			entries[i].root = index
		}
		found = append(found, entries...)
	}

	// The store is not a skills root, so it adds rows but no column: a skill
	// that lives only there shows up with every cell Absent.
	storeEntries, err := scanDir(StoreDir(home))
	if err != nil && !os.IsNotExist(err) {
		inv.Errors = append(inv.Errors, RootError{Root: "~/.agent-skills/store", Err: err.Error()})
	}
	for i := range storeEntries {
		storeEntries[i].stored = true
	}
	found = append(found, storeEntries...)

	inv.Skills = buildSkills(found, len(inv.Roots))
	return inv
}

// rootSpec is a Root plus the depth its skills sit at.
type rootSpec struct {
	Root
	depth int
}

// rootSpecs inverts the adapters' claims: one root per unique claim path, with
// every claiming agent as a Reader. This is the only reader map in the code.
func rootSpecs(home string, adapters []Adapter) []rootSpec {
	seen := make(map[string]int)
	var specs []rootSpec
	for _, adapter := range adapters {
		for _, claim := range adapter.Claims {
			if i, ok := seen[claim.Root]; ok {
				specs[i].Readers = append(specs[i].Readers, adapter.Agent)
				// A root the app may write must be linkable for every reader.
				specs[i].Linkable = specs[i].Linkable && claim.Linkable
				continue
			}
			seen[claim.Root] = len(specs)
			specs = append(specs, rootSpec{
				Root: Root{
					Path:     filepath.Join(home, filepath.FromSlash(claim.Root)),
					Label:    "~/" + filepath.ToSlash(claim.Root),
					Readers:  []string{adapter.Agent},
					Linkable: claim.Linkable,
				},
				depth: claim.Depth,
			})
		}
	}
	for i := range specs {
		sort.Strings(specs[i].Readers)
	}
	return specs
}

// entry is one skill discovered at one root or in the store, before rows are
// assembled.
type entry struct {
	root   int
	name   string
	state  State
	hash   string
	fmName string
	path   string
	// stored marks a skill found in the store rather than in a skills root.
	stored bool
}

func scanRoot(spec rootSpec) ([]entry, error) {
	if spec.depth < 2 {
		return scanDir(spec.Path)
	}
	// A depth-2 root holds an account directory between root and skill,
	// e.g. ~/.claude/skills/synced/<account>/<skill>.
	accounts, err := os.ReadDir(spec.Path)
	if err != nil {
		return nil, err
	}
	var out []entry
	for _, account := range accounts {
		dir := filepath.Join(spec.Path, account.Name())
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		entries, err := scanDir(dir)
		if err != nil {
			return out, err
		}
		out = append(out, entries...)
	}
	return out, nil
}

// scanDir lists the skill positions directly inside dir. A position is a skill
// only when its SKILL.md can be read; anything else is ignored, except a
// symlink whose target is gone, which is an Orphan link.
func scanDir(dir string) ([]entry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []entry
	for _, dirEntry := range dirEntries {
		name := dirEntry.Name()
		path := filepath.Join(dir, name)

		info, err := os.Lstat(path)
		if err != nil {
			out = append(out, entry{name: name, state: Orphan, path: path})
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if _, err := os.Stat(path); err != nil {
				// The link outlived its target.
				out = append(out, entry{name: name, state: Orphan, path: path})
				continue
			}
			hash, fmName, ok := readSkill(path)
			if !ok {
				continue
			}
			out = append(out, entry{name: name, state: Enabled, hash: hash, fmName: fmName, path: path})
			continue
		}

		if !info.IsDir() {
			continue
		}
		hash, fmName, ok := readSkill(path)
		if !ok {
			continue
		}
		out = append(out, entry{name: name, state: Loose, hash: hash, fmName: fmName, path: path})
	}
	return out, nil
}

func readSkill(dir string) (hash string, fmName string, ok bool) {
	content, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return "", "", false
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), frontmatterName(content), true
}

// frontmatterName pulls `name:` out of the YAML frontmatter. One key is not
// worth a YAML dependency.
func frontmatterName(content []byte) string {
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		line = strings.TrimSuffix(line, "\r")
		if strings.TrimSpace(line) == "---" {
			return ""
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "name" {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ""
}

// buildSkills collapses entries into one row per skill name, dense over roots.
func buildSkills(entries []entry, rootCount int) []Skill {
	type row struct {
		cells     []Cell
		hashes    map[string]struct{}
		rootHash  map[int]string // hash of the copy in each root; presence means the root has a cell
		warning   string
		stored    bool
		storeHash string
	}
	rows := make(map[string]*row)
	names := make([]string, 0, len(entries))

	for _, e := range entries {
		r, ok := rows[e.name]
		if !ok {
			r = &row{
				cells:    make([]Cell, rootCount),
				hashes:   make(map[string]struct{}),
				rootHash: make(map[int]string),
			}
			for i := range r.cells {
				r.cells[i] = Cell{State: Absent}
			}
			rows[e.name] = r
			names = append(names, e.name)
		}
		if r.warning == "" && e.fmName != "" && e.fmName != e.name {
			r.warning = `frontmatter name "` + e.fmName + `" differs from folder name "` + e.name + `"`
		}
		// The store is not a root: it is the canonical copy, so a root that
		// differs from it is a conflict, never a Variant.
		if e.stored {
			r.stored = true
			r.storeHash = e.hash
			continue
		}
		if e.hash != "" {
			r.hashes[e.hash] = struct{}{}
		}
		// shortcut: one cell per root, so a name held twice under the depth-2
		// root shows its first copy; split cells per account when a real root
		// needs both.
		if _, taken := r.rootHash[e.root]; taken {
			continue
		}
		r.rootHash[e.root] = e.hash
		r.cells[e.root] = Cell{State: e.state, Path: e.path}
	}

	sort.Strings(names)
	skills := make([]Skill, 0, len(names))
	for _, name := range names {
		r := rows[name]
		skill := Skill{Name: name, Status: StatusSingle, Stored: r.stored, Warning: r.warning, Cells: r.cells}

		conflict := false
		if r.stored {
			for i := range skill.Cells {
				if hash := r.rootHash[i]; hash != "" && hash != r.storeHash {
					skill.Cells[i].Conflict = true
					conflict = true
				}
			}
		}

		switch {
		case r.stored:
			// The store copy is the row's identity; a conflicting root keeps
			// its own hash on its cell only.
			skill.Hash = r.storeHash
			if conflict && skill.Warning == "" {
				skill.Warning = "the store copy differs from the skills roots; resolve the copies by hand"
			}
		case len(r.hashes) == 1:
			for hash := range r.hashes {
				skill.Hash = hash
			}
		}

		switch {
		case hasState(r.cells, Orphan):
			skill.Status = StatusOrphan
		case conflict:
			skill.Status = StatusConflict
		case len(r.hashes) > 1:
			skill.Status = StatusVariant
		case len(r.rootHash) > 1:
			skill.Status = StatusDuplicate
		}
		skills = append(skills, skill)
	}
	return skills
}

func hasState(cells []Cell, state State) bool {
	for _, c := range cells {
		if c.State == state {
			return true
		}
	}
	return false
}
