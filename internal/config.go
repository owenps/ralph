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

// GlobalConfig stores user preferences (in ~/.config/ralph/config.yaml)
type GlobalConfig struct {
	// Future: theme, default settings, etc.
	Initialized bool `yaml:"initialized"`
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

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
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
