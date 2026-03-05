package skills

import (
	"log"

	"dumbclaw/config"
)

// Factory is a function that constructs a Skill from the app config.
type Factory func(cfg *config.Config) Skill

var registry = map[string]Factory{}

// Register adds a skill factory to the registry.
// Call this from an init() function in each skill file.
func Register(name string, f Factory) {
	registry[name] = f
}

// Load instantiates and returns all skills listed in cfg.Skills.Enabled.
func Load(cfg *config.Config) []Skill {
	var list []Skill
	for _, name := range cfg.Skills.Enabled {
		f, ok := registry[name]
		if !ok {
			log.Printf("Warning: unknown skill %q — skipping", name)
			continue
		}
		list = append(list, f(cfg))
		log.Printf("Loaded skill: %s", name)
	}
	return list
}
