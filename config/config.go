package config

// Config represents the abstraction of the parsed
// configuration file
type Config struct {
	Folder  string              `yaml:"folder"`
	Aliases []map[string]string `yaml:"aliases"`
}

// URI returns the URI corresponding
// to the given alias key
func (cfg *Config) URI(alias string) (uri string) {
	if cfg.Aliases == nil {
		return
	}

	for _, entry := range cfg.Aliases {
		if value, ok := entry[alias]; ok {
			return value
		}
	}

	return
}
