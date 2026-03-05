package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"dumbclaw/llm"
	"dumbclaw/skills"
)

// Agent coordinates the LLM and skills to respond to messages.
type Agent struct {
	llm     *llm.LLM
	skills  map[string]skills.Skill
	history []llm.Message
}

func New(l *llm.LLM, skillList []skills.Skill) *Agent {
	skillMap := make(map[string]skills.Skill, len(skillList))
	for _, s := range skillList {
		skillMap[s.Name()] = s
	}
	return &Agent{llm: l, skills: skillMap}
}

func (a *Agent) systemPrompt() string {
	var sb strings.Builder
	sb.WriteString("You are DumbClaw, a helpful AI assistant.\n\n")

	if len(a.skills) > 0 {
		sb.WriteString("Available skills:\n")
		for name, skill := range a.skills {
			fmt.Fprintf(&sb, "- %s: %s\n", name, skill.Description())
		}
		sb.WriteString(`
IMPORTANT: When you need to use a skill, output ONLY the raw JSON — no explanation, no markdown, no code block, nothing else:
{"skill": "skill_name", "params": {"key": "value"}}

Do not write anything before or after the JSON. If you do not need a skill, respond naturally.`)
	}

	return sb.String()
}

// ProcessMessage takes a user message and returns the agent's response.
func (a *Agent) ProcessMessage(text string) string {
	a.history = append(a.history, llm.Message{Role: "user", Content: text})

	messages := append([]llm.Message{{Role: "system", Content: a.systemPrompt()}}, a.history...)

	response, err := a.llm.Chat(messages)
	if err != nil {
		return fmt.Sprintf("LLM error: %v", err)
	}

	if skillResult, ok := a.trySkill(response); ok {
		a.history = append(a.history, llm.Message{Role: "assistant", Content: response})
		a.history = append(a.history, llm.Message{Role: "user", Content: "Skill result: " + skillResult + "\nPlease respond naturally based on this result."})

		messages = append([]llm.Message{{Role: "system", Content: a.systemPrompt()}}, a.history...)
		finalResponse, err := a.llm.Chat(messages)
		if err != nil {
			finalResponse = skillResult
		}

		a.history = append(a.history, llm.Message{Role: "assistant", Content: finalResponse})
		return finalResponse
	}

	a.history = append(a.history, llm.Message{Role: "assistant", Content: response})
	return response
}

// trySkill tries to parse response as a skill invocation and executes it.
// It extracts the first JSON object from the response, so it works even if
// the LLM wraps the JSON in markdown code blocks or adds surrounding text.
func (a *Agent) trySkill(response string) (string, bool) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return "", false
	}
	var call struct {
		Skill  string         `json:"skill"`
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &call); err != nil {
		return "", false
	}
	skill, ok := a.skills[call.Skill]
	if !ok {
		return "", false
	}
	result, err := skill.Execute(call.Params)
	if err != nil {
		return fmt.Sprintf("skill error: %v", err), true
	}
	return result, true
}

// extractJSON finds the first complete JSON object in s.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// Reset clears the conversation history.
func (a *Agent) Reset() {
	a.history = nil
}

// SkillNames returns the names of all loaded skills.
func (a *Agent) SkillNames() []string {
	names := make([]string, 0, len(a.skills))
	for name := range a.skills {
		names = append(names, name)
	}
	return names
}
