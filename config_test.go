// ABOUTME: Unit tests for configuration loading and defaults.
// ABOUTME: Verifies default values, JSON file loading, and environment variable overrides.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ReachyHost != "localhost" {
		t.Errorf("default ReachyHost = %q, want %q", cfg.ReachyHost, "localhost")
	}
	if cfg.ReachyPort != 8000 {
		t.Errorf("default ReachyPort = %d, want %d", cfg.ReachyPort, 8000)
	}
	if cfg.ReachyName != "reachy" {
		t.Errorf("default ReachyName = %q, want %q", cfg.ReachyName, "reachy")
	}
	if cfg.MQTTBroker != "tcp://localhost:1883" {
		t.Errorf("default MQTTBroker = %q, want %q", cfg.MQTTBroker, "tcp://localhost:1883")
	}
	if cfg.MQTTClientID != "reachy-mqtt-bridge" {
		t.Errorf("default MQTTClientID = %q, want %q", cfg.MQTTClientID, "reachy-mqtt-bridge")
	}
}

func TestConfigURLs(t *testing.T) {
	cfg := Config{
		ReachyHost: "192.168.1.50",
		ReachyPort: 8000,
	}

	if got := cfg.WSURL(); got != "ws://192.168.1.50:8000/ws/sdk" {
		t.Errorf("WSURL() = %q, want %q", got, "ws://192.168.1.50:8000/ws/sdk")
	}
	if got := cfg.APIURL(); got != "http://192.168.1.50:8000" {
		t.Errorf("APIURL() = %q, want %q", got, "http://192.168.1.50:8000")
	}
}

func TestConfigDurations(t *testing.T) {
	cfg := Config{
		RESTPollInterval: "10s",
		WSReconnectDelay: "2s",
	}

	if got := cfg.RESTPollDuration().Seconds(); got != 10 {
		t.Errorf("RESTPollDuration() = %v, want 10s", got)
	}
	if got := cfg.WSReconnectDuration().Seconds(); got != 2 {
		t.Errorf("WSReconnectDuration() = %v, want 2s", got)
	}
}

func TestConfigDurationDefaults(t *testing.T) {
	cfg := Config{
		RESTPollInterval: "invalid",
		WSReconnectDelay: "",
	}

	if got := cfg.RESTPollDuration().Seconds(); got != 5 {
		t.Errorf("RESTPollDuration() with invalid = %v, want 5s default", got)
	}
	if got := cfg.WSReconnectDuration().Seconds(); got != 3 {
		t.Errorf("WSReconnectDuration() with empty = %v, want 3s default", got)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Write a temp config file
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfgJSON := `{
		"reachy_host": "10.0.0.5",
		"reachy_port": 9000,
		"reachy_name": "mini",
		"mqtt_broker": "tcp://mqtt.local:1883"
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ReachyHost != "10.0.0.5" {
		t.Errorf("ReachyHost = %q, want %q", cfg.ReachyHost, "10.0.0.5")
	}
	if cfg.ReachyPort != 9000 {
		t.Errorf("ReachyPort = %d, want %d", cfg.ReachyPort, 9000)
	}
	if cfg.ReachyName != "mini" {
		t.Errorf("ReachyName = %q, want %q", cfg.ReachyName, "mini")
	}
	if cfg.MQTTBroker != "tcp://mqtt.local:1883" {
		t.Errorf("MQTTBroker = %q, want %q", cfg.MQTTBroker, "tcp://mqtt.local:1883")
	}
	// Unset fields should retain defaults
	if cfg.MQTTClientID != "reachy-mqtt-bridge" {
		t.Errorf("MQTTClientID = %q, want default %q", cfg.MQTTClientID, "reachy-mqtt-bridge")
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	// Write a config file with one value
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfgJSON := `{"reachy_host": "from-file", "reachy_port": 8000}`
	os.WriteFile(cfgPath, []byte(cfgJSON), 0644)

	// Set env var to override
	t.Setenv("REACHY_HOST", "from-env")
	t.Setenv("REACHY_PORT", "9999")

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ReachyHost != "from-env" {
		t.Errorf("ReachyHost = %q, want env override %q", cfg.ReachyHost, "from-env")
	}
	if cfg.ReachyPort != 9999 {
		t.Errorf("ReachyPort = %d, want env override %d", cfg.ReachyPort, 9999)
	}
}

func TestLoadConfigNoFile(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}

	// Should return defaults
	if cfg.ReachyHost != "localhost" {
		t.Errorf("ReachyHost = %q, want default %q", cfg.ReachyHost, "localhost")
	}
}

func TestLoadConfigBadFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("expected error for missing config file")
	}
}
