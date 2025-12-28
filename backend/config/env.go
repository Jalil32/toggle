package config

import (
	"os"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Router   RouterConfig
	Backend  BackendConfig
	Database PostgresConfig
	JWT      JWTConfig
}

type RouterConfig struct {
	GinMode string
}

type BackendConfig struct {
	Port string
}

type PostgresConfig struct {
	User     string
	Name     string
	Password string
	Host     string
	Port     string
	SslMode  string
}

type JWTConfig struct {
	JWKSURL  string
	Issuer   string
	Audience string
	SkipAuth bool
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Router: RouterConfig{
			GinMode: os.Getenv("GIN_MODE"),
		},
		Backend: BackendConfig{
			Port: os.Getenv("BACKEND_PORT"),
		},
		Database: PostgresConfig{
			User:     os.Getenv("POSTGRES_USER"),
			Name:     os.Getenv("POSTGRES_NAME"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			SslMode:  os.Getenv("POSTGRES_SSL_MODE"),
		},
		JWT: JWTConfig{
			JWKSURL:  os.Getenv("JWT_JWKS_URL"),
			Issuer:   os.Getenv("JWT_ISSUER"),
			Audience: os.Getenv("JWT_AUDIENCE"),
			SkipAuth: os.Getenv("SKIP_AUTH") == "true",
		},
	}
	return cfg, nil
}
