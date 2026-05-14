package config

import (
	"errors"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Environment      string         `yaml:"environment"`
	Log              LogConfig      `yaml:"log"`
	HTTP             HTTPConfig     `yaml:"http"`
	Auth             AuthConfig     `yaml:"auth"`
	Database         DatabaseConfig `yaml:"database"`
	Redis            RedisConfig    `yaml:"redis"`
	BusinessConfigDB bool           `yaml:"businessConfigDB"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type HTTPConfig struct {
	Addr              string        `yaml:"addr"`
	ReadHeaderTimeout time.Duration `yaml:"readHeaderTimeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdownTimeout"`
}

type AuthConfig struct {
	JWTSecret       string        `yaml:"jwtSecret"`
	AccessTokenTTL  time.Duration `yaml:"accessTokenTTL"`
	RefreshTokenTTL time.Duration `yaml:"refreshTokenTTL"`
}

type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"maxOpenConns"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
}

type RedisConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Addr     string        `yaml:"addr"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	TTL      time.Duration `yaml:"ttl"`
}

func Load() Config {
	cfg, err := LoadFile("config.yaml")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}
	return cfg
}

func LoadFile(path string) (Config, error) {
	cfg := defaultConfig()
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, err
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return cfg, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}

func defaultConfig() Config {
	cfg := Config{}
	applyDefaults(&cfg)
	return cfg
}

func applyDefaults(cfg *Config) {
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}
	if cfg.HTTP.ReadHeaderTimeout == 0 {
		cfg.HTTP.ReadHeaderTimeout = 5 * time.Second
	}
	if cfg.HTTP.ShutdownTimeout == 0 {
		cfg.HTTP.ShutdownTimeout = 10 * time.Second
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "dev-secret-change-me"
	}
	if cfg.Auth.AccessTokenTTL == 0 {
		cfg.Auth.AccessTokenTTL = 15 * time.Minute
	}
	if cfg.Auth.RefreshTokenTTL == 0 {
		cfg.Auth.RefreshTokenTTL = 7 * 24 * time.Hour
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "mysql"
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 30 * time.Minute
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "127.0.0.1:6379"
	}
	if cfg.Redis.TTL == 0 {
		cfg.Redis.TTL = time.Hour
	}
	cfg.BusinessConfigDB = true
}
