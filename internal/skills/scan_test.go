package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	alphaSkill = "---\nname: alpha\ndescription: test\n---\n# Alpha\n"
	betaSkill  = "---\nname: beta\n---\n# Beta\n"
)

// fakeHome builds a home directory whose skills roots are the given relative
// paths, so tests never touch the real home.
func fakeHome(t *testing.T, roots ...string) string {
	t.Helper()
	home := t.TempDir()
	for _, root := range roots {
		if err := os.MkdirAll(filepath.Join(home, root), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func writeSkill(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func adaptersFor(roots ...string) []Adapter {
	adapters := make([]Adapter, 0, len(roots))
	for i, root := range roots {
		adapters = append(adapters, Adapter{
			Agent:  string(rune('A' + i)),
			Claims: []Claim{{Root: root, Depth: 1, Linkable: true, Source: "test"}},
		})
	}
	return adapters
}

func onlySkill(t *testing.T, inv Inventory) Skill {
	t.Helper()
	if len(inv.Skills) != 1 {
		t.Fatalf("skills = %d, want 1: %+v", len(inv.Skills), inv.Skills)
	}
	return inv.Skills[0]
}

func TestScanDuplicateCollapsesIntoOneRow(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, filepath.Join(home, "two/skills"), "alpha", alphaSkill)

	inv := Scan(home, adaptersFor("one/skills", "two/skills"))

	if len(inv.Roots) != 2 {
		t.Fatalf("roots = %d, want 2", len(inv.Roots))
	}
	skill := onlySkill(t, inv)
	if skill.Status != StatusDuplicate {
		t.Errorf("status = %q, want %q", skill.Status, StatusDuplicate)
	}
	if skill.Hash == "" {
		t.Error("hash is empty")
	}
	for i, cell := range skill.Cells {
		if cell.State != Loose {
			t.Errorf("cell %d state = %q, want %q", i, cell.State, Loose)
		}
	}
}

func TestScanSameNameDifferentHashIsVariant(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, filepath.Join(home, "two/skills"), "alpha", alphaSkill+"\nmore")

	skill := onlySkill(t, Scan(home, adaptersFor("one/skills", "two/skills")))

	if skill.Status != StatusVariant {
		t.Errorf("status = %q, want %q", skill.Status, StatusVariant)
	}
	if skill.Hash != "" {
		t.Errorf("hash = %q, want empty for a variant", skill.Hash)
	}
}

func TestScanRelativeSymlinkIsEnabled(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	target := filepath.Join(home, "one/skills", "alpha")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	link := filepath.Join(home, "two/skills", "alpha")
	rel, err := filepath.Rel(filepath.Dir(link), target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(rel, link); err != nil {
		t.Fatal(err)
	}

	inv := Scan(home, adaptersFor("one/skills", "two/skills"))
	skill := onlySkill(t, inv)

	if skill.Status != StatusDuplicate {
		t.Errorf("status = %q, want %q", skill.Status, StatusDuplicate)
	}
	if skill.Cells[0].State != Loose || skill.Cells[1].State != Enabled {
		t.Errorf("cells = %+v, want loose then enabled", skill.Cells)
	}
}

func TestScanBrokenSymlinkIsOrphan(t *testing.T) {
	home := fakeHome(t, "one/skills")
	link := filepath.Join(home, "one/skills", "alpha")
	if err := os.Symlink("../../nowhere/alpha", link); err != nil {
		t.Fatal(err)
	}

	skill := onlySkill(t, Scan(home, adaptersFor("one/skills")))

	if skill.Status != StatusOrphan {
		t.Errorf("status = %q, want %q", skill.Status, StatusOrphan)
	}
	if skill.Cells[0].State != Orphan {
		t.Errorf("cell state = %q, want %q", skill.Cells[0].State, Orphan)
	}
}

func TestScanIgnoresDirectoriesWithoutSkillMD(t *testing.T) {
	home := fakeHome(t, "one/skills")
	root := filepath.Join(home, "one/skills")
	if err := os.MkdirAll(filepath.Join(root, "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	inv := Scan(home, adaptersFor("one/skills"))

	if len(inv.Skills) != 0 {
		t.Errorf("skills = %+v, want none", inv.Skills)
	}
}

func TestScanUnreadableRootReportsErrorAndKeepsOthers(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, filepath.Join(home, "two/skills"), "beta", betaSkill)
	locked := filepath.Join(home, "one/skills")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	inv := Scan(home, adaptersFor("one/skills", "two/skills"))

	if len(inv.Errors) != 1 || inv.Errors[0].Root != "~/one/skills" {
		t.Fatalf("errors = %+v, want one for ~/one/skills", inv.Errors)
	}
	skill := onlySkill(t, inv)
	if skill.Name != "beta" {
		t.Errorf("skill = %q, want beta from the readable root", skill.Name)
	}
	if len(inv.Roots) != 2 {
		t.Errorf("roots = %d, want both to contribute a column", len(inv.Roots))
	}
}

func TestScanDerivesReadersAndSkipsMissingRoots(t *testing.T) {
	home := fakeHome(t, "shared/skills")
	writeSkill(t, filepath.Join(home, "shared/skills"), "alpha", alphaSkill)

	inv := Scan(home, []Adapter{
		{Agent: "Zed", Claims: []Claim{{Root: "shared/skills", Depth: 1, Linkable: true, Source: "test"}}},
		{Agent: "Ada", Claims: []Claim{
			{Root: "shared/skills", Depth: 1, Linkable: true, Source: "test"},
			{Root: "missing/skills", Depth: 1, Linkable: true, Source: "test"},
		}},
	})

	if len(inv.Roots) != 1 {
		t.Fatalf("roots = %d, want only the root that exists: %+v", len(inv.Roots), inv.Roots)
	}
	root := inv.Roots[0]
	if got := strings.Join(root.Readers, ","); got != "Ada,Zed" {
		t.Errorf("readers = %q, want Ada,Zed", got)
	}
	if root.Label != "~/shared/skills" {
		t.Errorf("label = %q", root.Label)
	}
	if !root.Linkable {
		t.Error("root should be linkable")
	}
}

func TestScanReadOnlyRoot(t *testing.T) {
	home := fakeHome(t, "synced")
	account := filepath.Join(home, "synced", "acct")
	writeSkill(t, account, "beta", betaSkill)
	if err := os.WriteFile(filepath.Join(account, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	inv := Scan(home, []Adapter{{
		Agent:  "Claude Code",
		Claims: []Claim{{Root: "synced", Depth: 2, Linkable: false, Source: "test"}},
	}})

	if len(inv.Roots) != 1 || inv.Roots[0].Linkable {
		t.Fatalf("roots = %+v, want one read-only root", inv.Roots)
	}
	skill := onlySkill(t, inv)
	if skill.Name != "beta" || skill.Cells[0].State != Loose {
		t.Errorf("skill = %+v, want the depth-2 loose copy", skill)
	}
}

func TestScanWarnsWhenFrontmatterNameDiffers(t *testing.T) {
	home := fakeHome(t, "one/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "folder-name", "---\nname: other-name\n---\n")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)

	inv := Scan(home, adaptersFor("one/skills"))

	if len(inv.Skills) != 2 {
		t.Fatalf("skills = %d, want 2", len(inv.Skills))
	}
	for _, skill := range inv.Skills {
		warning := skill.Warning != ""
		if skill.Name == "folder-name" && !warning {
			t.Error("folder-name row should carry a warning")
		}
		if skill.Name == "alpha" && warning {
			t.Errorf("alpha row warned: %q", skill.Warning)
		}
	}
}
