package skills

// Skill is the interface all skills must implement.
type Skill interface {
	Name() string
	Description() string
	Execute(params map[string]any) (string, error)
}
