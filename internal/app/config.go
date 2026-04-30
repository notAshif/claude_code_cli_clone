package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "configs/config.yaml"

var supportedProviders = []string{"ollama", "openai", "claude"}

var defaultModelsByProvider = map[string]string{
	"ollama": "llama3.1:8b",
	"openai": "gpt-4o-mini",
	"claude": "claude-3-5-sonnet-20241022",
}

// Config holds application configuration
type Config struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`

	Approval struct {
		Shell     string `yaml:"shell"`
		WriteFile string `yaml:"write_files"`
	} `yaml:"approval"`

	Workspace struct {
		MaxFiles     int `yaml:"max_files"`
		MaxFileBytes int `yaml:"max_file_bytes"`
	} `yaml:"workspace"`

	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	var c Config
	c.Provider = "ollama"
	c.Model = defaultModelsByProvider[c.Provider]
	c.Approval.Shell = "always"
	c.Approval.WriteFile = "always"
	c.Storage.Path = ".agent/sessions.db"
	c.Workspace.MaxFiles = 5000
	c.Workspace.MaxFileBytes = 200000
	return c
}

// LoadCliConfig loads configuration from a YAML file
func LoadCliConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil // Use defaults if file doesn't exist
		}
		return Config{}, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// SaveCliConfig persists configuration to a YAML file.
func SaveCliConfig(path string, cfg Config) error {
	if path == "" {
		path = DefaultConfigPath
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SupportedProviders returns the provider IDs accepted by the app.
func SupportedProviders() []string {
	return slices.Clone(supportedProviders)
}

// DefaultModelForProvider returns a productive default model for a provider.
func DefaultModelForProvider(provider string) (string, bool) {
	model, ok := defaultModelsByProvider[normalizeProvider(provider)]
	return model, ok
}

// Set applies a single key/value config update.
func (c *Config) Set(key, value string) error {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("config: %s cannot be empty", key)
	}

	switch key {
	case "provider":
		provider := normalizeProvider(value)
		if !IsSupportedProvider(provider) {
			return fmt.Errorf("config: unsupported provider %q (choose: %s)", value, strings.Join(supportedProviders, ", "))
		}
		c.Provider = provider
		if model, ok := DefaultModelForProvider(provider); ok {
			c.Model = model
		}
	case "model":
		c.Model = value
	case "approval.shell", "approval_shell", "shell":
		c.Approval.Shell = value
	case "approval.write_files", "approval_write", "write_files":
		c.Approval.WriteFile = value
	case "workspace.max_files", "max_files":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("config: workspace.max_files must be a positive integer")
		}
		c.Workspace.MaxFiles = n
	case "workspace.max_file_bytes", "max_file_bytes":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("config: workspace.max_file_bytes must be a positive integer")
		}
		c.Workspace.MaxFileBytes = n
	case "storage.path", "storage_path":
		c.Storage.Path = value
	default:
		return fmt.Errorf("config: unknown key %q", key)
	}

	return c.Validate()
}

// UseProviderModel sets provider and model together.
func (c *Config) UseProviderModel(provider, model string) error {
	provider = normalizeProvider(provider)
	if !IsSupportedProvider(provider) {
		return fmt.Errorf("config: unsupported provider %q (choose: %s)", provider, strings.Join(supportedProviders, ", "))
	}

	c.Provider = provider
	if strings.TrimSpace(model) == "" {
		defaultModel, ok := DefaultModelForProvider(provider)
		if !ok {
			return fmt.Errorf("config: no default model for provider %q", provider)
		}
		c.Model = defaultModel
	} else {
		c.Model = strings.TrimSpace(model)
	}

	return c.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	c.Provider = normalizeProvider(c.Provider)
	c.Model = strings.TrimSpace(c.Model)

	if c.Provider == "" {
		return errors.New("config: provider is required")
	}

	if !IsSupportedProvider(c.Provider) {
		return fmt.Errorf("config: unsupported provider %q (choose: %s)", c.Provider, strings.Join(supportedProviders, ", "))
	}

	if c.Model == "" {
		return errors.New("config: model is required")
	}

	return nil
}

func IsSupportedProvider(provider string) bool {
	return slices.Contains(supportedProviders, normalizeProvider(provider))
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
