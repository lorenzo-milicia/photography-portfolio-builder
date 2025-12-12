package util

import (
	"os"

	"gopkg.in/yaml.v3"
)

// LoadYAML loads a YAML file into the provided structure
func LoadYAML(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}

// SaveYAML saves a structure to a YAML file
func SaveYAML(path string, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
