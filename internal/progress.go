package internal

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const ProgressFile = "progress.yaml"

type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusAborted   RunStatus = "aborted"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type TaskProgress struct {
	ID          string     `yaml:"id"`
	Description string     `yaml:"description"`
	Status      TaskStatus `yaml:"status"`
	StartedAt   *time.Time `yaml:"started_at,omitempty"`
	CompletedAt *time.Time `yaml:"completed_at,omitempty"`
	FailedAt    *time.Time `yaml:"failed_at,omitempty"`
	Iterations  int        `yaml:"iterations"`
	CommitSHA   string     `yaml:"commit_sha,omitempty"`
	Error       string     `yaml:"error,omitempty"`
}

type OutputLogEntry struct {
	Timestamp time.Time `yaml:"timestamp"`
	TaskID    string    `yaml:"task_id"`
	Iteration int       `yaml:"iteration"`
	Output    string    `yaml:"output"`
}

type Run struct {
	ID             string           `yaml:"id"`
	StartedAt      time.Time        `yaml:"started_at"`
	EndedAt        *time.Time       `yaml:"ended_at,omitempty"`
	Status         RunStatus        `yaml:"status"`
	MaxIterations  int              `yaml:"max_iterations"`
	IterationsUsed int              `yaml:"iterations_used"`
	Tasks          []TaskProgress   `yaml:"tasks"`
	OutputLog      []OutputLogEntry `yaml:"output_log,omitempty"`
}

type ProgressStore struct {
	Runs []Run `yaml:"runs"`
}

type ProgressManager struct {
	filepath string
	store    ProgressStore
}

func ProgressFilePath() (string, error) {
	projectPath, err := ProjectPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectPath, ProgressFile), nil
}

func NewProgressManager() (*ProgressManager, error) {
	path, err := ProgressFilePath()
	if err != nil {
		return nil, err
	}
	pm := &ProgressManager{
		filepath: path,
		store:    ProgressStore{Runs: []Run{}},
	}
	if err := pm.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return pm, nil
}

func (pm *ProgressManager) Load() error {
	data, err := os.ReadFile(pm.filepath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		pm.store = ProgressStore{Runs: []Run{}}
		return nil
	}

	return yaml.Unmarshal(data, &pm.store)
}

func (pm *ProgressManager) Save() error {
	data, err := yaml.Marshal(&pm.store)
	if err != nil {
		return err
	}
	return os.WriteFile(pm.filepath, data, 0644)
}

func (pm *ProgressManager) StartRun(taskIDs []string, tasks []Task, maxIterations int) *Run {
	now := time.Now()
	run := Run{
		ID:            now.Format("20060102150405"),
		StartedAt:     now,
		Status:        RunStatusRunning,
		MaxIterations: maxIterations,
		Tasks:         make([]TaskProgress, len(taskIDs)),
	}

	// Build task progress entries
	taskMap := make(map[string]Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	for i, id := range taskIDs {
		t := taskMap[id]
		run.Tasks[i] = TaskProgress{
			ID:          id,
			Description: t.Description,
			Status:      TaskStatusPending,
		}
	}

	pm.store.Runs = append(pm.store.Runs, run)
	return &pm.store.Runs[len(pm.store.Runs)-1]
}

func (pm *ProgressManager) GetCurrentRun() *Run {
	if len(pm.store.Runs) == 0 {
		return nil
	}
	return &pm.store.Runs[len(pm.store.Runs)-1]
}

func (pm *ProgressManager) GetAllRuns() []Run {
	return pm.store.Runs
}

func (pm *ProgressManager) StartTask(runID, taskID string) {
	run := pm.findRun(runID)
	if run == nil {
		return
	}

	for i := range run.Tasks {
		if run.Tasks[i].ID == taskID {
			now := time.Now()
			run.Tasks[i].Status = TaskStatusRunning
			run.Tasks[i].StartedAt = &now
			break
		}
	}
}

func (pm *ProgressManager) CompleteTask(runID, taskID, commitSHA string) {
	run := pm.findRun(runID)
	if run == nil {
		return
	}

	for i := range run.Tasks {
		if run.Tasks[i].ID == taskID {
			now := time.Now()
			run.Tasks[i].Status = TaskStatusCompleted
			run.Tasks[i].CompletedAt = &now
			run.Tasks[i].CommitSHA = commitSHA
			break
		}
	}
}

func (pm *ProgressManager) FailTask(runID, taskID, errorMsg string) {
	run := pm.findRun(runID)
	if run == nil {
		return
	}

	for i := range run.Tasks {
		if run.Tasks[i].ID == taskID {
			now := time.Now()
			run.Tasks[i].Status = TaskStatusFailed
			run.Tasks[i].FailedAt = &now
			run.Tasks[i].Error = errorMsg
			break
		}
	}
}

func (pm *ProgressManager) IncrementIteration(runID, taskID string) {
	run := pm.findRun(runID)
	if run == nil {
		return
	}

	run.IterationsUsed++

	for i := range run.Tasks {
		if run.Tasks[i].ID == taskID {
			run.Tasks[i].Iterations++
			break
		}
	}
}

func (pm *ProgressManager) AddOutputLog(runID, taskID string, iteration int, output string) {
	run := pm.findRun(runID)
	if run == nil {
		return
	}

	entry := OutputLogEntry{
		Timestamp: time.Now(),
		TaskID:    taskID,
		Iteration: iteration,
		Output:    output,
	}
	run.OutputLog = append(run.OutputLog, entry)
}

func (pm *ProgressManager) CompleteRun(runID string, status RunStatus, errorMsg string) {
	run := pm.findRun(runID)
	if run == nil {
		return
	}

	now := time.Now()
	run.EndedAt = &now
	run.Status = status

	// Mark any pending tasks as not run
	for i := range run.Tasks {
		if run.Tasks[i].Status == TaskStatusPending {
			run.Tasks[i].Status = TaskStatusFailed
			run.Tasks[i].Error = "Run " + string(status)
		}
	}
}

func (pm *ProgressManager) findRun(runID string) *Run {
	for i := range pm.store.Runs {
		if pm.store.Runs[i].ID == runID {
			return &pm.store.Runs[i]
		}
	}
	return nil
}

func (pm *ProgressManager) IsRunning() bool {
	run := pm.GetCurrentRun()
	return run != nil && run.Status == RunStatusRunning
}
