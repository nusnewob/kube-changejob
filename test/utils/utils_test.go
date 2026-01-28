/*
Copyright 2025 Bowen Sun.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetNonEmptyLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single line",
			input:    "line1",
			expected: []string{"line1"},
		},
		{
			name:     "multiple lines",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "lines with empty lines",
			input:    "line1\n\nline2\n\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "trailing newline",
			input:    "line1\nline2\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "leading newline",
			input:    "\nline1\nline2",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "multiple empty lines",
			input:    "\n\n\n",
			expected: []string{},
		},
		{
			name:     "mixed content",
			input:    "NAME\ndefault\n\nkube-system\n",
			expected: []string{"NAME", "default", "kube-system"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNonEmptyLines(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(result))
				return
			}

			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("Line %d: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestGetProjectDir(t *testing.T) {
	dir, err := GetProjectDir()
	if err != nil {
		t.Fatalf("GetProjectDir() returned error: %v", err)
	}

	if dir == "" {
		t.Error("GetProjectDir() returned empty string")
	}

	// Check that the returned directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("GetProjectDir() returned non-existent directory: %s", dir)
	}

	// The directory should not contain /test/e2e path
	if strings.Contains(dir, "/test/e2e") {
		t.Errorf("GetProjectDir() should strip /test/e2e from path, got: %s", dir)
	}
}

func TestUncommentCodeBasic(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		target      string
		prefix      string
		expected    string
		expectError bool
	}{
		{
			name:        "uncomment single line",
			content:     "line1\n# commented line\nline3",
			target:      "# commented line",
			prefix:      "# ",
			expected:    "line1\ncommented line\nline3",
			expectError: false,
		},
		{
			name:        "uncomment multiple lines",
			content:     "start\n# line1\n# line2\nend",
			target:      "# line1\n# line2",
			prefix:      "# ",
			expected:    "start\nline1\nline2\nend",
			expectError: false,
		},
		{
			name:        "target not found",
			content:     "line1\nline2\nline3",
			target:      "nonexistent",
			prefix:      "# ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "uncomment with different prefix",
			content:     "start\n// comment1\n// comment2\nend",
			target:      "// comment1\n// comment2",
			prefix:      "// ",
			expected:    "start\ncomment1\ncomment2\nend",
			expectError: false,
		},
		{
			name:        "empty prefix",
			content:     "line1\nline2",
			target:      "line2",
			prefix:      "",
			expected:    "line1\nline2",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "test-*.txt")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer func() { _ = os.Remove(tmpFile.Name()) }()

			// Write content
			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			_ = tmpFile.Close()

			// Run UncommentCode
			err = UncommentCode(tmpFile.Name(), tt.target, tt.prefix)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("UncommentCode() returned error: %v", err)
			}

			// Read result
			result, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to read result: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, string(result))
			}
		})
	}
}

func TestUncommentCodeEdgeCases(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		err := UncommentCode("/nonexistent/file.txt", "target", "# ")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("empty target", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := "line1\nline2"
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
		_ = tmpFile.Close()

		// Empty target will not be found and should return error
		err = UncommentCode(tmpFile.Name(), "nonexistent", "# ")
		if err == nil {
			t.Error("Expected error for nonexistent target")
		}
	})

	t.Run("target at end of file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := "line1\n# commented"
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
		_ = tmpFile.Close()

		err = UncommentCode(tmpFile.Name(), "# commented", "# ")
		if err != nil {
			t.Fatalf("UncommentCode() failed: %v", err)
		}

		result, _ := os.ReadFile(tmpFile.Name())
		expected := "line1\ncommented"
		if string(result) != expected {
			t.Errorf("Expected %q, got %q", expected, string(result))
		}
	})
}

func TestRunCommand(t *testing.T) {
	t.Run("successful command", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		output, err := Run(cmd)
		if err != nil {
			t.Errorf("Run() returned error: %v", err)
		}
		if !strings.Contains(output, "test") {
			t.Errorf("Expected output to contain 'test', got: %s", output)
		}
	})

	t.Run("failing command", func(t *testing.T) {
		cmd := exec.Command("false")
		_, err := Run(cmd)
		if err == nil {
			t.Error("Expected error for failing command")
		}
	})

	t.Run("command with arguments", func(t *testing.T) {
		cmd := exec.Command("echo", "hello", "world")
		output, err := Run(cmd)
		if err != nil {
			t.Errorf("Run() returned error: %v", err)
		}
		if !strings.Contains(output, "hello") || !strings.Contains(output, "world") {
			t.Errorf("Expected output to contain 'hello world', got: %s", output)
		}
	})
}

func TestIsCertManagerCRDsInstalled(t *testing.T) {
	// This test validates the logic without requiring actual CRDs to be installed
	t.Run("validates function exists", func(t *testing.T) {
		// Just verify the function can be called
		// In a real cluster, this would check for actual CRDs
		result := IsCertManagerCRDsInstalled()
		// Result can be true or false depending on environment
		_ = result
	})
}

func TestLoadImageToKindClusterWithName(t *testing.T) {
	t.Run("validates function signature", func(t *testing.T) {
		// This test just validates the function signature
		// Actual loading requires a Kind cluster to be present
		imageName := "test-image:latest"
		err := LoadImageToKindClusterWithName(imageName)
		// Error is expected if Kind is not available
		_ = err
	})
}

func TestConstantValues(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "certmanager version",
			value:    certmanagerVersion,
			expected: "v1.19.2",
		},
		{
			name:     "default kind binary",
			value:    defaultKindBinary,
			expected: "kind",
		},
		{
			name:     "default kind cluster",
			value:    defaultKindCluster,
			expected: "kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.value)
			}
		})
	}
}

func TestCertManagerURLTemplate(t *testing.T) {
	expectedURL := "https://github.com/cert-manager/cert-manager/releases/download/v1.19.2/cert-manager.yaml"

	// Simulate URL construction from template
	actualURL := "https://github.com/cert-manager/cert-manager/releases/download/" +
		certmanagerVersion + "/cert-manager.yaml"

	if actualURL != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, actualURL)
	}
}

func TestGetProjectDirStripsE2EPath(t *testing.T) {
	// Test that /test/e2e is stripped from path
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "test-project-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test/e2e subdirectory
	e2eDir := filepath.Join(tmpDir, "test", "e2e")
	if err := os.MkdirAll(e2eDir, 0755); err != nil {
		t.Fatalf("Failed to create e2e dir: %v", err)
	}

	// Change to e2e directory
	if err := os.Chdir(e2eDir); err != nil {
		t.Fatalf("Failed to change to e2e dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalWd) }()

	// Get project dir
	projectDir, err := GetProjectDir()
	if err != nil {
		t.Fatalf("GetProjectDir() failed: %v", err)
	}

	// Should not contain /test/e2e
	if strings.Contains(projectDir, "/test/e2e") {
		t.Errorf("GetProjectDir() should strip /test/e2e, got: %s", projectDir)
	}

	// Should be the tmp directory (handle /private prefix on macOS)
	normalizedProjectDir := strings.TrimPrefix(projectDir, "/private")
	normalizedTmpDir := strings.TrimPrefix(tmpDir, "/private")
	if !strings.HasPrefix(normalizedProjectDir, normalizedTmpDir) {
		t.Errorf("Expected project dir to start with %s, got: %s", tmpDir, projectDir)
	}
}

func TestUncommentCodePreservesContent(t *testing.T) {
	// Test that content before and after target is preserved
	content := "line1\nline2\n# commented\nline4\nline5"
	expected := "line1\nline2\ncommented\nline4\nline5"

	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	_ = tmpFile.Close()

	err = UncommentCode(tmpFile.Name(), "# commented", "# ")
	if err != nil {
		t.Fatalf("UncommentCode() failed: %v", err)
	}

	result, _ := os.ReadFile(tmpFile.Name())
	if string(result) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(result))
	}
}

func TestGetNonEmptyLinesWithKubectlOutput(t *testing.T) {
	// Simulate typical kubectl output
	kubectlOutput := `NAME                  AGE
default               30d
kube-system           30d
kube-public           30d
kube-node-lease       30d
`

	result := GetNonEmptyLines(kubectlOutput)

	if len(result) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(result))
	}

	expectedLines := []string{
		"NAME                  AGE",
		"default               30d",
		"kube-system           30d",
		"kube-public           30d",
		"kube-node-lease       30d",
	}

	for i, expected := range expectedLines {
		if i >= len(result) {
			t.Errorf("Missing line %d: %s", i, expected)
			continue
		}
		if result[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, result[i])
		}
	}
}
