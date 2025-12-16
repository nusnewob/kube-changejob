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
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
	resourcepoller "github.com/nusnewob/kube-changejob/internal/resourcepoller"
)

// ChangeTriggeredJobReconciler reconciles a ChangeTriggeredJob object
type ChangeTriggeredJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs/finalizers,verbs=update

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

	poller := resourcepoller.Poller{
		Client: r.Client,
	}

	changeCount := 0
	for _, ref := range changeJob.Spec.Resources {
		result, err := poller.Poll(ctx, ref)
		if err != nil {
			return ctrl.Result{}, err
		}

		last, found := getResourceRefStatus(changeJob.Status.ResourceHashes, ref)
		if !found {
			return ctrl.Result{}, err
		}

		nodiff := cmp.Equal(last, result)
		if !nodiff {
			changeCount++
		}
	}

	if changeCount > 0 {
		switch changeJob.Spec.Condition {
		case triggersv1alpha.TriggerConditionAny:
			log.Info("TriggerCondition satisfied: Some resources are changed")
		case triggersv1alpha.TriggerConditionAll:
			if changeCount < len(changeJob.Spec.Resources) {
				log.Info("TriggerCondition not satisfied: Some resources are changed")
				return ctrl.Result{}, nil
			}
			log.Info("TriggerCondition satisfied: All resources are changed")
		}
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

// Get ResourceReference Hash
func getResourceRefStatus(statuses []triggersv1alpha.ResourceReferenceStatus, ref triggersv1alpha.ResourceReference) (triggersv1alpha.ResourceReferenceStatus, bool) {
	for _, status := range statuses {
		if status.APIVersion == ref.APIVersion &&
			status.Kind == ref.Kind &&
			status.Name == ref.Name &&
			status.Namespace == ref.Namespace {
			return status, true
		}
	}
	return triggersv1alpha.ResourceReferenceStatus{}, false
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangeTriggeredJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&triggersv1alpha.ChangeTriggeredJob{}).
		Named("changetriggeredjob").
		Complete(r)
}
