package internal

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	GlobalConfigDir  = "ralph"
	GlobalConfigFile = "config.yaml"
	ProjectDir       = ".ralph"
	TasksFile        = "tasks.yaml"
)

var (
	ErrGlobalConfigNotFound = errors.New("global config not found")
	ErrProjectNotFound      = errors.New("project not initialized")
)

// LoopConfig stores settings for the Ralph loop execution
type LoopConfig struct {
	DefaultMaxIterations int `yaml:"default_max_iterations"`
}

// ClaudeConfig stores settings for Claude CLI invocation
type ClaudeConfig struct {
	Model          string   `yaml:"model,omitempty"`
	AllowedTools   []string `yaml:"allowed_tools,omitempty"`
	MaxTurns       int      `yaml:"max_turns,omitempty"`
	TimeoutSeconds int      `yaml:"timeout_seconds,omitempty"`
}

// GlobalConfig stores user preferences (in ~/.config/ralph/config.yaml)
type GlobalConfig struct {
	Initialized bool         `yaml:"initialized"`
	Loop        LoopConfig   `yaml:"loop"`
	Claude      ClaudeConfig `yaml:"claude"`
}

// DefaultGlobalConfig returns a GlobalConfig with sensible defaults
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Initialized: true,
		Loop: LoopConfig{
			DefaultMaxIterations: 10,
		},
		Claude: ClaudeConfig{
			TimeoutSeconds: 300,
		},
	}
}

// GlobalConfigPath returns the path to the global config file
func GlobalConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, GlobalConfigDir, GlobalConfigFile), nil
}

// LoadGlobalConfig loads the global config from ~/.config/ralph/config.yaml
func LoadGlobalConfig() (*GlobalConfig, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrGlobalConfigNotFound
		}
		return nil, err
	}

	// Start with defaults, then overlay loaded config
	cfg := DefaultGlobalConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Ensure defaults for zero values
	if cfg.Loop.DefaultMaxIterations == 0 {
		cfg.Loop.DefaultMaxIterations = 10
	}
	if cfg.Claude.TimeoutSeconds == 0 {
		cfg.Claude.TimeoutSeconds = 300
	}

	return cfg, nil
}

// SaveGlobalConfig saves the global config to ~/.config/ralph/config.yaml
func SaveGlobalConfig(cfg *GlobalConfig) error {
	path, err := GlobalConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GlobalConfigExists checks if the global config file exists
func GlobalConfigExists() bool {
	path, err := GlobalConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// ProjectPath returns the path to the .ralph directory in the current working directory
func ProjectPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ProjectDir), nil
}

// TasksFilePath returns the path to the tasks.yaml file in the current project
func TasksFilePath() (string, error) {
	projectPath, err := ProjectPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectPath, TasksFile), nil
}

// ProjectExists checks if the current directory has a .ralph project
func ProjectExists() bool {
	path, err := ProjectPath()
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// InitProject creates the .ralph directory and empty tasks.yaml in the current directory
func InitProject() error {
	projectPath, err := ProjectPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return err
	}

	tasksPath, err := TasksFilePath()
	if err != nil {
		return err
	}

	// Create empty tasks file if it doesn't exist
	if _, err := os.Stat(tasksPath); errors.Is(err, os.ErrNotExist) {
		store := NewTaskStore(tasksPath)
		return store.CreateEmptyFile()
	}

	return nil
}
