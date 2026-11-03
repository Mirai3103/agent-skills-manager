package main

import (
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Mirai3103/agent-skills-manager/internal/skills"
)

// inventoryEvent tells the frontend a fresh inventory is available. The event
// carries the inventory itself, so a refresh costs no extra round trip.
const inventoryEvent = "inventory:changed"

// watchQuiet is how long a burst of filesystem changes must stay silent before
// the matrix refreshes. A skill copy is several writes; one refresh per burst.
const watchQuiet = 300 * time.Millisecond

// InventoryService exposes the read-only skill inventory to the frontend.
type InventoryService struct {
	home string

	mu       sync.Mutex
	watchErr string
	watcher  *skills.Watcher
}

// NewInventoryService resolves $HOME once, at the app edge, and hands it to the
// domain package. Everything below this seam takes the base directory as an
// argument.
func NewInventoryService(home string) *InventoryService {
	return &InventoryService{home: home}
}

// ServiceStartup begins watching the skills roots and the store. A watcher that
// will not start is reported, not fatal: Rescan is the whole story then.
func (s *InventoryService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	watcher, err := skills.NewWatcher(
		skills.WatchDirs(s.home, skills.DefaultAdapters()),
		watchQuiet,
		func() []string {
			s.emit()
			return skills.WatchDirs(s.home, skills.DefaultAdapters())
		},
		s.watchFailed,
	)
	if err != nil {
		s.watchFailed(err)
		return nil
	}
	s.mu.Lock()
	s.watcher = watcher
	s.mu.Unlock()
	return nil
}

// ServiceShutdown stops watching files.
func (s *InventoryService) ServiceShutdown() error {
	s.mu.Lock()
	watcher := s.watcher
	s.mu.Unlock()
	if watcher != nil {
		watcher.Close()
	}
	return nil
}

// Scan returns the inventory of every skills root that exists under $HOME.
func (s *InventoryService) Scan() skills.Inventory {
	return s.inventory()
}

// inventory scans and stamps the watcher state, so the UI can say live refresh
// is unavailable without a second call.
func (s *InventoryService) inventory() skills.Inventory {
	inv := skills.Scan(s.home, skills.DefaultAdapters())
	s.mu.Lock()
	inv.WatchError = s.watchErr
	s.mu.Unlock()
	return inv
}

// emit pushes a freshly scanned inventory to every open window.
func (s *InventoryService) emit() {
	if app := application.Get(); app != nil {
		app.Event.Emit(inventoryEvent, s.inventory())
	}
}

// watchFailed records the first watcher failure and pushes it to the UI. Later
// failures change nothing, so a flapping watch does not spam the frontend.
func (s *InventoryService) watchFailed(err error) {
	s.mu.Lock()
	first := s.watchErr == ""
	if first {
		s.watchErr = err.Error()
	}
	s.mu.Unlock()
	if first {
		s.emit()
	}
}

// Adopt copies a loose skill into the store and replaces every loose original
// with a relative link into it. It only ever runs on an explicit user action.
func (s *InventoryService) Adopt(name string) (skills.AdoptResult, error) {
	return skills.Adopt(s.home, name, skills.DefaultAdapters())
}

// ChooseCanonical adopts one named loose copy as canonical and leaves every
// other copy exactly where it is. It only ever runs on an explicit user action.
func (s *InventoryService) ChooseCanonical(name, chosen string) (skills.AdoptResult, error) {
	return skills.ChooseCanonical(s.home, name, chosen, skills.DefaultAdapters())
}

// Enable links a stored skill into one skills root. It writes nothing and
// returns the warnings until the caller repeats the call with confirm=true.
func (s *InventoryService) Enable(name, root string, confirm bool) (skills.LinkResult, error) {
	return skills.Enable(s.home, name, root, confirm, skills.DefaultAdapters())
}

// Disable removes the link at one skills root and leaves the store untouched.
// It warns, and writes nothing, until the caller confirms.
func (s *InventoryService) Disable(name, root string, confirm bool) (skills.LinkResult, error) {
	return skills.Disable(s.home, name, root, confirm, skills.DefaultAdapters())
}

// AddExtraRoot registers a skills root the adapters do not know about. The root
// may be missing; Scan reports that per root instead of failing.
func (s *InventoryService) AddExtraRoot(path string) error {
	err := skills.AddExtraRoot(s.home, path)
	s.resync()
	return err
}

// RemoveExtraRoot forgets a registered root and touches nothing on disk.
func (s *InventoryService) RemoveExtraRoot(path string) error {
	err := skills.RemoveExtraRoot(s.home, path)
	s.resync()
	return err
}

// resync makes the watcher follow the roots the config now names. Without it a
// root added this session stays unwatched until an unrelated event refreshes.
func (s *InventoryService) resync() {
	s.mu.Lock()
	watcher := s.watcher
	s.mu.Unlock()
	if watcher != nil {
		watcher.Sync(skills.WatchDirs(s.home, skills.DefaultAdapters()))
	}
}
