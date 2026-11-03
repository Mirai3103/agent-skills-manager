package main

import "github.com/Mirai3103/agent-skills-manager/internal/skills"

// InventoryService exposes the read-only skill inventory to the frontend.
type InventoryService struct {
	home string
}

// NewInventoryService resolves $HOME once, at the app edge, and hands it to the
// domain package. Everything below this seam takes the base directory as an
// argument.
func NewInventoryService(home string) *InventoryService {
	return &InventoryService{home: home}
}

// Scan returns the inventory of every skills root that exists under $HOME.
func (s *InventoryService) Scan() skills.Inventory {
	return skills.Scan(s.home, skills.DefaultAdapters())
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
