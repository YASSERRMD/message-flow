package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
}

func Load() Config {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return cfg
}
