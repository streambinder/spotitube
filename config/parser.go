package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// Parse generates a new Config instance
// starting from a configuration file path
func Parse() (*Config, error) {
	config := new(Config)
	if _, err := os.Stat(Path()); os.IsNotExist(err) {
		return config, nil
	}

	file, err := os.Open(Path())
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return process(config)
}

func process(cfg *Config) (*Config, error) {
	if strings.Contains(cfg.Folder, "~/") {
		cfg.Folder = RelativeTo(strings.ReplaceAll(cfg.Folder, "~/", ""), HomePath)
	}

	return cfg, nil
}
