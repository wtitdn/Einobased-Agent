package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	ModelConfig ModelConfig  `yaml:"modelconfig"`
	Server      ServerConfig `yaml:"server"`
}
type ModelConfig struct {
	Apikey  string `yaml:"apikey"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"baseURL"`
}
type ServerConfig struct {
	Port int `yaml:"port"`
}

func Load(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", filename, err)
	}

	cfg.ApplyEnv()

	return cfg, nil
}

func LoadLocalDev(filename string) (Config, bool, error) {
	cfg, err := Load(filename)
	if err == nil {
		return cfg, false, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return DefaultLocalConfig(), true, nil
	}
	return Config{}, false, err
}
func DefaultLocalConfig() Config {
	cfg := Config{
		Server: ServerConfig{
			Port: 8080,
		},
	}
	return cfg
}

func (c Config) ApplyEnv() {
	setEnvIfNotEmpty("OPENAI_API_KEY", c.ModelConfig.Apikey)
	setEnvIfNotEmpty("OPENAI_MODEL", c.ModelConfig.Model)
	setEnvIfNotEmpty("OPENAI_BASE_URL", c.ModelConfig.BaseURL)
}

func setEnvIfNotEmpty(key, value string) {
	if value == "" {
		return
	}
	if err := os.Setenv(key, value); err != nil {
		panic(fmt.Errorf("set env %s: %w", key, err))
	}
}
