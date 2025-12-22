package main

import (
	"os"
	"testing"
	"time"

	"github.com/nusnewob/kube-changejob/internal/config"
)

func TestDefaultPollInterval(t *testing.T) {
	cfg := config.DefaultControllerConfig
	expected := 60 * time.Second
	if cfg.PollInterval != expected {
		t.Errorf("Expected default poll interval to be %v, got %v", expected, cfg.PollInterval)
	}
}

func TestEnvironmentVariableOverride(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expected    time.Duration
		shouldParse bool
	}{
		{
			name:        "valid 30s",
			envValue:    "30s",
			expected:    30 * time.Second,
			shouldParse: true,
		},
		{
			name:        "valid 2m",
			envValue:    "2m",
			expected:    2 * time.Minute,
			shouldParse: true,
		},
		{
			name:        "valid 1h",
			envValue:    "1h",
			expected:    1 * time.Hour,
			shouldParse: true,
		},
		{
			name:        "invalid value",
			envValue:    "invalid",
			expected:    60 * time.Second, // should keep default
			shouldParse: false,
		},
		{
			name:        "empty value",
			envValue:    "",
			expected:    60 * time.Second, // should keep default
			shouldParse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultControllerConfig

			if tt.envValue != "" {
				if d, err := time.ParseDuration(tt.envValue); err == nil {
					cfg.PollInterval = d
				}
			}

			if tt.shouldParse {
				if cfg.PollInterval != tt.expected {
					t.Errorf("Expected poll interval to be %v, got %v", tt.expected, cfg.PollInterval)
				}
			} else {
				if cfg.PollInterval != tt.expected {
					t.Errorf("Expected poll interval to remain default %v, got %v", tt.expected, cfg.PollInterval)
				}
			}
		})
	}
}

func TestConfigInitialization(t *testing.T) {
	// Save and restore original env
	originalEnv := os.Getenv("POLL_INTERVAL")
	defer func() {
		if originalEnv != "" {
			if err := os.Setenv("POLL_INTERVAL", originalEnv); err != nil {
				t.Errorf("Failed to set POLL_INTERVAL env: %v", err)
			}
		} else {
			if err := os.Unsetenv("POLL_INTERVAL"); err != nil {
				t.Errorf("Failed to unset POLL_INTERVAL env: %v", err)
			}
		}
	}()

	t.Run("default config without env", func(t *testing.T) {
		_ = os.Unsetenv("POLL_INTERVAL")
		cfg := config.DefaultControllerConfig

		// Simulate main.go logic
		if v := os.Getenv("POLL_INTERVAL"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				cfg.PollInterval = d
			}
		}

		expected := 60 * time.Second
		if cfg.PollInterval != expected {
			t.Errorf("Expected poll interval to be %v, got %v", expected, cfg.PollInterval)
		}
	})

	t.Run("config with env override", func(t *testing.T) {
		if err := os.Setenv("POLL_INTERVAL", "45s"); err != nil {
			t.Errorf("Failed to set POLL_INTERVAL env: %v", err)
		}
		cfg := config.DefaultControllerConfig

		// Simulate main.go logic
		if v := os.Getenv("POLL_INTERVAL"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				cfg.PollInterval = d
			}
		}

		expected := 45 * time.Second
		if cfg.PollInterval != expected {
			t.Errorf("Expected poll interval to be %v, got %v", expected, cfg.PollInterval)
		}
	})
}
