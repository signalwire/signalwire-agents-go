package skills

import (
	"sort"
	"sync"
)

// registry holds all registered skill factories.
var (
	registry   = make(map[string]func(params map[string]any) SkillBase)
	registryMu sync.RWMutex
)

// RegisterSkill registers a skill factory function by name.
// This is typically called from init() functions in builtin skill packages.
func RegisterSkill(name string, factory func(params map[string]any) SkillBase) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// GetSkillFactory returns the factory function for a registered skill name.
// Returns nil if the skill is not registered.
func GetSkillFactory(name string) func(params map[string]any) SkillBase {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[name]
}

// ListSkills returns sorted names of all registered skills.
func ListSkills() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
