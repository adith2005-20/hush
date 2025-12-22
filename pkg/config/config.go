package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Project     string       `yaml:"project"`
	Server      string       `yaml:"server"`
	Environment string       `yaml:"environment"`
	Output      OutputConfig `yaml:"output"`
	Secrets     []string     `yaml:"secrets,omitempty"`
	Prefix      string       `yaml:"prefix,omitempty"`
}

type OutputConfig struct {
	Format string `yaml:"format"`
	Path   string `yaml:"path"`
}

const configFileName = ".hush"

func Load() (*Config, error) {
	data, err := os.ReadFile(configFileName)
	if err != nil {
		return nil, errors.New("failed to read config")
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Output.Format == "" {
		cfg.Output.Format = "dotenv"
	}
	if cfg.Output.Path == "" {
		cfg.Output.Path = ".env"
	}
	if cfg.Environment == "" {
		cfg.Environment = "production"
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configFileName, data, 0o644)
}

func GetMasterKeyPath() string {
	home, _ := os.UserHomeDir()

	return filepath.Join(home, ".hush", "master.key")
}

func LoadMasterKey() ([]byte, error) {
	path := GetMasterKeyPath()
	return os.ReadFile(path)
}

func SaveMasterKey(key []byte) error {
	path := GetMasterKeyPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, key, 0o600)
}
