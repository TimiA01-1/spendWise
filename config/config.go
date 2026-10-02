package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const minJWTSecretLen = 32

type Config struct{
	Env string
	HTTPPort string
	DatabaseURL string
	JWTSecret string
	JWTTTL time.Duration
}

func(c *Config) IsProduction() bool{return c.Env == "production"}

func Load()(*Config, error){
	_ = godotenv.Load()
	ttl, err := time.ParseDuration(getEnv("JWT_TTL", "24h"))
	if err != nil{
		return nil, fmt.Errorf("invalid JWT_TTL: %W", err)
	}

	cfg := &Config{
		Env: getEnv("APP_ENV", "development"),
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		JWTTTL: ttl,
	}
	if err := cfg.validate(); err != nil{
		return nil, err
	}
	return cfg, nil
}

func(c *Config) validate() error{
	var errs []error 
	if c.DatabaseURL == ""{
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if len(c.JWTSecret) < minJWTSecretLen{
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLen))
	}
	if c.JWTTTL <= 0 {
		errs = append(errs, errors.New("JWT_TTL must be positive"))
	}
	return errors.Join(errs...)
}

func getEnv(key, fallback string) string{
	if v := os.Getenv(key); v != ""{
		return v
	}
	return fallback
}