package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Rules struct {
	Allow []string `yaml:"allow"`
	Deny  []string `yaml:"deny"`
}

type Config struct {
	Root  string `yaml:"root"`
	Rules Rules  `yaml:"rules"`
}

type RepoConfig struct {
	Rules Rules `yaml:"rules"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Root == "" {
		return nil, errors.New("config: root is required")
	}

	return &cfg, nil
}

func LoadRepo(path string) (*RepoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &RepoConfig{}, nil
		}
		return nil, err
	}

	var cfg RepoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func MergeRules(global, repo Rules) Rules {
	return Rules{
		Allow: append(append([]string{}, global.Allow...), repo.Allow...),
		Deny:  append(append([]string{}, global.Deny...), repo.Deny...),
	}
}
