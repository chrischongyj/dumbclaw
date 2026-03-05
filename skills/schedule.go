package skills

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"dumbclaw/config"
)

var (
	schedulerPush    func(string)
	schedulerProcess func(string) string
	store            = &scheduleStore{}
)

// SetSchedulerCallbacks wires the schedule skill to the messaging layer.
// push sends a message to the user; process runs a task through the agent.
// Must be called before any scheduled jobs can fire.
func SetSchedulerCallbacks(push func(string), process func(string) string) {
	schedulerPush = push
	schedulerProcess = process
}

type job struct {
	ID       int
	Task     string
	Interval time.Duration
	NextRun  time.Time
}

type scheduleStore struct {
	mu   sync.Mutex
	jobs []*job
	next int
}

func (s *scheduleStore) add(task string, interval time.Duration) int {
	log.Printf("Scheduling new job: %q every %s", task, interval)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	s.jobs = append(s.jobs, &job{
		ID:       s.next,
		Task:     task,
		Interval: interval,
		NextRun:  time.Now().Add(interval),
	})
	return s.next
}

func (s *scheduleStore) list() []*job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*job, len(s.jobs))
	copy(out, s.jobs)
	return out
}

func (s *scheduleStore) remove(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, j := range s.jobs {
		if j.ID == id {
			s.jobs = append(s.jobs[:i], s.jobs[i+1:]...)
			return true
		}
	}
	return false
}

func (s *scheduleStore) run() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		var due []*job
		for _, j := range s.jobs {
			if now.After(j.NextRun) {
				j.NextRun = now.Add(j.Interval)
				due = append(due, j)
			}
		}
		s.mu.Unlock()

		for _, j := range due {
			if schedulerProcess == nil || schedulerPush == nil {
				log.Printf("Schedule: job #%d fired but callbacks not set", j.ID)
				continue
			}
			log.Printf("Schedule: running job #%d: %s", j.ID, j.Task)
			result := schedulerProcess(j.Task)
			schedulerPush(result)
		}
	}
}

type ScheduleSkill struct{}

func (s *ScheduleSkill) Name() string { return "schedule" }
func (s *ScheduleSkill) Description() string {
	return `Schedule recurring tasks. Params: action ("add"|"list"|"remove"), task (string, required for add), interval (e.g. "1h"/"30m", required for add), id (int, required for remove)`
}

func (s *ScheduleSkill) Execute(params map[string]any) (string, error) {
	action, _ := params["action"].(string)
	switch action {
	case "add":
		task, _ := params["task"].(string)
		intervalStr, _ := params["interval"].(string)
		if task == "" || intervalStr == "" {
			return "", fmt.Errorf("add requires task and interval")
		}
		dur, err := time.ParseDuration(intervalStr)
		if err != nil {
			return "", fmt.Errorf("invalid interval %q: %w", intervalStr, err)
		}
		id := store.add(task, dur)
		return fmt.Sprintf("Scheduled job #%d: %q every %s", id, task, intervalStr), nil

	case "list":
		jobs := store.list()
		if len(jobs) == 0 {
			return "No scheduled jobs.", nil
		}
		var sb strings.Builder
		for _, j := range jobs {
			fmt.Fprintf(&sb, "#%d: %q every %s (next: %s)\n", j.ID, j.Task, j.Interval, j.NextRun.Format("15:04:05"))
		}
		return strings.TrimSpace(sb.String()), nil

	case "remove":
		id, err := toInt(params["id"])
		if err != nil {
			return "", fmt.Errorf("remove requires a valid id")
		}
		if store.remove(id) {
			return fmt.Sprintf("Removed job #%d.", id), nil
		}
		return fmt.Sprintf("No job with id #%d.", id), nil

	default:
		return "", fmt.Errorf("unknown action %q; use add, list, or remove", action)
	}
}

func toInt(v any) (int, error) {
	switch x := v.(type) {
	case float64:
		return int(x), nil
	case int:
		return x, nil
	case string:
		var n int
		_, err := fmt.Sscanf(x, "%d", &n)
		return n, err
	}
	return 0, fmt.Errorf("not a number")
}

func init() {
	go store.run()
	Register("schedule", func(cfg *config.Config) Skill {
		return &ScheduleSkill{}
	})
}
