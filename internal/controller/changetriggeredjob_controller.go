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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
)

// ChangeTriggeredJobReconciler reconciles a ChangeTriggeredJob object
type ChangeTriggeredJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	PollInterval = 60 * time.Second
	DefaultLabel = "changejob.dev/owner"
)

// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs/finalizers,verbs=update

// Manage triggered jobs
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// Watched resources
// +kubebuilder:rbac:groups="*",resources="*",verbs=get;list;watch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *ChangeTriggeredJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ChangeTriggeredJob", req.NamespacedName)

	var changeJob triggersv1alpha.ChangeTriggeredJob
	if err := r.Get(ctx, req.NamespacedName, &changeJob); err != nil {
		log.Error(err, "unable to fetch ChangeTriggeredJob")
		return ctrl.Result{RequeueAfter: PollInterval}, client.IgnoreNotFound(err)
	}

	changed, updatedStatuses, err := r.pollResources(ctx, &changeJob)
	if err != nil {
		return ctrl.Result{RequeueAfter: PollInterval}, err
	}

	// Initialize resource hashes on first run (no job triggered)
	if changeJob.Status.ResourceHashes == nil {
		changeJob.Status.ResourceHashes = updatedStatuses
		if err := r.Status().Update(ctx, &changeJob); err != nil {
			return ctrl.Result{RequeueAfter: PollInterval}, err
		}
		return ctrl.Result{RequeueAfter: PollInterval}, nil
	}

	// Update LastJobStatus status
	histories, err := r.listOwnedJobs(ctx, &changeJob)
	if err != nil {
		return ctrl.Result{RequeueAfter: PollInterval}, err
	}
	if len(histories) != 0 {
		if histories[0].Status.Failed != 0 {
			changeJob.Status.LastJobStatus = triggersv1alpha.JobStateFailed
		} else if histories[0].Status.Active != 0 {
			changeJob.Status.LastJobStatus = triggersv1alpha.JobStateActive
		} else if histories[0].Status.Succeeded != 0 {
			changeJob.Status.LastJobStatus = triggersv1alpha.JobStateSucceeded
		}
		if err := r.Status().Update(ctx, &changeJob); err != nil {
			return ctrl.Result{RequeueAfter: PollInterval}, err
		}
	}

	// Delete job history > changeJob.Spec.History
	if len(histories) > int(changeJob.Spec.History) {
		log.Info("Cleaning up old jobs", "total", len(histories), "limit", changeJob.Spec.History, "toDelete", len(histories)-int(changeJob.Spec.History))
		for _, history := range histories[changeJob.Spec.History:] {
			if err := r.Delete(ctx, &history); err != nil {
				log.Error(err, "Failed to delete old job", "job", history.Name)
				return ctrl.Result{RequeueAfter: PollInterval}, err
			}
			log.Info("Deleted old job", "job", history.Name)
		}
	}

	if !changed {
		return ctrl.Result{RequeueAfter: PollInterval}, nil
	}

	// Check if we should trigger (first time or after cooldown)
	if changeJob.Status.LastTriggeredTime == nil ||
		time.Since(changeJob.Status.LastTriggeredTime.Time) >= changeJob.Spec.Cooldown.Duration {
		log.Info("ChangeTriggeredJob triggered", "name", changeJob.Name)
		log.Info("Creating Job")
		job, err := r.triggerJob(ctx, &changeJob)
		if err != nil {
			return ctrl.Result{RequeueAfter: PollInterval}, err
		}

		if err := r.updateStatus(ctx, &changeJob, job, updatedStatuses); err != nil {
			return ctrl.Result{RequeueAfter: PollInterval}, err
		}
	}

	// Always requeue to keep polling
	return ctrl.Result{RequeueAfter: PollInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangeTriggeredJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&triggersv1alpha.ChangeTriggeredJob{}).
		Named("changetriggeredjob").
		Complete(r)
}
