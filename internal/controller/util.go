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

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
)

// Poller fetches and hashes Kubernetes resources
type Poller struct {
	Client client.Client
}

// Trigger Job
func (r *ChangeTriggeredJobReconciler) triggerJob(ctx context.Context, changeJob *triggersv1alpha.ChangeTriggeredJob) (*batchv1.Job, error) {
	// Generate unique job name using GenerateName to stay within K8s 63 char label limit
	// The job controller will add a unique suffix
	var labels map[string]string
	if changeJob.Labels != nil {
		labels = maps.Clone(changeJob.Labels)
	} else {
		labels = make(map[string]string)
	}
	labels[DefaultLabel] = changeJob.Name

	job := &batchv1.Job{}
	job.ObjectMeta = metav1.ObjectMeta{
		GenerateName: fmt.Sprintf("%s-", changeJob.Name),
		Namespace:    changeJob.Namespace,
		Annotations:  changeJob.Annotations,
		Labels:       labels,
	}
	job.Spec = changeJob.Spec.JobTemplate.Spec

	if err := controllerutil.SetControllerReference(changeJob, job, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, job); err != nil {
		return nil, err
	}

	log.Info("Job created", "job", job.Name)
	log.V(1).Info("Job created", "job", job)
	return job, nil
}

// Poll fetches the resource, extracts fields, and hashes them
func (p *Poller) Poll(ctx context.Context, ref triggersv1alpha.ResourceReference) (triggersv1alpha.ResourceReferenceStatus, error) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(ref.APIVersion)
	obj.SetKind(ref.Kind)

	key := client.ObjectKey{
		Name: ref.Name,
	}
	if ref.Namespace != "" {
		key.Namespace = ref.Namespace
	}

	if err := p.Client.Get(ctx, key, obj); err != nil {
		return triggersv1alpha.ResourceReferenceStatus{}, err
	}
	log.V(1).Info("Resource fetched", "resource", obj)

	hashes := make([]triggersv1alpha.ResourceFieldHash, 0, len(ref.Fields))
	extracted := make(map[string]any)

	for _, field := range ref.Fields {
		if field == "*" {
			val, err := HashObject(obj.Object)
			if err != nil {
				return triggersv1alpha.ResourceReferenceStatus{}, err
			}
			hashes = append(hashes, triggersv1alpha.ResourceFieldHash{
				Field:    field,
				LastHash: val,
			})
			continue
		}

		val, found, err := unstructured.NestedFieldNoCopy(
			obj.Object,
			strings.Split(field, ".")...,
		)
		if err != nil {
			return triggersv1alpha.ResourceReferenceStatus{}, err
		}
		if found {
			extracted[field] = val
			hash, err := HashObject(extracted)
			if err != nil {
				return triggersv1alpha.ResourceReferenceStatus{}, err
			}
			hashes = append(hashes, triggersv1alpha.ResourceFieldHash{
				Field:    field,
				LastHash: hash,
			})
		}
	}

	return triggersv1alpha.ResourceReferenceStatus{
		APIVersion: ref.APIVersion,
		Kind:       ref.Kind,
		Name:       ref.Name,
		Namespace:  ref.Namespace,
		Fields:     hashes,
	}, nil
}

// PollResources polls the resources referenced by the given ChangeTriggeredJob.
func (r *ChangeTriggeredJobReconciler) pollResources(ctx context.Context, changeJob *triggersv1alpha.ChangeTriggeredJob) (bool, []triggersv1alpha.ResourceReferenceStatus, error) {
	poller := Poller{Client: r.Client}

	// existing := IndexResourceStatuses(changeJob.Status.ResourceHashes)
	updated := make([]triggersv1alpha.ResourceReferenceStatus, 0, len(changeJob.Spec.Resources))

	changed := false
	changeCount := 0
	fieldCount := 0

	for _, ref := range changeJob.Spec.Resources {
		result, err := poller.Poll(ctx, ref)
		if err != nil {
			return false, nil, err
		}

		// Always add to updated list
		updated = append(updated, result)

		// Find existing hash for comparison
		lastIndexFound := slices.ContainsFunc(changeJob.Status.ResourceHashes, func(status triggersv1alpha.ResourceReferenceStatus) bool {
			return status.APIVersion == ref.APIVersion && status.Kind == ref.Kind && status.Namespace == ref.Namespace && status.Name == ref.Name
		})
		if !lastIndexFound {
			// First time seeing this resource - no comparison needed, just track it
			log.V(1).Info("First poll of resource, establishing baseline", "APIVersion", ref.APIVersion, "Kind", ref.Kind, "Namespace", ref.Namespace, "Name", ref.Name)
			continue
		}

		lastIndex := slices.IndexFunc(changeJob.Status.ResourceHashes, func(status triggersv1alpha.ResourceReferenceStatus) bool {
			return status.APIVersion == ref.APIVersion && status.Kind == ref.Kind && status.Namespace == ref.Namespace && status.Name == ref.Name
		})

		last := changeJob.Status.ResourceHashes[lastIndex]

		// Compare fields to detect changes
		for _, field := range last.Fields {
			fieldCount++

			updatedFieldFound := slices.ContainsFunc(result.Fields, func(resultField triggersv1alpha.ResourceFieldHash) bool {
				return resultField.Field == field.Field
			})

			if !updatedFieldFound {
				continue // Field no longer exists, skip it - this could be considered a change
			}

			updatedFieldIndex := slices.IndexFunc(result.Fields, func(resultField triggersv1alpha.ResourceFieldHash) bool {
				return resultField.Field == field.Field
			})

			if field.LastHash != result.Fields[updatedFieldIndex].LastHash {
				changeCount++
				log.V(1).Info("Resource field changed", "APIVersion", ref.APIVersion, "Kind", ref.Kind, "Namespace", ref.Namespace, "Name", ref.Name, "Field", field.Field)
			}
		}
	}

	if changeCount > 0 {
		log.V(1).Info(fmt.Sprintf("%d of %d resources changed", changeCount, len(changeJob.Spec.Resources)))
		switch *changeJob.Spec.Condition {
		case triggersv1alpha.TriggerConditionAny:
			changed = true
			log.V(1).Info("Trigger condition satisfied")
		case triggersv1alpha.TriggerConditionAll:
			if changeCount == fieldCount {
				changed = true
				log.V(1).Info("Trigger condition satisfied")
			} else {
				changed = false
				log.V(1).Info("Trigger condition not satisfied")
			}
		}
	}

	return changed, updated, nil
}

// Update Status
func (r *ChangeTriggeredJobReconciler) updateStatus(ctx context.Context, changeJob *triggersv1alpha.ChangeTriggeredJob, job *batchv1.Job, status []triggersv1alpha.ResourceReferenceStatus) error {
	changeJob.Status.LastJobName = job.Name
	// Use current time if StartTime is not set yet
	if job.Status.StartTime != nil {
		changeJob.Status.LastTriggeredTime = job.Status.StartTime
	} else {
		changeJob.Status.LastTriggeredTime = ptr.To(metav1.Now())
	}

	changeJob.Status.ResourceHashes = status
	if err := r.Status().Update(ctx, changeJob); err != nil {
		return err
	}

	log.Info("Status updated", "job", job.Name)
	return nil
}

// Get a list of owned Jobs
func (r *ChangeTriggeredJobReconciler) listOwnedJobs(ctx context.Context, changeJob *triggersv1alpha.ChangeTriggeredJob) ([]batchv1.Job, error) {
	var jobs batchv1.JobList
	if err := r.List(ctx, &jobs, client.InNamespace(changeJob.Namespace), client.MatchingLabels{"changejob.dev/owner": changeJob.Name}); err != nil {
		return nil, err
	}

	sort.Slice(jobs.Items, func(i, j int) bool {
		return jobs.Items[i].CreationTimestamp.After(jobs.Items[j].CreationTimestamp.Time)
	})

	return jobs.Items, nil
}

// ValidateGVK validates the GroupVersionKind for a given APIVersion and Kind.
func ValidateGVK(ctx context.Context, mapper meta.RESTMapper, apiVersion string, kind string, namespace string) (*schema.GroupVersionKind, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid apiVersion %q: %w", apiVersion, err)
	}

	// Find resource for Kind
	mapping, err := mapper.RESTMapping(schema.GroupKind{Group: gv.Group, Kind: kind}, gv.Version)
	if err != nil {
		return nil, fmt.Errorf("unknown kind %q in apiVersion %q", kind, apiVersion)
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace && namespace == "" {
		return nil, fmt.Errorf("namespace is required for namespaced resource %s", kind)
	}
	if mapping.Scope.Name() == meta.RESTScopeNameRoot && namespace != "" {
		return nil, fmt.Errorf("cluster-scoped resource %s must not have namespace", kind)
	}

	return &mapping.GroupVersionKind, nil
}

// hashObject produces a stable hash for arbitrary JSON data
func HashObject(obj map[string]any) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
