package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         int
	APIURL       string
	APIParameter string
	APITimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:         8080,
		APIParameter: "q",
		APITimeout:   10 * time.Second,
		APIURL:       os.Getenv("API_URL"),
	}

	if value := os.Getenv("PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("invalid PORT %q: expected a number from 1 to 65535", value)
		}
		cfg.Port = port
	}
	if value := os.Getenv("API_PARAMETER"); value != "" {
		cfg.APIParameter = value
	}
	if value := os.Getenv("API_TIMEOUT"); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("invalid API_TIMEOUT %q: expected a positive duration such as 10s", value)
		}
		cfg.APITimeout = timeout
	}

	return cfg, nil
}
