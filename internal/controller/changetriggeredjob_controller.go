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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"

	"github.com/r3labs/diff/v3"
)

// ChangeTriggeredJobReconciler reconciles a ChangeTriggeredJob object
type ChangeTriggeredJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=triggers.changejob.io,resources=changetriggeredjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=triggers.changejob.io,resources=changetriggeredjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=triggers.changejob.io,resources=changetriggeredjobs/finalizers,verbs=update

// Manage triggered jobs
// +kubebuilder:rbac:groups=batch,resources=job,verbs=get;list;watch;create;update;patch;delete
// Watched resources
// +kubebuilder:rbac:groups="",resources="",verbs=get;list;watch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *ChangeTriggeredJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ChangeTriggeredJob", req.NamespacedName)

	var changeJob triggersv1alpha.ChangeTriggeredJob
	if err := r.Get(ctx, req.NamespacedName, &changeJob); err != nil {
		log.Error(err, "unable to fetch ChangeTriggeredJob")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	updatedResourceStatusList := make([]triggersv1alpha.ResourceReferenceStatus, 0, len(changeJob.Spec.Resources))
	for _, ref := range changeJob.Spec.Resources {
		var obj *unstructured.Unstructured
		var updatedResourceStatus triggersv1alpha.ResourceReferenceStatus

		obj, err := r.fetchResource(ctx, ref)
		if err != nil {
			return ctrl.Result{}, err
		}

		updatedResourceStatus.APIVersion = obj.GetAPIVersion()
		updatedResourceStatus.Kind = obj.GetKind()
		updatedResourceStatus.Name = obj.GetName()
		updatedResourceStatus.Namespace = obj.GetNamespace()

		var updatedResourceStatusFieldList []triggersv1alpha.ResourceFieldHash
		for _, path := range ref.Fields {
			value, found, err := unstructured.NestedMap(obj.Object, path)
			if err != nil {
				return ctrl.Result{}, err
			}
			if !found {
				log.Error(nil, "field path %s not found in resource %s", path, ref.Name)
				continue
			}
			log.Info("field path %s found in resource %s, value %v", path, ref.Name, value)

			data, err := json.Marshal(value)
			if err != nil {
				return ctrl.Result{}, err
			}
			hash := ObjectHash(data)

			var updatedResourceStatusField triggersv1alpha.ResourceFieldHash
			updatedResourceStatusField.Field = path
			updatedResourceStatusField.LastHash = hash
			updatedResourceStatusFieldList = append(updatedResourceStatusFieldList, updatedResourceStatusField)
		}
		updatedResourceStatus.Fields = updatedResourceStatusFieldList
		updatedResourceStatusList = append(updatedResourceStatusList, updatedResourceStatus)
	}

	updatedResourceJson, err := json.Marshal(updatedResourceStatusList)
	if err != nil {
		return ctrl.Result{}, err
	}

	// lastResourceHashes, err := changeJob.Status.ResourceHashes
	lastResourceJson, err := json.Marshal(changeJob.Status.ResourceHashes)
	if err != nil {
		return ctrl.Result{}, err
	}

	if bytes.Equal(updatedResourceJson, lastResourceJson) {
		log.Info("No changes detected")
		return ctrl.Result{}, nil
	}

	diff, _ := diff.Diff(updatedResourceJson, lastResourceJson)
	if len(diff) < len(lastResourceJson) {
		switch changeJob.Spec.Condition {
		case triggersv1alpha.TriggerConditionAny:
			log.Info("TriggerCondition satisfied: Some resources are changed")
		case triggersv1alpha.TriggerConditionAll:
			log.Info("TriggerCondition not satisfied: Some resources are changed")
			return ctrl.Result{}, nil
		}
	} else {
		log.Info("TriggerCondition satisfied: All resources are changed")
	}

	log.Info("ChangeTriggeredJob %s triggered", changeJob.Name)
	log.Info("Creating Job")
	job := &batchv1.Job{}
	job.ObjectMeta = metav1.ObjectMeta{
		Name:        fmt.Sprintf("change-triggered-job-%s", changeJob.Name),
		Namespace:   changeJob.Namespace,
		Annotations: changeJob.Annotations,
		Labels:      changeJob.Labels,
	}
	job.Spec = changeJob.Spec.JobTemplate.Spec
	if err := controllerutil.SetControllerReference(&changeJob, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ChangeTriggeredJobReconciler) fetchResource(ctx context.Context, ref triggersv1alpha.ResourceReference) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(ref.APIVersion)
	u.SetKind(ref.Kind)

	key := client.ObjectKey{
		Name:      ref.Name,
		Namespace: ref.Namespace,
	}

	if err := r.Get(ctx, key, u); err != nil {
		return nil, err
	}

	return u, nil
}

// Calculate last change hash
func ObjectHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangeTriggeredJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&triggersv1alpha.ChangeTriggeredJob{}).
		Named("changetriggeredjob").
		Complete(r)
}
