package main

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultGitHubOwner      string              `yaml:"defaultGithubOwner"`
	DefaultGitHubRepository string              `yaml:"defaultGithubRepository"`
	DefaultGitHubToken      string              `yaml:"defaultGithubToken"`
	DefaultLabels           []string            `yaml:"defaultLabels"`
	DefaultNoDefaultLabels  bool                `yaml:"defaultNoDefaultLabels"`
	RunnerGroups            []RunnerGroupConfig `yaml:"runnerGroups"`
}

type RunnerGroupConfig struct {
	Name             string   `yaml:"name"`
	Count            int      `yaml:"count"`
	GitHubOwner      string   `yaml:"githubOwner"`
	GitHubRepository string   `yaml:"githubRepository"`
	GitHubToken      string   `yaml:"githubToken"`
	Labels           []string `yaml:"labels"`
	NoDefaultLabels  *bool    `yaml:"noDefaultLabels"`
}

func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot open config file path: %s, err: %w", configPath, err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("Cannot parse config file: %s, err: %w", configPath, err)
	}

	overrideWithEnvironmentVariable(config)

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("Invalid config file: %s, err: %w", configPath, err)
	}

	config.canonicalize()

	return config, nil
}

func overrideWithEnvironmentVariable(config *Config) {
	if githubOwner := os.Getenv("DEFAULT_GITHUB_OWNER"); githubOwner != "" {
		slog.Info(fmt.Sprintf("Override defaultGithubOwner with env: %s", githubOwner))
		config.DefaultGitHubOwner = githubOwner
	}

	if githubRepo := os.Getenv("DEFAULT_GITHUB_REPOSITORY"); githubRepo != "" {
		slog.Info(fmt.Sprintf("Override defaultGithubRepository with env: %s", githubRepo))
		config.DefaultGitHubRepository = githubRepo
	}

	if githubToken := os.Getenv("DEFAULT_GITHUB_TOKEN"); githubToken != "" {
		slog.Info("Override defaultGithubToken with env: [REDACTED]")
		config.DefaultGitHubToken = githubToken
	}

	if labels := os.Getenv("DEFAULT_LABELS"); labels != "" {
		labelList := strings.Split(labels, ",")

		for i, label := range labelList {
			labelList[i] = strings.TrimSpace(label)
		}

		slog.Info(fmt.Sprintf("Override defaultLabels with env: %v", labelList))
		config.DefaultLabels = labelList
	}

	if noDefaultLabels := os.Getenv("DEFAULT_NO_DEFAULT_LABELS"); noDefaultLabels != "" {
		if noDefaultLabels == "true" || noDefaultLabels == "1" {
			slog.Info("Override defaultNoDefaultLabels with env: true")
			config.DefaultNoDefaultLabels = true
		} else if noDefaultLabels == "false" || noDefaultLabels == "0" {
			slog.Info("Override defaultNoDefaultLabels with env: false")
			config.DefaultNoDefaultLabels = false
		}
	}
}

func validateConfig(config *Config) error {
	if len(config.RunnerGroups) == 0 {
		return fmt.Errorf("at least one runner group must be defined")
	}

	allowedNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	for _, runnerConfig := range config.RunnerGroups {
		if runnerConfig.Name == "" {
			return fmt.Errorf("runner group name is required")
		}

		if !allowedNamePattern.MatchString(runnerConfig.Name) {
			return fmt.Errorf("runner group name '%s' contains invalid characters. Only alphanumeric characters, hyphens, and underscores are allowed", runnerConfig.Name)
		}

		if runnerConfig.Count <= 0 {
			return fmt.Errorf("runner group count must be greater than 0 for runner group: %s", runnerConfig.Name)
		}

		if runnerConfig.GitHubOwner == "" && config.DefaultGitHubOwner == "" {
			return fmt.Errorf("global defaultGithubOwner nor runner group specific githubOwner is set for runner group: %s", runnerConfig.Name)
		}

		if runnerConfig.GitHubRepository == "" && config.DefaultGitHubRepository == "" {
			return fmt.Errorf("global defaultGithubRepository nor runner group specific githubRepository is set for runner group: %s", runnerConfig.Name)
		}

		if runnerConfig.GitHubToken == "" && config.DefaultGitHubToken == "" {
			return fmt.Errorf("global defaultGithubToken nor runner group specific githubToken is set for runner group: %s", runnerConfig.Name)
		}
	}

	return nil
}

func (c *Config) canonicalize() {
	// This assumes that validation has already been done

	for i, runnerConfig := range c.RunnerGroups {
		if runnerConfig.GitHubOwner == "" {
			c.RunnerGroups[i].GitHubOwner = c.DefaultGitHubOwner
		}

		if runnerConfig.GitHubRepository == "" {
			c.RunnerGroups[i].GitHubRepository = c.DefaultGitHubRepository
		}

		if runnerConfig.GitHubToken == "" {
			c.RunnerGroups[i].GitHubToken = c.DefaultGitHubToken
		}

		if len(runnerConfig.Labels) == 0 && len(c.DefaultLabels) > 0 {
			c.RunnerGroups[i].Labels = c.DefaultLabels
		}

		if runnerConfig.NoDefaultLabels == nil {
			c.RunnerGroups[i].NoDefaultLabels = &c.DefaultNoDefaultLabels
		}
	}
}
