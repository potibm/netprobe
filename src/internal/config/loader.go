package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

func LoadConfig(filename string) (*Config, error) {
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return LoadConfigFromBytes(yamlFile)
}

func LoadConfigFromBytes(data []byte) (*Config, error) {
	c := &Config{}

	err := yaml.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
