package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
	FrontendOrigin string
}

func Load() Config {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		FrontendOrigin: os.Getenv("FRONTEND_ORIGIN"),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.FrontendOrigin == "" {
		cfg.FrontendOrigin = "http://localhost:5173"
	}
	return cfg
}
