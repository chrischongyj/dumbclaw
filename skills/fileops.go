package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dumbclaw/config"
)

func init() {
	Register("file_operations", func(cfg *config.Config) Skill {
		return &FileOpsSkill{AllowedDirs: cfg.Skills.FileOps.AllowedDirs}
	})
}

// FileOpsSkill reads, writes, or lists files within allowed directories.
type FileOpsSkill struct {
	AllowedDirs []string
}

func (s *FileOpsSkill) Name() string { return "file_operations" }
func (s *FileOpsSkill) Description() string {
	return `Read, write, or list files. Params: {"action": "read"|"write"|"list", "path": "...", "content": "... (write only)"}`
}

func (s *FileOpsSkill) Execute(params map[string]any) (string, error) {
	action, _ := params["action"].(string)
	path, _ := params["path"].(string)

	if path == "" {
		return "", fmt.Errorf("missing required param: path")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !s.isAllowed(absPath) {
		return "", fmt.Errorf("access denied: %q is outside allowed directories", path)
	}

	switch action {
	case "read":
		data, err := os.ReadFile(absPath)
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "write":
		content, _ := params["content"].(string)
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return "", err
		}
		return "File written: " + absPath, nil

	case "list":
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return "", err
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		return strings.Join(names, "\n"), nil

	default:
		return "", fmt.Errorf("unknown action %q — use read, write, or list", action)
	}
}

func (s *FileOpsSkill) isAllowed(path string) bool {
	if len(s.AllowedDirs) == 0 {
		return true
	}
	for _, dir := range s.AllowedDirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(path, abs+string(filepath.Separator)) || path == abs {
			return true
		}
	}
	return false
}
