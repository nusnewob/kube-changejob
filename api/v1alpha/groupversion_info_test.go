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

package v1alpha

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGroupVersionValues(t *testing.T) {
	expectedGroup := "triggers.changejob.dev"
	expectedVersion := "v1alpha"

	if GroupVersion.Group != expectedGroup {
		t.Errorf("Expected Group to be %s, got %s", expectedGroup, GroupVersion.Group)
	}

	if GroupVersion.Version != expectedVersion {
		t.Errorf("Expected Version to be %s, got %s", expectedVersion, GroupVersion.Version)
	}
}

func TestGroupVersionString(t *testing.T) {
	expected := "triggers.changejob.dev/v1alpha"
	actual := GroupVersion.String()

	if actual != expected {
		t.Errorf("Expected GroupVersion string to be %s, got %s", expected, actual)
	}
}

func TestGroupVersionWithKind(t *testing.T) {
	tests := []struct {
		name        string
		kind        string
		expectedGVK schema.GroupVersionKind
	}{
		{
			name: "ChangeTriggeredJob kind",
			kind: "ChangeTriggeredJob",
			expectedGVK: schema.GroupVersionKind{
				Group:   "triggers.changejob.dev",
				Version: "v1alpha",
				Kind:    "ChangeTriggeredJob",
			},
		},
		{
			name: "ChangeTriggeredJobList kind",
			kind: "ChangeTriggeredJobList",
			expectedGVK: schema.GroupVersionKind{
				Group:   "triggers.changejob.dev",
				Version: "v1alpha",
				Kind:    "ChangeTriggeredJobList",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvk := GroupVersion.WithKind(tt.kind)

			if gvk != tt.expectedGVK {
				t.Errorf("Expected GVK %v, got %v", tt.expectedGVK, gvk)
			}
		})
	}
}

func TestSchemeBuilderRegistration(t *testing.T) {
	scheme := runtime.NewScheme()

	if SchemeBuilder == nil {
		t.Fatal("SchemeBuilder should not be nil")
	}

	// Test that AddToScheme works
	err := AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Failed to add types to scheme: %v", err)
	}

	// Verify that our types are registered
	gvk := schema.GroupVersionKind{
		Group:   "triggers.changejob.dev",
		Version: "v1alpha",
		Kind:    "ChangeTriggeredJob",
	}

	// Check if the type is known
	if !scheme.Recognizes(gvk) {
		t.Errorf("Scheme does not recognize GroupVersionKind: %v", gvk)
	}
}

func TestAddToSchemeFunction(t *testing.T) {
	tests := []struct {
		name      string
		scheme    *runtime.Scheme
		expectErr bool
	}{
		{
			name:      "valid scheme",
			scheme:    runtime.NewScheme(),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddToScheme(tt.scheme)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSchemeRegisteredTypes(t *testing.T) {
	scheme := runtime.NewScheme()
	err := AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Failed to add types to scheme: %v", err)
	}

	// Test for ChangeTriggeredJob
	ctjGVK := schema.GroupVersionKind{
		Group:   "triggers.changejob.dev",
		Version: "v1alpha",
		Kind:    "ChangeTriggeredJob",
	}

	if !scheme.Recognizes(ctjGVK) {
		t.Errorf("Scheme should recognize ChangeTriggeredJob: %v", ctjGVK)
	}

	// Test for ChangeTriggeredJobList
	ctjListGVK := schema.GroupVersionKind{
		Group:   "triggers.changejob.dev",
		Version: "v1alpha",
		Kind:    "ChangeTriggeredJobList",
	}

	if !scheme.Recognizes(ctjListGVK) {
		t.Errorf("Scheme should recognize ChangeTriggeredJobList: %v", ctjListGVK)
	}
}

func TestGroupVersionResource(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		expected schema.GroupVersionResource
	}{
		{
			name:     "changetriggeredjobs resource",
			resource: "changetriggeredjobs",
			expected: schema.GroupVersionResource{
				Group:    "triggers.changejob.dev",
				Version:  "v1alpha",
				Resource: "changetriggeredjobs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvr := GroupVersion.WithResource(tt.resource)

			if gvr != tt.expected {
				t.Errorf("Expected GVR %v, got %v", tt.expected, gvr)
			}
		})
	}
}

func TestSchemeBuilderNotNil(t *testing.T) {
	if SchemeBuilder == nil {
		t.Error("SchemeBuilder should not be nil")
	}

	if SchemeBuilder.GroupVersion != GroupVersion {
		t.Errorf("SchemeBuilder.GroupVersion should be %v, got %v", GroupVersion, SchemeBuilder.GroupVersion)
	}
}

func TestMultipleSchemeRegistrations(t *testing.T) {
	// Test that registering multiple times doesn't cause issues
	scheme := runtime.NewScheme()

	err := AddToScheme(scheme)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Register again - should handle gracefully
	err = AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Second registration failed: %v", err)
	}
}
