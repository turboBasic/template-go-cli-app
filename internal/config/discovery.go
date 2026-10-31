package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigFiles contains the optional user file and selected project file.
type ConfigFiles struct {
	User    string
	Project string
}

// DiscoverFiles finds the user config and the project config for a working directory.
func DiscoverFiles(workingDir, configID, explicitConfig string) (ConfigFiles, error) {
	workingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return ConfigFiles{}, fmt.Errorf("resolve working directory: %w", err)
	}
	if configID == "" {
		return ConfigFiles{}, fmt.Errorf("configuration identifier is required")
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ConfigFiles{}, fmt.Errorf("find user configuration directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	userPath := filepath.Join(configHome, configID, "config.toml")
	user, err := existingConfig(userPath)
	if err != nil {
		return ConfigFiles{}, fmt.Errorf("discover user configuration: %w", err)
	}
	projectPath := filepath.Join(workingDir, "."+configID+".toml")
	if explicitConfig != "" {
		projectPath = explicitConfig
		if !filepath.IsAbs(projectPath) {
			projectPath = filepath.Join(workingDir, projectPath)
		}
		project, err := requiredConfig(projectPath)
		if err != nil {
			return ConfigFiles{}, fmt.Errorf("load explicit configuration: %w", err)
		}
		return ConfigFiles{User: user, Project: project}, nil
	}
	project, err := existingConfig(projectPath)
	if err != nil {
		return ConfigFiles{}, fmt.Errorf("discover project configuration: %w", err)
	}
	return ConfigFiles{User: user, Project: project}, nil
}

func existingConfig(path string) (string, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", path)
	}
	return path, nil
}

func requiredConfig(path string) (string, error) {
	config, err := existingConfig(path)
	if err != nil {
		return "", err
	}
	if config == "" {
		return "", fmt.Errorf("file %s does not exist", path)
	}
	return config, nil
}
