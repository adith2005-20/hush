package config

import (
    "fmt"
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

type Credentials struct {
    Server string `yaml:"server"`
    Token  string `yaml:"token"`
}

const (
    ProjectConfigFile = "hush.yaml"
    CredentialsFile   = "credentials.yaml"
)

func GetConfigDir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    
    configDir := filepath.Join(home, ".config", "hush")
    if err := os.MkdirAll(configDir, 0700); err != nil {
        return "", err
    }
    
    return configDir, nil
}

func LoadProjectConfig() (*Config, error) {
    data, err := os.ReadFile(ProjectConfigFile)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("no hush.yaml found. Run 'hush init' first")
        }
        return nil, fmt.Errorf("failed to read %s: %w", ProjectConfigFile, err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse %s: %w", ProjectConfigFile, err)
    }

    // Set defaults
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
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    return os.WriteFile(ProjectConfigFile, data, 0644)
}

func LoadCredentials() (*Credentials, error) {
    configDir, err := GetConfigDir()
    if err != nil {
        return nil, err
    }

    credPath := filepath.Join(configDir, CredentialsFile)
    data, err := os.ReadFile(credPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("not logged in. Run 'hush login' first")
        }
        return nil, fmt.Errorf("failed to read credentials: %w", err)
    }

    var creds Credentials
    if err := yaml.Unmarshal(data, &creds); err != nil {
        return nil, fmt.Errorf("failed to parse credentials: %w", err)
    }

    return &creds, nil
}

func SaveCredentials(server, token string) error {
    configDir, err := GetConfigDir()
    if err != nil {
        return err
    }

    creds := Credentials{
        Server: server,
        Token:  token,
    }

    data, err := yaml.Marshal(creds)
    if err != nil {
        return err
    }

    credPath := filepath.Join(configDir, CredentialsFile)
    return os.WriteFile(credPath, data, 0600)
}

func GetMasterKeyPath() (string, error) {
    configDir, err := GetConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(configDir, "master.key"), nil
}

func LoadMasterKey() ([]byte, error) {
    path, err := GetMasterKeyPath()
    if err != nil {
        return nil, err
    }
    return os.ReadFile(path)
}

func SaveMasterKey(key []byte) error {
    path, err := GetMasterKeyPath()
    if err != nil {
        return err
    }
    return os.WriteFile(path, key, 0600)
}
