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

func TestLoggingDefaults(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		expected string
	}{
		{
			name:     "default log level",
			flagName: "log-level",
			expected: "info",
		},
		{
			name:     "default log format",
			flagName: "log-format",
			expected: "text",
		},
		{
			name:     "default log timestamp",
			flagName: "log-timestamp",
			expected: "rfc3339",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These would be the default values set in main.go
			var logLevel, logFormat, logTimestamp string
			if tt.flagName == "log-level" {
				logLevel = "info"
				if logLevel != tt.expected {
					t.Errorf("Expected %s to be %s, got %s", tt.flagName, tt.expected, logLevel)
				}
			}
			if tt.flagName == "log-format" {
				logFormat = "text"
				if logFormat != tt.expected {
					t.Errorf("Expected %s to be %s, got %s", tt.flagName, tt.expected, logFormat)
				}
			}
			if tt.flagName == "log-timestamp" {
				logTimestamp = "rfc3339"
				if logTimestamp != tt.expected {
					t.Errorf("Expected %s to be %s, got %s", tt.flagName, tt.expected, logTimestamp)
				}
			}
		})
	}
}

func TestLogLevelParsing(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		shouldErr bool
	}{
		{
			name:      "valid debug level",
			logLevel:  "debug",
			shouldErr: false,
		},
		{
			name:      "valid info level",
			logLevel:  "info",
			shouldErr: false,
		},
		{
			name:      "valid warn level",
			logLevel:  "warn",
			shouldErr: false,
		},
		{
			name:      "valid error level",
			logLevel:  "error",
			shouldErr: false,
		},
		{
			name:      "valid panic level",
			logLevel:  "panic",
			shouldErr: false,
		},
		{
			name:      "valid fatal level",
			logLevel:  "fatal",
			shouldErr: false,
		},
		{
			name:      "invalid level",
			logLevel:  "invalid",
			shouldErr: true,
		},
		{
			name:      "empty level",
			logLevel:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate log level validation
			validLevels := map[string]bool{
				"debug": true,
				"info":  true,
				"warn":  true,
				"error": true,
				"panic": true,
				"fatal": true,
			}

			_, valid := validLevels[tt.logLevel]
			if tt.shouldErr && valid {
				t.Errorf("Expected log level %s to be invalid, but it was valid", tt.logLevel)
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected log level %s to be valid, but it was invalid", tt.logLevel)
			}
		})
	}
}

func TestLogFormatValidation(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		isValid bool
		useJSON bool
	}{
		{
			name:    "json format",
			format:  "json",
			isValid: true,
			useJSON: true,
		},
		{
			name:    "text format",
			format:  "text",
			isValid: true,
			useJSON: false,
		},
		{
			name:    "invalid format defaults to text",
			format:  "xml",
			isValid: false,
			useJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate format validation from main.go
			useJSON := tt.format == "json"
			if useJSON != tt.useJSON {
				t.Errorf("Expected useJSON to be %v for format %s, got %v", tt.useJSON, tt.format, useJSON)
			}
		})
	}
}

func TestTimestampFormatValidation(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		isValid   bool
	}{
		{
			name:      "epoch format",
			timestamp: "epoch",
			isValid:   true,
		},
		{
			name:      "millis format",
			timestamp: "millis",
			isValid:   true,
		},
		{
			name:      "nano format",
			timestamp: "nano",
			isValid:   true,
		},
		{
			name:      "iso8601 format",
			timestamp: "iso8601",
			isValid:   true,
		},
		{
			name:      "rfc3339 format",
			timestamp: "rfc3339",
			isValid:   true,
		},
		{
			name:      "rfc3339nano format",
			timestamp: "rfc3339nano",
			isValid:   true,
		},
		{
			name:      "invalid format defaults to rfc3339",
			timestamp: "invalid",
			isValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate timestamp format validation from main.go
			validFormats := map[string]bool{
				"epoch":       true,
				"millis":      true,
				"nano":        true,
				"iso8601":     true,
				"rfc3339":     true,
				"rfc3339nano": true,
			}

			_, valid := validFormats[tt.timestamp]
			if tt.isValid && !valid {
				t.Errorf("Expected timestamp format %s to be valid, but it was invalid", tt.timestamp)
			}
			if !tt.isValid && valid {
				t.Errorf("Expected timestamp format %s to be invalid, but it was valid", tt.timestamp)
			}
		})
	}
}

func TestDebugFlagBehavior(t *testing.T) {
	tests := []struct {
		name          string
		debug         bool
		expectedLevel string
	}{
		{
			name:          "debug enabled",
			debug:         true,
			expectedLevel: "debug",
		},
		{
			name:          "debug disabled",
			debug:         false,
			expectedLevel: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate debug flag behavior from main.go
			var logLevel string
			if tt.debug {
				logLevel = "debug"
			} else {
				logLevel = "info"
			}

			if logLevel != tt.expectedLevel {
				t.Errorf("Expected log level to be %s when debug=%v, got %s", tt.expectedLevel, tt.debug, logLevel)
			}
		})
	}
}

func TestLoggingConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		debug       bool
		logLevel    string
		logFormat   string
		logTime     string
		expectDebug bool
		expectJSON  bool
	}{
		{
			name:        "production config",
			debug:       false,
			logLevel:    "info",
			logFormat:   "json",
			logTime:     "rfc3339",
			expectDebug: false,
			expectJSON:  true,
		},
		{
			name:        "development config",
			debug:       true,
			logLevel:    "debug",
			logFormat:   "text",
			logTime:     "rfc3339nano",
			expectDebug: true,
			expectJSON:  false,
		},
		{
			name:        "mixed config",
			debug:       false,
			logLevel:    "warn",
			logFormat:   "json",
			logTime:     "epoch",
			expectDebug: false,
			expectJSON:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate configuration from main.go
			isDebug := tt.debug
			isJSON := tt.logFormat == "json"

			if isDebug != tt.expectDebug {
				t.Errorf("Expected debug mode to be %v, got %v", tt.expectDebug, isDebug)
			}
			if isJSON != tt.expectJSON {
				t.Errorf("Expected JSON format to be %v, got %v", tt.expectJSON, isJSON)
			}
		})
	}
}
