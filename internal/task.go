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
)

var Categories = []Category{CategoryBug, CategoryFeature, CategoryRefactor, CategoryResearch}

func (c Category) String() string {
	return string(c)
}

type Task struct {
	ID          string   `yaml:"id"`
	Category    Category `yaml:"category"`
	Description string   `yaml:"description"`
	Steps       []string `yaml:"steps,omitempty"`
	Done        bool     `yaml:"done"`
}

func NewTask(category Category, description string, steps []string) *Task {
	return &Task{
		ID:          generateID(),
		Category:    category,
		Description: description,
		Steps:       steps,
		Done:        false,
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
	return nil
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
