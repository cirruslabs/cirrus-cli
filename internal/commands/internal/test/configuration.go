package test

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
)

type Configuration struct {
	Environment   map[string]string `yaml:"env"`
	AffectedFiles []string          `yaml:"affected_files"`
}

func LoadConfiguration(path string) (*Configuration, error) {
	// Create an empty configuration
	config := &Configuration{
		Environment: make(map[string]string),
	}

	// Load configuration from file (if exists)
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}

		return nil, err
	}

	// Parse configuration
	if err := yaml.Unmarshal(yamlBytes, config); err != nil {
		return nil, err
	}

	return config, nil
}
