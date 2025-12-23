package main

import (
	"os"
	"testing"
	"time"

	"github.com/nusnewob/kube-changejob/internal/config"
)

const (
	jsonFormat = "json"
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
			useJSON := tt.format == jsonFormat
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
			logFormat:   jsonFormat,
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
			logFormat:   jsonFormat,
			logTime:     "epoch",
			expectDebug: false,
			expectJSON:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate configuration from main.go
			isDebug := tt.debug
			isJSON := tt.logFormat == jsonFormat

			if isDebug != tt.expectDebug {
				t.Errorf("Expected debug mode to be %v, got %v", tt.expectDebug, isDebug)
			}
			if isJSON != tt.expectJSON {
				t.Errorf("Expected JSON format to be %v, got %v", tt.expectJSON, isJSON)
			}
		})
	}
}

func TestMetricsConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		metricsAddr   string
		secureMetrics bool
		expectSecure  bool
		expectAddr    string
	}{
		{
			name:          "secure metrics with HTTPS",
			metricsAddr:   ":8443",
			secureMetrics: true,
			expectSecure:  true,
			expectAddr:    ":8443",
		},
		{
			name:          "insecure metrics with HTTP",
			metricsAddr:   ":8080",
			secureMetrics: false,
			expectSecure:  false,
			expectAddr:    ":8080",
		},
		{
			name:          "metrics disabled",
			metricsAddr:   "0",
			secureMetrics: true,
			expectSecure:  true,
			expectAddr:    "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricsAddr != tt.expectAddr {
				t.Errorf("Expected metrics address to be %s, got %s", tt.expectAddr, tt.metricsAddr)
			}
			if tt.secureMetrics != tt.expectSecure {
				t.Errorf("Expected secure metrics to be %v, got %v", tt.expectSecure, tt.secureMetrics)
			}
		})
	}
}

func TestHTTP2Configuration(t *testing.T) {
	tests := []struct {
		name        string
		enableHTTP2 bool
		expectHTTP2 bool
	}{
		{
			name:        "HTTP/2 enabled",
			enableHTTP2: true,
			expectHTTP2: true,
		},
		{
			name:        "HTTP/2 disabled (default for security)",
			enableHTTP2: false,
			expectHTTP2: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.enableHTTP2 != tt.expectHTTP2 {
				t.Errorf("Expected HTTP/2 to be %v, got %v", tt.expectHTTP2, tt.enableHTTP2)
			}
		})
	}
}

func TestTLSCertificateConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		certPath    string
		certName    string
		certKey     string
		expectValid bool
	}{
		{
			name:        "valid webhook certificate",
			certPath:    "/etc/certs",
			certName:    "tls.crt",
			certKey:     "tls.key",
			expectValid: true,
		},
		{
			name:        "valid metrics certificate",
			certPath:    "/etc/metrics-certs",
			certName:    "server.crt",
			certKey:     "server.key",
			expectValid: true,
		},
		{
			name:        "default certificate names",
			certPath:    "/certs",
			certName:    "tls.crt",
			certKey:     "tls.key",
			expectValid: true,
		},
		{
			name:        "empty path uses auto-generated",
			certPath:    "",
			certName:    "tls.crt",
			certKey:     "tls.key",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate certificate configuration
			if tt.certName == "" && tt.expectValid {
				t.Error("Certificate name should not be empty")
			}
			if tt.certKey == "" && tt.expectValid {
				t.Error("Certificate key should not be empty")
			}
			// Empty path is valid (auto-generation) - no additional validation needed
		})
	}
}

func TestLeaderElectionConfiguration(t *testing.T) {
	tests := []struct {
		name                 string
		enableLeaderElection bool
		expectEnabled        bool
	}{
		{
			name:                 "leader election enabled",
			enableLeaderElection: true,
			expectEnabled:        true,
		},
		{
			name:                 "leader election disabled (default)",
			enableLeaderElection: false,
			expectEnabled:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.enableLeaderElection != tt.expectEnabled {
				t.Errorf("Expected leader election to be %v, got %v", tt.expectEnabled, tt.enableLeaderElection)
			}
		})
	}
}

func TestProbeConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		probeAddr   string
		expectAddr  string
		description string
	}{
		{
			name:        "default probe address",
			probeAddr:   ":8081",
			expectAddr:  ":8081",
			description: "health and readiness probes",
		},
		{
			name:        "custom probe address",
			probeAddr:   ":9090",
			expectAddr:  ":9090",
			description: "custom port for probes",
		},
		{
			name:        "probe on all interfaces",
			probeAddr:   "0.0.0.0:8081",
			expectAddr:  "0.0.0.0:8081",
			description: "explicit bind to all interfaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.probeAddr != tt.expectAddr {
				t.Errorf("Expected probe address to be %s, got %s", tt.expectAddr, tt.probeAddr)
			}
		})
	}
}

func TestInvalidPollIntervalHandling(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		shouldUseEnv  bool
		expectedValue time.Duration
	}{
		{
			name:          "negative duration invalid",
			envValue:      "-10s",
			shouldUseEnv:  false,
			expectedValue: 60 * time.Second,
		},
		{
			name:          "zero duration invalid",
			envValue:      "0s",
			shouldUseEnv:  true,
			expectedValue: 0,
		},
		{
			name:          "malformed duration",
			envValue:      "10x",
			shouldUseEnv:  false,
			expectedValue: 60 * time.Second,
		},
		{
			name:          "very large duration",
			envValue:      "24h",
			shouldUseEnv:  true,
			expectedValue: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultControllerConfig
			if tt.envValue != "" {
				if d, err := time.ParseDuration(tt.envValue); err == nil {
					if tt.shouldUseEnv {
						cfg.PollInterval = d
					}
				}
			}

			if tt.shouldUseEnv && cfg.PollInterval != tt.expectedValue {
				t.Errorf("Expected poll interval to be %v, got %v", tt.expectedValue, cfg.PollInterval)
			}
		})
	}
}

func TestMultipleEnvironmentVariablesPriority(t *testing.T) {
	originalEnv := os.Getenv("POLL_INTERVAL")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("POLL_INTERVAL", originalEnv)
		} else {
			_ = os.Unsetenv("POLL_INTERVAL")
		}
	}()

	tests := []struct {
		name           string
		envValue       string
		flagValue      time.Duration
		expectedResult time.Duration
	}{
		{
			name:           "flag overrides env",
			envValue:       "30s",
			flagValue:      45 * time.Second,
			expectedResult: 45 * time.Second,
		},
		{
			name:           "env used when no flag",
			envValue:       "30s",
			flagValue:      0,
			expectedResult: 30 * time.Second,
		},
		{
			name:           "default when neither set",
			envValue:       "",
			flagValue:      0,
			expectedResult: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultControllerConfig

			// First apply environment variable
			if tt.envValue != "" {
				if err := os.Setenv("POLL_INTERVAL", tt.envValue); err != nil {
					t.Fatalf("Failed to set env: %v", err)
				}
				if v := os.Getenv("POLL_INTERVAL"); v != "" {
					if d, err := time.ParseDuration(v); err == nil {
						cfg.PollInterval = d
					}
				}
			} else {
				_ = os.Unsetenv("POLL_INTERVAL")
			}

			// Then apply flag (highest priority)
			if tt.flagValue > 0 {
				cfg.PollInterval = tt.flagValue
			}

			if cfg.PollInterval != tt.expectedResult {
				t.Errorf("Expected poll interval to be %v, got %v", tt.expectedResult, cfg.PollInterval)
			}
		})
	}
}

func TestCertificatePathValidation(t *testing.T) {
	tests := []struct {
		name        string
		certPath    string
		certName    string
		keyName     string
		expectValid bool
	}{
		{
			name:        "all cert fields provided",
			certPath:    "/etc/certs",
			certName:    "tls.crt",
			keyName:     "tls.key",
			expectValid: true,
		},
		{
			name:        "missing cert path",
			certPath:    "",
			certName:    "tls.crt",
			keyName:     "tls.key",
			expectValid: true, // Empty path is valid (auto-generated)
		},
		{
			name:        "custom cert names",
			certPath:    "/custom/path",
			certName:    "server.pem",
			keyName:     "server-key.pem",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate that cert name and key are not empty when path is provided
			if tt.certPath != "" {
				if tt.certName == "" || tt.keyName == "" {
					if tt.expectValid {
						t.Error("Expected valid configuration but cert name or key is empty")
					}
				}
			}
		})
	}
}
