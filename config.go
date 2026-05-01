// ABOUTME: Configuration loading for the reachy-mqtt bridge daemon.
// ABOUTME: Supports JSON config file with .env and environment variable overrides.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ReachyHost       string `json:"reachy_host"`
	ReachyPort       int    `json:"reachy_port"`
	ReachyName       string `json:"reachy_name"`
	MQTTBroker       string `json:"mqtt_broker"`
	MQTTClientID     string `json:"mqtt_client_id"`
	MQTTUsername     string `json:"mqtt_username"`
	MQTTPassword     string `json:"mqtt_password"`
	RESTPollInterval string `json:"rest_poll_interval"`
	WSReconnectDelay string `json:"ws_reconnect_delay"`
}

func DefaultConfig() Config {
	return Config{
		ReachyHost:       "localhost",
		ReachyPort:       8000,
		ReachyName:       "reachy",
		MQTTBroker:       "tcp://localhost:1883",
		MQTTClientID:     "reachy-mqtt-bridge",
		RESTPollInterval: "5s",
		WSReconnectDelay: "3s",
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	// Load .env file if present (silently skip if missing)
	_ = godotenv.Load()

	// Load JSON config file if path provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("read config file: %w", err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config file: %w", err)
		}
	}

	// Environment variables override everything
	if v := os.Getenv("REACHY_HOST"); v != "" {
		cfg.ReachyHost = v
	}
	if v := os.Getenv("REACHY_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 {
			cfg.ReachyPort = port
		}
	}
	if v := os.Getenv("REACHY_NAME"); v != "" {
		cfg.ReachyName = v
	}
	if v := os.Getenv("MQTT_BROKER"); v != "" {
		cfg.MQTTBroker = v
	}
	if v := os.Getenv("MQTT_CLIENT_ID"); v != "" {
		cfg.MQTTClientID = v
	}
	if v := os.Getenv("MQTT_USERNAME"); v != "" {
		cfg.MQTTUsername = v
	}
	if v := os.Getenv("MQTT_PASSWORD"); v != "" {
		cfg.MQTTPassword = v
	}
	if v := os.Getenv("REST_POLL_INTERVAL"); v != "" {
		cfg.RESTPollInterval = v
	}
	if v := os.Getenv("WS_RECONNECT_DELAY"); v != "" {
		cfg.WSReconnectDelay = v
	}

	return cfg, nil
}

func (c Config) WSURL() string {
	return fmt.Sprintf("ws://%s:%d/ws/sdk", c.ReachyHost, c.ReachyPort)
}

func (c Config) APIURL() string {
	return fmt.Sprintf("http://%s:%d", c.ReachyHost, c.ReachyPort)
}

func (c Config) RESTPollDuration() time.Duration {
	d, err := time.ParseDuration(c.RESTPollInterval)
	if err != nil {
		return 5 * time.Second
	}
	return d
}

func (c Config) WSReconnectDuration() time.Duration {
	d, err := time.ParseDuration(c.WSReconnectDelay)
	if err != nil {
		return 3 * time.Second
	}
	return d
}
