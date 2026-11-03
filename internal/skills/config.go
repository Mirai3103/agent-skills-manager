package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// appDir is the app-owned directory. Nothing inside it may become a skills
// root: Adopt writes the store there, and a root pointing at it would replace
// the canonical copies with links to themselves.
func appDir(home string) string {
	return filepath.Join(home, ".agent-skills")
}

// ConfigPath is the only file the app persists (ADR-0002). It holds the extra
// Skills roots the adapters do not know about.
func ConfigPath(home string) string {
	return filepath.Join(appDir(home), "config.json")
}

// config is the on-disk shape. ExtraRoots holds absolute paths.
type config struct {
	ExtraRoots []string `json:"extraRoots"`
}

func readConfig(home string) (config, error) {
	var cfg config
	data, err := os.ReadFile(ConfigPath(home))
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", ConfigPath(home), err)
	}
	return cfg, nil
}

func writeConfig(home string, cfg config) error {
	path := ConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	// A sibling then a rename, like the store: a torn write must not wipe the
	// only file the app persists.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ExtraRoots reads the registered root overrides, resolved against home. A
// missing config file is not an error: it just means no overrides yet. One
// unresolvable entry is reported alone, so a bad hand edit hides no others.
func ExtraRoots(home string) ([]string, []RootError) {
	cfg, err := readConfig(home)
	if err != nil {
		return nil, []RootError{{Root: rootLabel(home, ConfigPath(home)), Err: err.Error()}}
	}
	roots := make([]string, 0, len(cfg.ExtraRoots))
	var rootErrors []RootError
	for _, raw := range cfg.ExtraRoots {
		path, err := resolveRoot(home, raw)
		if err != nil {
			rootErrors = append(rootErrors, RootError{Root: raw, Err: err.Error()})
			continue
		}
		roots = append(roots, path)
	}
	return roots, rootErrors
}

// AddExtraRoot registers one root override. The root does not have to exist:
// a missing root is reported by Scan, not refused here.
func AddExtraRoot(home, raw string) error {
	path, err := resolveRoot(home, raw)
	if err != nil {
		return err
	}
	app := appDir(home)
	if path == app || strings.HasPrefix(path, app+string(filepath.Separator)) {
		return fmt.Errorf("skills root %q is inside %s, which the app owns", raw, rootLabel(home, app))
	}
	cfg, err := readConfig(home)
	if err != nil {
		return err
	}
	if slices.Contains(cfg.ExtraRoots, path) {
		return fmt.Errorf("%s is already a registered skills root", path)
	}
	cfg.ExtraRoots = append(cfg.ExtraRoots, path)
	return writeConfig(home, cfg)
}

// RemoveExtraRoot forgets one root override and touches nothing on disk.
func RemoveExtraRoot(home, raw string) error {
	path, err := resolveRoot(home, raw)
	if err != nil {
		return err
	}
	cfg, err := readConfig(home)
	if err != nil {
		return err
	}
	before := len(cfg.ExtraRoots)
	cfg.ExtraRoots = slices.DeleteFunc(cfg.ExtraRoots, func(registered string) bool {
		if strings.TrimSpace(registered) == strings.TrimSpace(raw) {
			return true
		}
		resolved, err := resolveRoot(home, registered)
		return err == nil && resolved == path
	})
	if len(cfg.ExtraRoots) == before {
		return fmt.Errorf("%s is not a registered skills root", path)
	}
	return writeConfig(home, cfg)
}

// resolveRoot turns a user-supplied root into an absolute path. Relative paths
// are refused: they would mean something different after a restart.
func resolveRoot(home, raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", errors.New("skills root path is empty")
	}
	if path == "~" {
		path = home
	} else if rest, ok := strings.CutPrefix(path, "~/"); ok {
		path = filepath.Join(home, rest)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("skills root %q is not absolute; use an absolute path or ~/...", raw)
	}
	return filepath.Clean(path), nil
}
