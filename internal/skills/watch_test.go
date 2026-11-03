package skills

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchDirsCoversRootsSkillDirsAndStore(t *testing.T) {
	home := fakeHome(t, "one/skills", "two/skills/account")
	writeSkill(t, filepath.Join(home, "one/skills"), "alpha", alphaSkill)
	writeSkill(t, filepath.Join(home, "one/skills/nested/acct"), "beta", betaSkill)
	writeSkill(t, filepath.Join(home, "two/skills/account"), "beta", betaSkill)
	writeSkill(t, StoreDir(home), "gamma", alphaSkill)

	adapters := []Adapter{
		{Agent: "A", Claims: []Claim{{Root: "one/skills", Depth: 1, Linkable: true, Source: "test"}}},
		// Depth-2 roots keep skills one directory further down, and one of them
		// sits inside a depth-1 root: it must still be descended.
		{Agent: "B", Claims: []Claim{{Root: "one/skills/nested", Depth: 2, Linkable: true, Source: "test"}}},
		{Agent: "C", Claims: []Claim{{Root: "two/skills", Depth: 2, Linkable: true, Source: "test"}}},
		{Agent: "D", Claims: []Claim{{Root: "missing/skills", Depth: 1, Linkable: true, Source: "test"}}},
	}
	dirs := WatchDirs(home, adapters)
	got := make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		got[dir] = true
	}

	want := []string{
		filepath.Join(home, "one/skills"),
		filepath.Join(home, "one/skills/alpha"),
		filepath.Join(home, "one/skills/nested"),
		filepath.Join(home, "one/skills/nested/acct"),      // seen by the depth-2 root inside a shallow one
		filepath.Join(home, "one/skills/nested/acct/beta"), // so its SKILL.md edits are seen
		filepath.Join(home, "two/skills"),                  // a new account dir is seen
		filepath.Join(home, "two/skills/account"),          // a new skill dir is seen
		filepath.Join(home, "two/skills/account/beta"),     // and its SKILL.md edits are seen
		StoreDir(home),
		filepath.Join(StoreDir(home), "gamma"),
		appDir(home), // registering a root override re-syncs the watch set
	}
	for _, dir := range want {
		if !got[dir] {
			t.Errorf("missing watch dir %q in %v", dir, dirs)
		}
	}
	if got[filepath.Join(home, "missing/skills")] {
		t.Error("watched a root that does not exist")
	}
}

// TestWatcherCollapsesBurstIntoOneRefresh is the acceptance criterion: a burst
// of changes refreshes once, and the refresh's directories are watched after.
func TestWatcherCollapsesBurstIntoOneRefresh(t *testing.T) {
	home := fakeHome(t, "one/skills")
	adapters := adaptersFor("one/skills")
	root := filepath.Join(home, "one/skills")
	writeSkill(t, root, "alpha", alphaSkill)

	refreshed := make(chan string, 8)
	reported := make(chan error, 8)
	watcher, err := NewWatcher(
		WatchDirs(home, adapters),
		50*time.Millisecond,
		func() []string {
			refreshed <- ""
			return WatchDirs(home, adapters)
		},
		func(err error) { reported <- err },
	)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	writeSkill(t, root, "beta", betaSkill)
	writeSkill(t, root, "gamma", betaSkill)

	select {
	case <-refreshed:
	case err := <-reported:
		t.Fatalf("watcher reported %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("no refresh after a burst of creates")
	}

	// The burst is one refresh: nothing follows the quiet period.
	select {
	case <-refreshed:
		t.Fatal("burst produced more than one refresh")
	case <-time.After(10 * 50 * time.Millisecond):
	}

	// An edit inside a skill directory is a change too.
	if err := os.WriteFile(filepath.Join(root, "beta", "SKILL.md"), []byte(alphaSkill), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-refreshed:
	case err := <-reported:
		t.Fatalf("watcher reported %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("no refresh after editing SKILL.md inside a watched skill")
	}
}

func TestWatcherReportsDirectoriesItCannotWatch(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root may watch a directory it cannot read")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o755) }()

	reported := make(chan error, 1)
	watcher, err := NewWatcher(
		[]string{dir},
		50*time.Millisecond,
		func() []string { return nil },
		func(err error) { reported <- err },
	)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	select {
	case err := <-reported:
		if err == nil {
			t.Fatal("nil error reported")
		}
	default:
		t.Fatal("unwatchable directory reported no error")
	}
}

func TestWatcherIgnoresDirectoriesThatVanish(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone")

	reported := make(chan error, 1)
	watcher, err := NewWatcher(
		[]string{missing},
		50*time.Millisecond,
		func() []string { return nil },
		func(err error) { reported <- err },
	)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	select {
	case err := <-reported:
		t.Fatalf("removed directory reported as a failure: %v", err)
	default:
	}
}

// TestWatcherSyncWatchesDirectoriesAddedLater covers registering a root after
// the app started: its changes must refresh without waiting for Rescan.
func TestWatcherSyncWatchesDirectoriesAddedLater(t *testing.T) {
	home := fakeHome(t, "one/skills", "extra")
	refreshed := make(chan string, 4)
	reported := make(chan error, 4)
	watcher, err := NewWatcher(
		[]string{filepath.Join(home, "one/skills")},
		50*time.Millisecond,
		func() []string { refreshed <- ""; return nil },
		func(err error) { reported <- err },
	)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	watcher.Sync([]string{filepath.Join(home, "extra")})
	writeSkill(t, filepath.Join(home, "extra"), "delta", betaSkill)

	select {
	case <-refreshed:
	case err := <-reported:
		t.Fatalf("watcher reported %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("no refresh after a change in a directory synced in later")
	}
}
