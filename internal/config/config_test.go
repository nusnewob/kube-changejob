package config

import (
	"testing"
	"time"
)

func TestDefaultControllerConfig(t *testing.T) {
	if DefaultControllerConfig.PollInterval != 60*time.Second {
		t.Errorf("Expected DefaultControllerConfig.PollInterval to be 60s, got %v", DefaultControllerConfig.PollInterval)
	}
}

func TestControllerConfig(t *testing.T) {
	tests := []struct {
		name         string
		pollInterval time.Duration
	}{
		{
			name:         "10 seconds",
			pollInterval: 10 * time.Second,
		},
		{
			name:         "1 minute",
			pollInterval: 1 * time.Minute,
		},
		{
			name:         "5 minutes",
			pollInterval: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ControllerConfig{
				PollInterval: tt.pollInterval,
			}
			if cfg.PollInterval != tt.pollInterval {
				t.Errorf("Expected PollInterval to be %v, got %v", tt.pollInterval, cfg.PollInterval)
			}
		})
	}
}
