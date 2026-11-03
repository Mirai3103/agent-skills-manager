package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// LinkResult reports what one Enable or Disable call changed. Path is the link
// written or removed; it is empty when the call only produced warnings.
type LinkResult struct {
	Path     string   `json:"path"`
	Warnings []string `json:"warnings"`
}

// Enable links the store copy of name into root, the toggleable unit (ADR-0004).
// With confirm=false it returns the warnings and writes nothing; the same call
// with confirm=true performs the write. A read-only root is always refused.
func Enable(home, name, root string, confirm bool, adapters []Adapter) (LinkResult, error) {
	if err := checkName(name); err != nil {
		return LinkResult{}, err
	}

	inv := Scan(home, adapters)
	index, ok := rootIndex(inv, root)
	if !ok {
		return LinkResult{}, fmt.Errorf("unknown skills root %s", root)
	}
	label := inv.Roots[index].Label
	if !inv.Roots[index].Linkable {
		return LinkResult{}, fmt.Errorf("cannot enable %q: %s is a read-only root", name, label)
	}
	if inv.Roots[index].depth != 1 {
		return LinkResult{}, fmt.Errorf("cannot enable %q: %s keeps its skills below an account directory", name, label)
	}
	skill, ok := findSkill(inv, name)
	if !ok {
		return LinkResult{}, fmt.Errorf("no skill named %q", name)
	}
	switch cell := skill.Cells[index]; cell.State {
	case Enabled:
		return LinkResult{}, fmt.Errorf("%q is already enabled at %s", name, label)
	case Loose:
		return LinkResult{}, fmt.Errorf("cannot enable %q: a real copy sits at %s; adopt it first", name, cell.Path)
	case Orphan:
		return LinkResult{}, fmt.Errorf("cannot enable %q: an orphan link sits at %s; remove it first", name, cell.Path)
	}

	storeSkill := filepath.Join(StoreDir(home), name)
	if _, _, ok := readSkill(storeSkill); !ok {
		return LinkResult{}, fmt.Errorf("cannot enable %q: no store copy; adopt it first", name)
	}

	warnings := doubleVisibility(inv, index, skill)
	if len(warnings) > 0 && !confirm {
		return LinkResult{Warnings: warnings}, nil
	}

	link := filepath.Join(inv.Roots[index].Path, name)
	rel, err := filepath.Rel(filepath.Dir(link), storeSkill)
	if err != nil {
		return LinkResult{}, fmt.Errorf("enable %q: linking %s: %w", name, link, err)
	}
	if err := os.Symlink(rel, link); err != nil {
		return LinkResult{}, fmt.Errorf("enable %q: linking %s: %w", name, link, err)
	}
	return LinkResult{Path: link}, nil
}

// Disable removes the link at root and nothing else: the store copy stays, and
// every other root keeps its own state. Disabling the last enabled root for a
// skill warns first, like a double visibility warning.
func Disable(home, name, root string, confirm bool, adapters []Adapter) (LinkResult, error) {
	if err := checkName(name); err != nil {
		return LinkResult{}, err
	}

	inv := Scan(home, adapters)
	index, ok := rootIndex(inv, root)
	if !ok {
		return LinkResult{}, fmt.Errorf("unknown skills root %s", root)
	}
	label := inv.Roots[index].Label
	if !inv.Roots[index].Linkable {
		return LinkResult{}, fmt.Errorf("cannot disable %q: %s is a read-only root", name, label)
	}
	skill, ok := findSkill(inv, name)
	if !ok {
		return LinkResult{}, fmt.Errorf("no skill named %q", name)
	}
	cell := skill.Cells[index]
	if cell.State != Enabled {
		return LinkResult{}, fmt.Errorf("%q is not enabled at %s", name, label)
	}

	var warnings []string
	if countState(skill.Cells, Enabled) == 1 {
		// The skill may still sit loose in another root, so this only claims
		// what is true: the link goes away.
		warnings = append(warnings, fmt.Sprintf(
			"%s is the last root with %s enabled; disabling it removes the only link to the store copy", label, name))
	}
	if len(warnings) > 0 && !confirm {
		return LinkResult{Warnings: warnings}, nil
	}

	// os.Remove refuses directories, so a copy the scan no longer sees as a
	// link is never deleted by a stale click.
	if err := os.Remove(cell.Path); err != nil {
		return LinkResult{}, fmt.Errorf("disable %q: removing %s: %w", name, cell.Path, err)
	}
	return LinkResult{Path: cell.Path}, nil
}

// doubleVisibility names every agent that would read skill through two roots
// once the target root holds it (ADR-0004). The wording stays true for Claude
// Code, which deduplicates by link target, so no agent is special-cased.
func doubleVisibility(inv Inventory, target int, skill Skill) []string {
	var warnings []string
	for _, reader := range inv.Roots[target].Readers {
		for i, root := range inv.Roots {
			if i == target || !slices.Contains(root.Readers, reader) {
				continue
			}
			if state := skill.Cells[i].State; state != Enabled && state != Loose {
				continue
			}
			warnings = append(warnings, fmt.Sprintf(
				"Double visibility: %s already reads %s through %s; enabling it at %s gives that agent two roots",
				reader, skill.Name, root.Label, inv.Roots[target].Label))
		}
	}
	return warnings
}

func rootIndex(inv Inventory, root string) (int, bool) {
	for i, r := range inv.Roots {
		if filepath.Clean(r.Path) == filepath.Clean(root) {
			return i, true
		}
	}
	return 0, false
}

func countState(cells []Cell, state State) int {
	n := 0
	for _, cell := range cells {
		if cell.State == state {
			n++
		}
	}
	return n
}
