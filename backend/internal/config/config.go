package config

import "os"

type Config struct {
	DatabaseURL    string
	Port           string
	JWTSecret      string
	FrontendOrigin string
	RedisURL       string
	MasterKey      string
}

func Load() Config {
	cfg := Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		Port:           os.Getenv("PORT"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		FrontendOrigin: os.Getenv("FRONTEND_ORIGIN"),
		RedisURL:       os.Getenv("REDIS_URL"),
		MasterKey:      os.Getenv("MASTER_KEY"),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.FrontendOrigin == "" {
		cfg.FrontendOrigin = "http://localhost:5173"
	}
	return cfg
}
