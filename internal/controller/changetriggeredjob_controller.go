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
	"time"

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

const (
	PollInterval = 60 * time.Second
)

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

	now := time.Now()

	changed, updatedStatuses, err := r.pollResources(ctx, &changeJob)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !changed {
		return ctrl.Result{}, err
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

	changeJob.Status.LastTriggeredTime = &metav1.Time{Time: now}
	changeJob.Status.LastJobName = fmt.Sprintf("change-triggered-job-%s", changeJob.Name)
	// changeJob.Status.LastJobStatus = ""
	changeJob.Status.ResourceHashes = updatedStatuses
	if err := r.Status().Update(ctx, &changeJob); err != nil {
		return ctrl.Result{}, err
	}

	// Always requeue to keep polling
	return ctrl.Result{
		RequeueAfter: PollInterval,
	}, nil
}

// PollResources polls the resources referenced by the given ChangeTriggeredJob.
func (r *ChangeTriggeredJobReconciler) pollResources(ctx context.Context, changeJob *triggersv1alpha.ChangeTriggeredJob) (bool, []triggersv1alpha.ResourceReferenceStatus, error) {
	poller := resourcepoller.Poller{Client: r.Client}

	existing := resourcepoller.IndexResourceStatuses(changeJob.Status.ResourceHashes)
	updated := make([]triggersv1alpha.ResourceReferenceStatus, 0, len(changeJob.Spec.Resources))

	changed := false
	changeCount := 0

	for _, ref := range changeJob.Spec.Resources {
		result, err := poller.Poll(ctx, ref)
		if err != nil {
			return false, nil, err
		}

		key := resourcepoller.ResourceKey(ref.APIVersion, ref.Kind, ref.Namespace, ref.Name)

		last, found := existing[key]
		if !found || !cmp.Equal(last, result) {
			changeCount++
		}

		updated = append(updated, result)
	}

	if changeCount > 0 {
		switch changeJob.Spec.Condition {
		case triggersv1alpha.TriggerConditionAny:
			changed = true
		case triggersv1alpha.TriggerConditionAll:
			if changeCount < len(changeJob.Spec.Resources) {
				changed = false
			} else {
				changed = true
			}
		}
	}

	return changed, updated, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangeTriggeredJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&triggersv1alpha.ChangeTriggeredJob{}).
		Named("changetriggeredjob").
		Complete(r)
}
