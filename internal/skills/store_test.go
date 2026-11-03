package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mustRelativeLink fails unless path is a symlink with a relative target.
func mustRelativeLink(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", path)
	}
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.IsAbs(target) {
		t.Fatalf("link %s has absolute target %q, want relative", path, target)
	}
}

func TestAdoptCopiesSkillIntoStoreAndLinksOriginals(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	for _, root := range []string{"one/skills", "two/skills"} {
		dir := filepath.Join(home, root)
		writeSkill(t, dir, "alpha", alphaSkill)
		writeFile(t, filepath.Join(dir, "alpha", "assets", "note.txt"), "asset\n")
	}

	result, err := Adopt(home, "alpha", adaptersFor("one/skills", "two/skills"))
	if err != nil {
		t.Fatalf("Adopt: %v", err)
	}

	storeSkill := filepath.Join(StoreDir(home), "alpha")
	if result.StorePath != storeSkill {
		t.Errorf("store path = %q, want %q", result.StorePath, storeSkill)
	}
	if len(result.Linked) != 2 {
		t.Errorf("linked = %v, want both roots", result.Linked)
	}
	if content, err := os.ReadFile(filepath.Join(storeSkill, "SKILL.md")); err != nil || string(content) != alphaSkill {
		t.Errorf("store SKILL.md = %q, %v; want the source bytes", content, err)
	}
	if content, err := os.ReadFile(filepath.Join(storeSkill, "assets", "note.txt")); err != nil || string(content) != "asset\n" {
		t.Errorf("store asset = %q, %v; want the source bytes", content, err)
	}

	for _, root := range []string{"one/skills", "two/skills"} {
		link := filepath.Join(home, root, "alpha")
		mustRelativeLink(t, link)
		resolved, err := filepath.EvalSymlinks(link)
		if err != nil {
			t.Fatal(err)
		}
		if resolved != storeSkill {
			t.Errorf("%s resolves to %s, want %s", link, resolved, storeSkill)
		}
	}

	skill := onlySkill(t, Scan(home, adaptersFor("one/skills", "two/skills")))
	if !skill.Stored {
		t.Error("adopted skill is not marked stored")
	}
	for i, cell := range skill.Cells {
		if cell.State != Enabled {
			t.Errorf("cell %d state = %q, want %q", i, cell.State, Enabled)
		}
	}
}

func TestAdoptLinksToExistingIdenticalStoreCopy(t *testing.T) {
	home := fakeHome(t, "one/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, StoreDir(home), "alpha", alphaSkill)

	if _, err := Adopt(home, "alpha", adaptersFor("one/skills")); err != nil {
		t.Fatalf("Adopt: %v", err)
	}
	mustRelativeLink(t, filepath.Join(home, "one/skills", "alpha"))
}

func TestAdoptRefusesVariantAndWritesNothing(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, filepath.Join(home, "two/skills"), "alpha", alphaSkill+"\nmore")

	if _, err := Adopt(home, "alpha", adaptersFor("one/skills", "two/skills")); err == nil {
		t.Fatal("Adopt succeeded on a variant")
	}
	if _, err := os.Stat(filepath.Join(StoreDir(home), "alpha")); !os.IsNotExist(err) {
		t.Errorf("store entry exists after refused adopt: %v", err)
	}
	for _, root := range []string{"one/skills", "two/skills"} {
		path := filepath.Join(home, root, "alpha")
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			t.Errorf("%s is no longer a real directory", path)
		}
	}
}

func TestAdoptRefusesUnlinkableRoot(t *testing.T) {
	home := fakeHome(t, "synced/acct", "one/skills")
	writeSkill(t, filepath.Join(home, "synced/acct"), "alpha", alphaSkill)
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	adapters := []Adapter{
		{Agent: "A", Claims: []Claim{{Root: "synced/acct", Depth: 1, Linkable: false, Source: "test"}}},
		{Agent: "B", Claims: []Claim{{Root: "one/skills", Depth: 1, Linkable: true, Source: "test"}}},
	}

	_, err := Adopt(home, "alpha", adapters)
	if err == nil {
		t.Fatal("Adopt succeeded with a loose copy in a read-only root")
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("error = %v, want the read-only root named", err)
	}
	if _, err := os.Stat(filepath.Join(StoreDir(home), "alpha")); !os.IsNotExist(err) {
		t.Errorf("store entry exists after refused adopt: %v", err)
	}
}

func TestAdoptFailureLeavesOriginalsUntouched(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based failure injection does not apply to root")
	}
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, filepath.Join(home, "two/skills"), "alpha", alphaSkill)
	readOnly := filepath.Join(home, "two/skills")
	if err := os.Chmod(readOnly, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o755) })

	_, err := Adopt(home, "alpha", adaptersFor("one/skills", "two/skills"))
	if err == nil {
		t.Fatal("Adopt succeeded with an unwritable root")
	}
	if !strings.Contains(err.Error(), "two/skills") {
		t.Errorf("error = %v, want the failing path named", err)
	}

	for _, root := range []string{"one/skills", "two/skills"} {
		path := filepath.Join(home, root, "alpha")
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if !info.IsDir() {
			t.Errorf("%s is not a real directory after rollback", path)
		}
		content, err := os.ReadFile(filepath.Join(path, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != alphaSkill {
			t.Errorf("%s content changed: %q", path, content)
		}
	}
	if _, err := os.Stat(filepath.Join(StoreDir(home), "alpha")); !os.IsNotExist(err) {
		t.Errorf("store entry survived rollback: %v", err)
	}
}

func TestAdoptRefusesDivergentAssetsAndWritesNothing(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, filepath.Join(home, "two/skills"), "alpha", alphaSkill)
	writeFile(t, filepath.Join(home, "one/skills", "alpha", "note.txt"), "one\n")
	writeFile(t, filepath.Join(home, "two/skills", "alpha", "note.txt"), "two\n")

	_, err := Adopt(home, "alpha", adaptersFor("one/skills", "two/skills"))
	if err == nil {
		t.Fatal("Adopt succeeded although the copies differ outside SKILL.md")
	}
	if !strings.Contains(err.Error(), "note.txt") {
		t.Errorf("error = %v, want the differing file named", err)
	}
	if _, err := os.Stat(filepath.Join(StoreDir(home), "alpha")); !os.IsNotExist(err) {
		t.Errorf("store entry exists after refused adopt: %v", err)
	}
	for root, want := range map[string]string{"one/skills": "one\n", "two/skills": "two\n"} {
		path := filepath.Join(home, root, "alpha")
		if info, err := os.Lstat(path); err != nil || !info.IsDir() {
			t.Fatalf("%s is not a real directory after refusal: %v", path, err)
		}
		if content, err := os.ReadFile(filepath.Join(path, "note.txt")); err != nil || string(content) != want {
			t.Errorf("%s content = %q, %v; want %q", path, content, err, want)
		}
	}
}

func TestAdoptRefusesStoreCopyThatDiffersFromRoots(t *testing.T) {
	home := fakeHome(t, "one/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, StoreDir(home), "alpha", alphaSkill+"\nmore")

	skill := onlySkill(t, Scan(home, adaptersFor("one/skills")))
	if skill.Status == StatusVariant {
		t.Errorf("status = %q: the store is not a root", skill.Status)
	}
	if !skill.Stored || skill.Hash == "" {
		t.Errorf("skill = %+v, want stored with the root hash", skill)
	}
	if !strings.Contains(skill.Warning, "store copy differs") {
		t.Errorf("warning = %q, want the store conflict surfaced", skill.Warning)
	}

	if _, err := Adopt(home, "alpha", adaptersFor("one/skills")); err == nil {
		t.Fatal("Adopt succeeded although the store copy differs")
	}
	if info, err := os.Lstat(filepath.Join(home, "one/skills", "alpha")); err != nil || !info.IsDir() {
		t.Errorf("loose original is gone after refused adopt: %v", err)
	}
}

func TestScanListsStoreOnlySkillWithEveryCellAbsent(t *testing.T) {
	home := fakeHome(t, "one/skills")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, StoreDir(home), "beta", betaSkill)

	inv := Scan(home, adaptersFor("one/skills"))
	if len(inv.Roots) != 1 {
		t.Fatalf("roots = %d, want 1: the store is not a column", len(inv.Roots))
	}
	if len(inv.Skills) != 2 {
		t.Fatalf("skills = %+v, want alpha and beta", inv.Skills)
	}

	var beta *Skill
	for i := range inv.Skills {
		if inv.Skills[i].Name == "beta" {
			beta = &inv.Skills[i]
		}
	}
	if beta == nil {
		t.Fatal("store-only skill missing from the inventory")
	}
	if !beta.Stored {
		t.Error("store-only skill is not marked stored")
	}
	if beta.Hash == "" {
		t.Error("store-only skill has no hash")
	}
	for i, cell := range beta.Cells {
		if cell.State != Absent {
			t.Errorf("cell %d state = %q, want %q", i, cell.State, Absent)
		}
	}
}
