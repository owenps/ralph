package internal

import (
	"errors"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Category string

const (
	CategoryBug      Category = "bug"
	CategoryFeature  Category = "feature"
	CategoryRefactor Category = "refactor"
	CategoryResearch Category = "research"
	CategoryNotes    Category = "notes"
)

var Categories = []Category{CategoryBug, CategoryFeature, CategoryRefactor, CategoryResearch, CategoryNotes}

func (c Category) String() string {
	return string(c)
}

type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusActive  TaskStatus = "active"
	TaskStatusDone    TaskStatus = "done"
	TaskStatusFailed  TaskStatus = "failed"
)

type TaskSource string

const (
	TaskSourceLocal  TaskSource = "local"
	TaskSourceGitHub TaskSource = "github"
)

type Task struct {
	ID           string     `yaml:"id"`
	Category     Category   `yaml:"category"`
	Description  string     `yaml:"description"`
	Steps        []string   `yaml:"steps,omitempty"`
	Done         bool       `yaml:"done"`
	Status       TaskStatus `yaml:"status,omitempty"`
	Source       TaskSource `yaml:"source,omitempty"`
	IssueNumber  int        `yaml:"issue_number,omitempty"`
	Repo         string     `yaml:"repo,omitempty"`
	Branch       string     `yaml:"branch,omitempty"`
	WorktreePath string     `yaml:"worktree_path,omitempty"`
	PRNumber     int        `yaml:"pr_number,omitempty"`
}

func NewTask(category Category, description string, steps []string) *Task {
	return &Task{
		ID:          generateID(),
		Category:    category,
		Description: description,
		Steps:       steps,
		Done:        false,
		Status:      TaskStatusPending,
		Source:      TaskSourceLocal,
	}
}

func generateID() string {
	return time.Now().Format("20060102150405")
}

type TaskStore struct {
	filepath string
	tasks    []Task
}

func NewTaskStore(filepath string) *TaskStore {
	return &TaskStore{
		filepath: filepath,
		tasks:    []Task{},
	}
}

func (s *TaskStore) Load() error {
	data, err := os.ReadFile(s.filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.tasks = []Task{}
			return nil
		}
		return err
	}

	if len(data) == 0 {
		s.tasks = []Task{}
		return nil
	}

	var tasks []Task
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return err
	}

	s.tasks = tasks
	s.migrate()
	return nil
}

// migrate backfills Status and Source on old tasks that lack them.
func (s *TaskStore) migrate() {
	changed := false
	for i := range s.tasks {
		if s.tasks[i].Status == "" {
			if s.tasks[i].Done {
				s.tasks[i].Status = TaskStatusDone
			} else {
				s.tasks[i].Status = TaskStatusPending
			}
			changed = true
		}
		if s.tasks[i].Source == "" {
			s.tasks[i].Source = TaskSourceLocal
			changed = true
		}
	}
	if changed {
		s.Save()
	}
}

func (s *TaskStore) Save() error {
	data, err := yaml.Marshal(s.tasks)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filepath, data, 0644)
}

func (s *TaskStore) Add(t *Task) {
	s.tasks = append(s.tasks, *t)
}

func (s *TaskStore) All() []Task {
	return s.tasks
}

func (s *TaskStore) ByCategory(cat Category) []Task {
	var filtered []Task
	for _, t := range s.tasks {
		if t.Category == cat {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func (s *TaskStore) ByStatus(status TaskStatus) []Task {
	var filtered []Task
	for _, t := range s.tasks {
		if t.Status == status {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func (s *TaskStore) FindByID(id string) *Task {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			return &s.tasks[i]
		}
	}
	return nil
}

func (s *TaskStore) Count() int {
	return len(s.tasks)
}

func (s *TaskStore) Delete(id string) bool {
	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			return true
		}
	}
	return false
}

func (s *TaskStore) ToggleDone(id string) bool {
	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks[i].Done = !t.Done
			if s.tasks[i].Done {
				s.tasks[i].Status = TaskStatusDone
			} else {
				s.tasks[i].Status = TaskStatusPending
			}
			return true
		}
	}
	return false
}

func (s *TaskStore) SetStatus(id string, status TaskStatus) bool {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].Status = status
			if status == TaskStatusDone {
				s.tasks[i].Done = true
			}
			return true
		}
	}
	return false
}

func (s *TaskStore) SetBranch(id, branch string) bool {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].Branch = branch
			return true
		}
	}
	return false
}

func (s *TaskStore) SetWorktreePath(id, path string) bool {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].WorktreePath = path
			return true
		}
	}
	return false
}

func (s *TaskStore) SetPRNumber(id string, prNumber int) bool {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].PRNumber = prNumber
			return true
		}
	}
	return false
}

func (s *TaskStore) Update(id string, updated *Task) bool {
	for i, t := range s.tasks {
		if t.ID == id {
			// Preserve the original ID and status fields
			updated.ID = id
			updated.Done = t.Done
			updated.Status = t.Status
			updated.Source = t.Source
			updated.Branch = t.Branch
			updated.WorktreePath = t.WorktreePath
			updated.PRNumber = t.PRNumber
			updated.IssueNumber = t.IssueNumber
			updated.Repo = t.Repo
			s.tasks[i] = *updated
			return true
		}
	}
	return false
}

func (s *TaskStore) CreateEmptyFile() error {
	return os.WriteFile(s.filepath, []byte("[]\n"), 0644)
}

func TaskFileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return !errors.Is(err, os.ErrNotExist)
}
