package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	ModelConfig  ModelConfig  `yaml:"modelconfig"`
	ModelConfig1 ModelConfig  `yaml:"modelconfig1"`
	Server       ServerConfig `yaml:"server"`
	Db           DbConfig     `yaml:"db"`
	Redis        RedisConfig  `yaml:"redis"`
	Embed        EmbedConfig  `yaml:"textmodelconfig"`
}
type ModelConfig struct {
	Apikey  string `yaml:"apikey"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"baseURL"`
}
type EmbedConfig struct {
	Apikey  string `yaml:"apikey"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"baseURL"`
}
type DbConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
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
	setEnvIfNotEmpty("OPENAI_API_KEY", c.ModelConfig1.Apikey)
	setEnvIfNotEmpty("OPENAI_MODEL", c.ModelConfig1.Model)
	setEnvIfNotEmpty("OPENAI_BASE_URL", c.ModelConfig1.BaseURL)

}

func setEnvIfNotEmpty(key, value string) {
	if value == "" {
		return
	}
	if err := os.Setenv(key, value); err != nil {
		panic(fmt.Errorf("set env %s: %w", key, err))
	}
}
