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
	"encoding/json"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestChangeTriggeredJobCreation(t *testing.T) {
	jobTemplate := batchv1.JobTemplateSpec{
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "busybox:latest",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	resources := []ResourceReference{
		{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       "test-config",
			Namespace:  "default",
		},
	}

	ctj := &ChangeTriggeredJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "triggers.changejob.dev/v1alpha",
			Kind:       "ChangeTriggeredJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ctj",
			Namespace: "default",
		},
		Spec: ChangeTriggeredJobSpec{
			JobTemplate: jobTemplate,
			Resources:   resources,
		},
	}

	if ctj.Name != "test-ctj" {
		t.Errorf("Expected name to be 'test-ctj', got %s", ctj.Name)
	}

	if len(ctj.Spec.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(ctj.Spec.Resources))
	}

	if ctj.Spec.Resources[0].Kind != "ConfigMap" {
		t.Errorf("Expected Kind to be 'ConfigMap', got %s", ctj.Spec.Resources[0].Kind)
	}
}

func TestResourceReferenceValidation(t *testing.T) {
	tests := []struct {
		name      string
		resource  ResourceReference
		expectErr bool
	}{
		{
			name: "valid namespaced resource",
			resource: ResourceReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "my-config",
				Namespace:  "default",
			},
			expectErr: false,
		},
		{
			name: "valid cluster-scoped resource",
			resource: ResourceReference{
				APIVersion: "v1",
				Kind:       "Node",
				Name:       "my-node",
			},
			expectErr: false,
		},
		{
			name: "resource with fields",
			resource: ResourceReference{
				APIVersion: "v1",
				Kind:       "Secret",
				Name:       "my-secret",
				Namespace:  "default",
				Fields:     []string{"data.password", "data.username"},
			},
			expectErr: false,
		},
		{
			name: "resource with wildcard field",
			resource: ResourceReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "my-deploy",
				Namespace:  "default",
				Fields:     []string{"*"},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resource.APIVersion == "" {
				t.Error("APIVersion should not be empty")
			}
			if tt.resource.Kind == "" {
				t.Error("Kind should not be empty")
			}
			if tt.resource.Name == "" {
				t.Error("Name should not be empty")
			}
		})
	}
}

func TestTriggerConditionEnum(t *testing.T) {
	tests := []struct {
		name      string
		condition TriggerCondition
		valid     bool
	}{
		{
			name:      "valid All condition",
			condition: TriggerConditionAll,
			valid:     true,
		},
		{
			name:      "valid Any condition",
			condition: TriggerConditionAny,
			valid:     true,
		},
		{
			name:      "string value All",
			condition: TriggerCondition("All"),
			valid:     true,
		},
		{
			name:      "string value Any",
			condition: TriggerCondition("Any"),
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validConditions := map[TriggerCondition]bool{
				TriggerConditionAll: true,
				TriggerConditionAny: true,
			}

			if _, ok := validConditions[tt.condition]; !ok && tt.valid {
				t.Errorf("Expected condition %v to be valid", tt.condition)
			}
		})
	}
}

func TestJobStateEnum(t *testing.T) {
	tests := []struct {
		name  string
		state JobState
		valid bool
	}{
		{
			name:  "valid Active state",
			state: JobStateActive,
			valid: true,
		},
		{
			name:  "valid Succeeded state",
			state: JobStateSucceeded,
			valid: true,
		},
		{
			name:  "valid Failed state",
			state: JobStateFailed,
			valid: true,
		},
		{
			name:  "string value Active",
			state: JobState("Active"),
			valid: true,
		},
		{
			name:  "string value Succeeded",
			state: JobState("Succeeded"),
			valid: true,
		},
		{
			name:  "string value Failed",
			state: JobState("Failed"),
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validStates := map[JobState]bool{
				JobStateActive:    true,
				JobStateSucceeded: true,
				JobStateFailed:    true,
			}

			if _, ok := validStates[tt.state]; !ok && tt.valid {
				t.Errorf("Expected state %v to be valid", tt.state)
			}
		})
	}
}

func TestChangeTriggeredJobStatus(t *testing.T) {
	now := metav1.Now()
	status := ChangeTriggeredJobStatus{
		Conditions: []metav1.Condition{
			{
				Type:               "Available",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: now,
				Reason:             "JobTriggered",
				Message:            "Job has been triggered successfully",
			},
		},
		ResourceHashes: []ResourceReferenceStatus{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "test-config",
				Namespace:  "default",
				Fields: []ResourceFieldHash{
					{
						Field:    "data.key1",
						LastHash: "abc123",
					},
				},
			},
		},
		LastTriggeredTime: &now,
		LastJobName:       "test-job-1",
		LastJobStatus:     JobStateSucceeded,
	}

	if len(status.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(status.Conditions))
	}

	if status.Conditions[0].Type != "Available" {
		t.Errorf("Expected condition type 'Available', got %s", status.Conditions[0].Type)
	}

	if len(status.ResourceHashes) != 1 {
		t.Errorf("Expected 1 resource hash, got %d", len(status.ResourceHashes))
	}

	if status.LastJobStatus != JobStateSucceeded {
		t.Errorf("Expected last job status to be Succeeded, got %s", status.LastJobStatus)
	}

	if status.LastJobName != "test-job-1" {
		t.Errorf("Expected last job name to be 'test-job-1', got %s", status.LastJobName)
	}
}

func TestResourceFieldHash(t *testing.T) {
	tests := []struct {
		name           string
		field          string
		hash           string
		expectNonEmpty bool
	}{
		{
			name:           "data field hash",
			field:          "data.password",
			hash:           "hash123",
			expectNonEmpty: true,
		},
		{
			name:           "wildcard field hash",
			field:          "*",
			hash:           "fullhash456",
			expectNonEmpty: true,
		},
		{
			name:           "nested field hash",
			field:          "spec.template.spec.containers[0].image",
			hash:           "imagehash789",
			expectNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldHash := ResourceFieldHash{
				Field:    tt.field,
				LastHash: tt.hash,
			}

			if tt.expectNonEmpty && fieldHash.Field == "" {
				t.Error("Field should not be empty")
			}

			if tt.expectNonEmpty && fieldHash.LastHash == "" {
				t.Error("LastHash should not be empty")
			}
		})
	}
}

func TestChangeTriggeredJobList(t *testing.T) {
	list := &ChangeTriggeredJobList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "triggers.changejob.dev/v1alpha",
			Kind:       "ChangeTriggeredJobList",
		},
		Items: []ChangeTriggeredJob{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ctj-1",
					Namespace: "default",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ctj-2",
					Namespace: "default",
				},
			},
		},
	}

	if len(list.Items) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(list.Items))
	}

	if list.Items[0].Name != "ctj-1" {
		t.Errorf("Expected first item name to be 'ctj-1', got %s", list.Items[0].Name)
	}

	if list.Items[1].Name != "ctj-2" {
		t.Errorf("Expected second item name to be 'ctj-2', got %s", list.Items[1].Name)
	}
}

func TestChangeTriggeredJobSpecWithDefaults(t *testing.T) {
	tests := []struct {
		name            string
		spec            ChangeTriggeredJobSpec
		expectCondition *TriggerCondition
		expectCooldown  *metav1.Duration
		expectHistory   *int32
	}{
		{
			name: "with all optional fields",
			spec: ChangeTriggeredJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{},
				Resources:   []ResourceReference{},
				Condition:   ptrToTriggerCondition(TriggerConditionAll),
				Cooldown:    &metav1.Duration{Duration: 30 * time.Second},
				History:     ptrToInt32(10),
			},
			expectCondition: ptrToTriggerCondition(TriggerConditionAll),
			expectCooldown:  &metav1.Duration{Duration: 30 * time.Second},
			expectHistory:   ptrToInt32(10),
		},
		{
			name: "without optional fields",
			spec: ChangeTriggeredJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{},
				Resources:   []ResourceReference{},
			},
			expectCondition: nil,
			expectCooldown:  nil,
			expectHistory:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectCondition != nil && tt.spec.Condition == nil {
				t.Error("Expected Condition to be set")
			}
			if tt.expectCondition != nil && tt.spec.Condition != nil {
				if *tt.spec.Condition != *tt.expectCondition {
					t.Errorf("Expected condition %v, got %v", *tt.expectCondition, *tt.spec.Condition)
				}
			}
			if tt.expectCooldown != nil && tt.spec.Cooldown == nil {
				t.Error("Expected Cooldown to be set")
			}
			if tt.expectHistory != nil && tt.spec.History == nil {
				t.Error("Expected History to be set")
			}
		})
	}
}

func TestChangeTriggeredJobJSONMarshaling(t *testing.T) {
	ctj := &ChangeTriggeredJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "triggers.changejob.dev/v1alpha",
			Kind:       "ChangeTriggeredJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ctj",
			Namespace: "default",
		},
		Spec: ChangeTriggeredJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "busybox:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			},
			Resources: []ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-config",
					Namespace:  "default",
				},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(ctj)
	if err != nil {
		t.Fatalf("Failed to marshal ChangeTriggeredJob: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled ChangeTriggeredJob
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ChangeTriggeredJob: %v", err)
	}

	// Verify key fields
	if unmarshaled.Name != ctj.Name {
		t.Errorf("Expected name %s, got %s", ctj.Name, unmarshaled.Name)
	}

	if len(unmarshaled.Spec.Resources) != len(ctj.Spec.Resources) {
		t.Errorf("Expected %d resources, got %d", len(ctj.Spec.Resources), len(unmarshaled.Spec.Resources))
	}
}

func TestResourceReferenceWithMultipleFields(t *testing.T) {
	resource := ResourceReference{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       "my-config",
		Namespace:  "default",
		Fields: []string{
			"data.key1",
			"data.key2",
			"data.key3",
		},
	}

	if len(resource.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(resource.Fields))
	}

	expectedFields := map[string]bool{
		"data.key1": true,
		"data.key2": true,
		"data.key3": true,
	}

	for _, field := range resource.Fields {
		if !expectedFields[field] {
			t.Errorf("Unexpected field: %s", field)
		}
	}
}

func TestResourceReferenceStatusWithMultipleHashes(t *testing.T) {
	status := ResourceReferenceStatus{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       "test-config",
		Namespace:  "default",
		Fields: []ResourceFieldHash{
			{Field: "data.key1", LastHash: "hash1"},
			{Field: "data.key2", LastHash: "hash2"},
			{Field: "data.key3", LastHash: "hash3"},
		},
	}

	if len(status.Fields) != 3 {
		t.Errorf("Expected 3 field hashes, got %d", len(status.Fields))
	}

	// Verify each hash
	expectedHashes := map[string]string{
		"data.key1": "hash1",
		"data.key2": "hash2",
		"data.key3": "hash3",
	}

	for _, fieldHash := range status.Fields {
		expectedHash, ok := expectedHashes[fieldHash.Field]
		if !ok {
			t.Errorf("Unexpected field: %s", fieldHash.Field)
			continue
		}
		if fieldHash.LastHash != expectedHash {
			t.Errorf("Expected hash %s for field %s, got %s",
				expectedHash, fieldHash.Field, fieldHash.LastHash)
		}
	}
}

func TestChangeTriggeredJobConditions(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name       string
		conditions []metav1.Condition
		expectLen  int
	}{
		{
			name: "single condition",
			conditions: []metav1.Condition{
				{
					Type:               "Available",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "Ready",
					Message:            "ChangeTriggeredJob is ready",
				},
			},
			expectLen: 1,
		},
		{
			name: "multiple conditions",
			conditions: []metav1.Condition{
				{
					Type:               "Available",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "Ready",
					Message:            "Ready",
				},
				{
					Type:               "Progressing",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "JobTriggered",
					Message:            "Job triggered",
				},
			},
			expectLen: 2,
		},
		{
			name:       "no conditions",
			conditions: []metav1.Condition{},
			expectLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := ChangeTriggeredJobStatus{
				Conditions: tt.conditions,
			}

			if len(status.Conditions) != tt.expectLen {
				t.Errorf("Expected %d conditions, got %d", tt.expectLen, len(status.Conditions))
			}
		})
	}
}

// Helper functions
func ptrToTriggerCondition(c TriggerCondition) *TriggerCondition {
	return &c
}

func ptrToInt32(i int32) *int32 {
	return &i
}
