package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// StoreDir is the app-owned Skill store: one canonical copy per adopted skill
// (ADR-0002). It is not a skills root and never becomes a matrix column.
func StoreDir(home string) string {
	return filepath.Join(home, ".agent-skills", "store")
}

// Suffixes for the short-lived siblings Adopt renames originals to, so a
// rollback is a rename away. Visible on purpose if a crash leaves one behind.
const (
	adoptOldSuffix  = ".adopt-old"
	adoptTempSuffix = ".adopt-tmp"
)

// AdoptResult reports what one Adopt call changed.
type AdoptResult struct {
	StorePath string   `json:"storePath"`
	Linked    []string `json:"linked"`
}

// Adopt copies a Loose skill into the store, verifies the copy's SKILL.md hash
// against the source, and only then replaces every loose original with a
// relative Link into the store. A Variant, an unlinkable root, or a failed
// write rolls the roots back and leaves every original as it was.
func Adopt(home, name string, adapters []Adapter) (AdoptResult, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return AdoptResult{}, fmt.Errorf("invalid skill name %q", name)
	}

	inv := Scan(home, adapters)
	skill, ok := findSkill(inv, name)
	if !ok {
		return AdoptResult{}, fmt.Errorf("no skill named %q", name)
	}
	if skill.Status == StatusVariant {
		return AdoptResult{}, fmt.Errorf("cannot adopt %q: variants exist, pick the canonical copy first", name)
	}

	// Pre-flight every write, so a refusal happens before anything changes.
	var originals []string
	for i, cell := range skill.Cells {
		if cell.State != Loose {
			continue
		}
		if !inv.Roots[i].Linkable {
			return AdoptResult{}, fmt.Errorf("cannot adopt %q: %s is a read-only root", name, inv.Roots[i].Label)
		}
		if _, err := os.Lstat(cell.Path + adoptOldSuffix); err == nil {
			return AdoptResult{}, fmt.Errorf("cannot adopt %q: %s already exists", name, cell.Path+adoptOldSuffix)
		}
		originals = append(originals, cell.Path)
	}
	if len(originals) == 0 {
		return AdoptResult{}, fmt.Errorf("cannot adopt %q: it is not loose in any root", name)
	}

	// Every original holds the row's single hash, so any of them is a valid source.
	source := originals[0]
	sourceHash, _, ok := readSkill(source)
	if !ok {
		return AdoptResult{}, fmt.Errorf("cannot adopt %q: reading %s", name, filepath.Join(source, "SKILL.md"))
	}

	storePath := filepath.Join(StoreDir(home), name)
	created := false
	if hash, _, ok := readSkill(storePath); ok {
		if hash != sourceHash {
			return AdoptResult{}, fmt.Errorf("cannot adopt %q: the store copy differs from the source", name)
		}
	} else {
		if err := copySkill(source, storePath); err != nil {
			return AdoptResult{}, fmt.Errorf("adopt %q: copying into the store: %w", name, err)
		}
		created = true
		hash, _, ok := readSkill(storePath)
		if !ok || hash != sourceHash {
			_ = os.RemoveAll(storePath)
			return AdoptResult{}, fmt.Errorf("adopt %q: the store copy does not match the source hash", name)
		}
	}

	type swap struct{ original, temp string }
	var swaps []swap
	fail := func(err error) (AdoptResult, error) {
		for i := len(swaps) - 1; i >= 0; i-- {
			_ = os.Remove(swaps[i].original)                // the link, if it got created
			_ = os.Rename(swaps[i].temp, swaps[i].original) // the original, back in place
		}
		if created {
			_ = os.RemoveAll(storePath)
		}
		return AdoptResult{}, err
	}

	// SKILL.md hashes can agree while a root still holds different assets, so
	// compare every original's whole tree before any of them is replaced.
	for _, original := range originals {
		diff, err := diffTree(storePath, original)
		if err != nil {
			return fail(fmt.Errorf("adopt %q: comparing %s with the store copy: %w", name, original, err))
		}
		if diff != "" {
			return fail(fmt.Errorf("cannot adopt %q: %s differs from the store copy; resolve the copies by hand first", name, filepath.Join(original, diff)))
		}
	}

	for _, original := range originals {
		temp := original + adoptOldSuffix
		if err := os.Rename(original, temp); err != nil {
			return fail(fmt.Errorf("adopt %q: moving %s aside: %w", name, original, err))
		}
		swaps = append(swaps, swap{original: original, temp: temp})

		rel, err := filepath.Rel(filepath.Dir(original), storePath)
		if err != nil {
			return fail(fmt.Errorf("adopt %q: linking %s: %w", name, original, err))
		}
		if err := os.Symlink(rel, original); err != nil {
			return fail(fmt.Errorf("adopt %q: linking %s: %w", name, original, err))
		}
	}

	// Each temp equals the verified store copy, so removing it loses nothing:
	// the store keeps the only real copy in play.
	for _, s := range swaps {
		if err := os.RemoveAll(s.temp); err != nil {
			return AdoptResult{StorePath: storePath, Linked: originals},
				fmt.Errorf("adopt %q: stored and linked, but removing the replaced copy %s failed: %w", name, s.temp, err)
		}
	}
	return AdoptResult{StorePath: storePath, Linked: originals}, nil
}

func findSkill(inv Inventory, name string) (Skill, bool) {
	for _, skill := range inv.Skills {
		if skill.Name == name {
			return skill, true
		}
	}
	return Skill{}, false
}

// diffTree names the first path where two skill trees disagree, or "" when
// they match. Content identity is bytes for files and target for symlinks.
func diffTree(a, b string) (string, error) {
	left, err := treeFingerprint(a)
	if err != nil {
		return "", err
	}
	right, err := treeFingerprint(b)
	if err != nil {
		return "", err
	}

	rels := make([]string, 0, len(left))
	for rel := range left {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		if fingerprint, ok := right[rel]; !ok || fingerprint != left[rel] {
			return rel, nil
		}
	}
	rels = rels[:0]
	for rel := range right {
		if _, ok := left[rel]; !ok {
			rels = append(rels, rel)
		}
	}
	sort.Strings(rels)
	if len(rels) > 0 {
		return rels[0], nil
	}
	return "", nil
}

func treeFingerprint(root string) (map[string]string, error) {
	fingerprint := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			fingerprint[rel] = "dir"
		case info.Mode()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			fingerprint[rel] = "link:" + link
		case info.Mode().IsRegular():
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(content)
			fingerprint[rel] = hex.EncodeToString(sum[:])
		default:
			fingerprint[rel] = "special"
		}
		return nil
	})
	return fingerprint, err
}

// copySkill copies source into dest through a hidden sibling, so a failed copy
// never looks like a complete store entry.
func copySkill(source, dest string) error {
	store := filepath.Dir(dest)
	if err := os.MkdirAll(store, 0o755); err != nil {
		return err
	}
	temp := filepath.Join(store, "."+filepath.Base(dest)+adoptTempSuffix)
	if err := os.RemoveAll(temp); err != nil {
		return err
	}
	if err := copyTree(source, temp); err != nil {
		_ = os.RemoveAll(temp)
		return err
	}
	if err := os.Rename(temp, dest); err != nil {
		_ = os.RemoveAll(temp)
		return err
	}
	return nil
}

func copyTree(source, dest string) error {
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		case info.Mode()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case info.Mode().IsRegular():
			return copyFile(path, target, info.Mode().Perm())
		default:
			// Sockets, devices, and FIFOs are not skill content.
			return nil
		}
	})
}

func copyFile(source, dest string, mode fs.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
