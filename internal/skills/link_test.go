package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sharedAdapters returns one agent that reads every given root, so double
// visibility is expressible without depending on the real adapter table.
func sharedAdapters(roots ...string) []Adapter {
	claims := make([]Claim, 0, len(roots))
	for _, root := range roots {
		claims = append(claims, Claim{Root: root, Depth: 1, Linkable: true, Source: "test"})
	}
	return []Adapter{{Agent: "OpenCode", Claims: claims}}
}

func TestEnableLinksTheStoreCopyIntoOneRoot(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, StoreDir(home), "alpha", alphaSkill)
	storeSkill := filepath.Join(StoreDir(home), "alpha")
	one := filepath.Join(home, "one/skills")
	adapters := adaptersFor("one/skills", "two/skills")

	result, err := Enable(home, "alpha", one, false, adapters)
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	link := filepath.Join(one, "alpha")
	if result.Path != link {
		t.Errorf("path = %q, want %q", result.Path, link)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", result.Warnings)
	}
	mustRelativeLink(t, link)
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != storeSkill {
		t.Errorf("%s resolves to %s, want %s", link, resolved, storeSkill)
	}
	if content, err := os.ReadFile(filepath.Join(storeSkill, "SKILL.md")); err != nil || string(content) != alphaSkill {
		t.Errorf("store SKILL.md = %q, %v; want the store entry untouched", content, err)
	}
	if _, err := os.Lstat(filepath.Join(home, "two/skills", "alpha")); !os.IsNotExist(err) {
		t.Errorf("enabling one root wrote into another: %v", err)
	}

	skill := onlySkill(t, Scan(home, adapters))
	if !skill.Stored {
		t.Error("enabled skill is not marked stored")
	}
	if skill.Cells[0].State != Enabled || skill.Cells[1].State != Absent {
		t.Errorf("cells = %+v, want enabled at one and absent at two", skill.Cells)
	}
}

func TestDisableRemovesOnlyTheLink(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, StoreDir(home), "alpha", alphaSkill)
	one := filepath.Join(home, "one/skills")
	two := filepath.Join(home, "two/skills")
	adapters := adaptersFor("one/skills", "two/skills")
	for _, root := range []string{one, two} {
		if _, err := Enable(home, "alpha", root, false, adapters); err != nil {
			t.Fatalf("Enable at %s: %v", root, err)
		}
	}

	result, err := Disable(home, "alpha", one, false, adapters)
	if err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if result.Path != filepath.Join(one, "alpha") {
		t.Errorf("path = %q, want the removed link", result.Path)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", result.Warnings)
	}
	if _, err := os.Lstat(filepath.Join(one, "alpha")); !os.IsNotExist(err) {
		t.Errorf("link survived Disable: %v", err)
	}
	mustRelativeLink(t, filepath.Join(two, "alpha"))
	if content, err := os.ReadFile(filepath.Join(StoreDir(home), "alpha", "SKILL.md")); err != nil || string(content) != alphaSkill {
		t.Errorf("store SKILL.md = %q, %v; want the store entry untouched", content, err)
	}

	skill := onlySkill(t, Scan(home, adapters))
	if !skill.Stored {
		t.Error("skill left the store listing")
	}
	if skill.Cells[0].State != Absent || skill.Cells[1].State != Enabled {
		t.Errorf("cells = %+v, want absent at one and enabled at two", skill.Cells)
	}
}

func TestDisableLastEnabledRootWarnsBeforeRemoving(t *testing.T) {
	home := fakeHome(t, "one/skills")
	writeSkill(t, StoreDir(home), "alpha", alphaSkill)
	one := filepath.Join(home, "one/skills")
	adapters := adaptersFor("one/skills")
	if _, err := Enable(home, "alpha", one, false, adapters); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	link := filepath.Join(one, "alpha")

	result, err := Disable(home, "alpha", one, false, adapters)
	if err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if result.Path != "" {
		t.Errorf("path = %q, want nothing removed before the warning is confirmed", result.Path)
	}
	warning := strings.Join(result.Warnings, "\n")
	if !strings.Contains(warning, "~/one/skills") || !strings.Contains(warning, "alpha") {
		t.Errorf("warnings = %q, want the last root and the skill named", warning)
	}
	mustRelativeLink(t, link)

	if _, err := Disable(home, "alpha", one, true, adapters); err != nil {
		t.Fatalf("confirmed Disable: %v", err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("link survived the confirmed Disable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(StoreDir(home), "alpha", "SKILL.md")); err != nil {
		t.Errorf("store entry gone after Disable: %v", err)
	}
}

func TestEnableAndDisableRefuseAReadOnlyRoot(t *testing.T) {
	home := fakeHome(t, "synced/acct")
	writeSkill(t, StoreDir(home), "alpha", alphaSkill)
	root := filepath.Join(home, "synced/acct")
	adapters := []Adapter{
		{Agent: "A", Claims: []Claim{{Root: "synced/acct", Depth: 1, Linkable: false, Source: "test"}}},
	}

	if _, err := Enable(home, "alpha", root, false, adapters); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("Enable error = %v, want the read-only root named", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "alpha")); !os.IsNotExist(err) {
		t.Errorf("Enable wrote into a read-only root: %v", err)
	}

	// A link another tool put there is that tool's to remove.
	link := filepath.Join(root, "alpha")
	rel, err := filepath.Rel(filepath.Dir(link), filepath.Join(StoreDir(home), "alpha"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(rel, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Disable(home, "alpha", root, false, adapters); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("Disable error = %v, want the read-only root named", err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("Disable removed a link from a read-only root: %v", err)
	}
}

func TestEnableRefusesARootThatNestsItsSkills(t *testing.T) {
	home := fakeHome(t, "synced/acct")
	writeSkill(t, StoreDir(home), "alpha", alphaSkill)
	adapters := []Adapter{
		{Agent: "A", Claims: []Claim{{Root: "synced/acct", Depth: 2, Linkable: true, Source: "test"}}},
	}

	_, err := Enable(home, "alpha", filepath.Join(home, "synced/acct"), false, adapters)
	if err == nil {
		t.Fatal("Enable accepted a root whose skills sit below an account directory")
	}
	if _, err := os.Lstat(filepath.Join(home, "synced/acct", "alpha")); !os.IsNotExist(err) {
		t.Errorf("Enable wrote at the root instead of refusing: %v", err)
	}
}

func TestEnableWarnsDoubleVisibilityBeforeWriting(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, StoreDir(home), "alpha", alphaSkill)
	one := filepath.Join(home, "one/skills")
	two := filepath.Join(home, "two/skills")
	adapters := sharedAdapters("one/skills", "two/skills")
	if _, err := Enable(home, "alpha", one, false, adapters); err != nil {
		t.Fatalf("Enable at one: %v", err)
	}

	result, err := Enable(home, "alpha", two, false, adapters)
	if err != nil {
		t.Fatalf("Enable at two: %v", err)
	}
	if result.Path != "" {
		t.Errorf("path = %q, want nothing written before the warning is confirmed", result.Path)
	}
	if _, err := os.Lstat(filepath.Join(two, "alpha")); !os.IsNotExist(err) {
		t.Errorf("Enable wrote before the warning was confirmed: %v", err)
	}
	warning := strings.Join(result.Warnings, "\n")
	for _, want := range []string{"OpenCode", "~/one/skills", "~/two/skills"} {
		if !strings.Contains(warning, want) {
			t.Errorf("warnings = %q, want %q", warning, want)
		}
	}

	if _, err := Enable(home, "alpha", two, true, adapters); err != nil {
		t.Fatalf("confirmed Enable: %v", err)
	}
	mustRelativeLink(t, filepath.Join(two, "alpha"))

	skill := onlySkill(t, Scan(home, adapters))
	if skill.Status != StatusDuplicate {
		t.Errorf("status = %q, want the double visibility visible on the row", skill.Status)
	}
}
