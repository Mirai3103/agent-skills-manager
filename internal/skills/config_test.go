package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtraRootPersistsAndContributesColumn(t *testing.T) {
	home := fakeHome(t)
	extra := filepath.Join(home, "custom", "skills")
	writeSkill(t, extra, "alpha", alphaSkill)

	if err := AddExtraRoot(home, "~/custom/skills"); err != nil {
		t.Fatalf("AddExtraRoot: %v", err)
	}
	if _, err := os.Stat(ConfigPath(home)); err != nil {
		t.Fatalf("config was not written: %v", err)
	}

	// A second Scan reads the config from disk, which is what a restart does.
	inv := Scan(home, nil)

	if len(inv.Roots) != 1 {
		t.Fatalf("roots = %+v, want the registered root", inv.Roots)
	}
	root := inv.Roots[0]
	if !root.Extra || !root.Linkable {
		t.Errorf("root = %+v, want extra and linkable", root)
	}
	if root.Label != "~/custom/skills" {
		t.Errorf("label = %q, want ~/custom/skills", root.Label)
	}
	skill := onlySkill(t, inv)
	if skill.Name != "alpha" || skill.Cells[0].State != Loose {
		t.Errorf("skill = %+v, want alpha loose under the extra root", skill)
	}
}

func TestExtraRootMissingReportsErrorAndKeepsMatrix(t *testing.T) {
	home := fakeHome(t, "one/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	if err := AddExtraRoot(home, "~/gone/skills"); err != nil {
		t.Fatalf("AddExtraRoot: %v", err)
	}

	inv := Scan(home, adaptersFor("one/skills"))

	if len(inv.Roots) != 2 {
		t.Fatalf("roots = %+v, want the adapter root and the missing extra root", inv.Roots)
	}
	if len(inv.Errors) != 1 || inv.Errors[0].Root != "~/gone/skills" {
		t.Fatalf("errors = %+v, want one for ~/gone/skills", inv.Errors)
	}
	if inv.Roots[1].Linkable {
		t.Error("a missing extra root must not offer writes")
	}
	skill := onlySkill(t, inv)
	if skill.Cells[0].State != Loose || skill.Cells[1].State != Absent {
		t.Errorf("cells = %+v, want loose then absent", skill.Cells)
	}
}

func TestExtraRootUnreadableReportsErrorAndKeepsColumn(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	home := fakeHome(t)
	extra := filepath.Join(home, "custom", "skills")
	writeSkill(t, extra, "alpha", alphaSkill)
	if err := AddExtraRoot(home, extra); err != nil {
		t.Fatalf("AddExtraRoot: %v", err)
	}
	if err := os.Chmod(extra, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(extra, 0o755) })

	inv := Scan(home, nil)

	if len(inv.Roots) != 1 {
		t.Errorf("roots = %d, want the unreadable extra root to keep its column", len(inv.Roots))
	} else if inv.Roots[0].Linkable {
		t.Error("an unreadable extra root must not offer writes")
	}
	if len(inv.Errors) != 1 || inv.Errors[0].Root != "~/custom/skills" {
		t.Errorf("errors = %+v, want one for ~/custom/skills", inv.Errors)
	}
}

func TestExtraRootsSkipUnresolvableEntry(t *testing.T) {
	home := fakeHome(t)
	writeSkill(t, filepath.Join(home, "good", "skills"), "alpha", alphaSkill)
	writeFile(t, ConfigPath(home), `{"extraRoots": ["relative/skills", "~/good/skills"]}`)

	inv := Scan(home, nil)

	if len(inv.Roots) != 1 || inv.Roots[0].Label != "~/good/skills" {
		t.Errorf("roots = %+v, want only the resolvable entry", inv.Roots)
	}
	if len(inv.Errors) != 1 || inv.Errors[0].Root != "relative/skills" {
		t.Errorf("errors = %+v, want one for the bad entry", inv.Errors)
	}
}

func TestRemoveExtraRootDropsColumnAndLeavesDiskAlone(t *testing.T) {
	home := fakeHome(t)
	extra := filepath.Join(home, "custom", "skills")
	writeSkill(t, extra, "alpha", alphaSkill)
	if err := AddExtraRoot(home, extra); err != nil {
		t.Fatalf("AddExtraRoot: %v", err)
	}

	if err := RemoveExtraRoot(home, "~/custom/skills"); err != nil {
		t.Fatalf("RemoveExtraRoot: %v", err)
	}

	inv := Scan(home, nil)
	if len(inv.Roots) != 0 {
		t.Errorf("roots = %+v, want none after removal", inv.Roots)
	}
	if len(inv.Skills) != 0 {
		t.Errorf("skills = %+v, want none after removal", inv.Skills)
	}
	if _, err := os.Stat(filepath.Join(extra, "alpha", "SKILL.md")); err != nil {
		t.Errorf("removal touched disk: %v", err)
	}
	if err := RemoveExtraRoot(home, extra); err == nil {
		t.Error("removing an unregistered root should fail")
	}
}

func TestExtraRootSkillIsAdoptable(t *testing.T) {
	home := fakeHome(t)
	extra := filepath.Join(home, "custom", "skills")
	writeSkill(t, extra, "alpha", alphaSkill)
	if err := AddExtraRoot(home, "~/custom/skills"); err != nil {
		t.Fatalf("AddExtraRoot: %v", err)
	}

	if _, err := Adopt(home, "alpha", nil); err != nil {
		t.Fatalf("Adopt: %v", err)
	}

	mustRelativeLink(t, filepath.Join(extra, "alpha"))
	skill := onlySkill(t, Scan(home, nil))
	if !skill.Stored || skill.Cells[0].State != Enabled {
		t.Errorf("skill = %+v, want stored and enabled under the extra root", skill)
	}
}

func TestAddExtraRootRejectsRelativeAndDuplicatePaths(t *testing.T) {
	home := fakeHome(t)

	if err := AddExtraRoot(home, "skills"); err == nil {
		t.Error("relative path should be refused")
	}
	if err := AddExtraRoot(home, "~/skills"); err != nil {
		t.Fatalf("AddExtraRoot: %v", err)
	}
	if err := AddExtraRoot(home, filepath.Join(home, "skills")); err == nil {
		t.Error("the same root registered twice should be refused")
	}
	if err := AddExtraRoot(home, "~/.agent-skills/store"); err == nil {
		t.Error("the app's own directory must never become a skills root")
	}
}

func TestExtraRootOverlappingAnAdapterRootStaysOneColumn(t *testing.T) {
	home := fakeHome(t, "one/skills")
	if err := AddExtraRoot(home, "~/one/skills"); err != nil {
		t.Fatalf("AddExtraRoot: %v", err)
	}

	inv := Scan(home, adaptersFor("one/skills"))

	if len(inv.Roots) != 1 {
		t.Fatalf("roots = %+v, want one column", inv.Roots)
	}
	if !inv.Roots[0].Extra {
		t.Error("overlapping override should mark the column extra so it can be removed")
	}
}

func TestScanReportsBrokenConfigAndKeepsAdapterRoots(t *testing.T) {
	home := fakeHome(t, "one/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeFile(t, ConfigPath(home), "{not json")

	inv := Scan(home, adaptersFor("one/skills"))

	if len(inv.Roots) != 1 {
		t.Errorf("roots = %+v, want the adapter root to survive", inv.Roots)
	}
	if len(inv.Errors) != 1 || inv.Errors[0].Root != "~/.agent-skills/config.json" {
		t.Errorf("errors = %+v, want one for the config file", inv.Errors)
	}
	onlySkill(t, inv)
}
