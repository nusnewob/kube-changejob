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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
	"github.com/nusnewob/kube-changejob/internal/config"
)

// ChangeTriggeredJobReconciler reconciles a ChangeTriggeredJob object
type ChangeTriggeredJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config config.ControllerConfig
	Log    logr.Logger
}

const (
	DefaultLabel = "changejob.dev/owner"
)

var log = logf.Log.WithName("ChangeTriggeredJob")

// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=triggers.changejob.dev,resources=changetriggeredjobs/finalizers,verbs=update

// Manage triggered jobs
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// Watched resources
// +kubebuilder:rbac:groups="*",resources="*",verbs=get;watch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *ChangeTriggeredJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var changeJob triggersv1alpha.ChangeTriggeredJob
	if err := r.Get(ctx, req.NamespacedName, &changeJob); err != nil {
		log.Error(err, "unable to fetch ChangeTriggeredJob")
		return ctrl.Result{RequeueAfter: r.Config.PollInterval}, client.IgnoreNotFound(err)
	}

	// Validate JobTemplate
	if err := ValidateJobTemplate(ctx, r.Client, changeJob.Namespace, changeJob.Spec.JobTemplate); err != nil {
		log.Error(err, "invalid job template")
		// Don't requeue, as this is a configuration error
		return ctrl.Result{}, nil
	}

	changed, updatedStatuses, err := r.pollResources(ctx, &changeJob)
	if err != nil {
		log.Error(err, "unable to poll resources")
		return ctrl.Result{RequeueAfter: r.Config.PollInterval}, err
	}

	// Initialize resource hashes on first run or update status
	isFirstPoll := changeJob.Status.ResourceHashes == nil
	if isFirstPoll {
		log.V(1).Info("First poll of resource, establishing baseline", "name", changeJob.Name)
	}

	// Always update hashes
	changeJob.Status.ResourceHashes = updatedStatuses

	if changed {
		// Check if we should trigger (first time or after cooldown)
		if changeJob.Status.LastTriggeredTime == nil || time.Since(changeJob.Status.LastTriggeredTime.Time) > changeJob.Spec.Cooldown.Duration {
			log.Info("ChangeTriggeredJob triggered", "name", changeJob.Name)
			if _, err := r.triggerJob(ctx, &changeJob); err != nil {
				log.Error(err, "unable to trigger job")
				return ctrl.Result{RequeueAfter: r.Config.PollInterval}, err
			}
		}
	}

	// Always update status, including job history and latest job info
	if err := r.updateStatus(ctx, &changeJob); err != nil {
		log.Error(err, "unable to update status")
		return ctrl.Result{RequeueAfter: r.Config.PollInterval}, err
	}

	// Delete old jobs on every reconcile
	histories, err := r.listOwnedJobs(ctx, &changeJob)
	if err != nil {
		log.Error(err, "unable to get job histories")
		// Continue, as this is non-critical
	}

	if len(histories) > int(*changeJob.Spec.History) {
		log.Info("Cleaning up old jobs", "total", len(histories), "limit", *changeJob.Spec.History, "toDelete", len(histories)-int(*changeJob.Spec.History))
		for _, history := range histories[*changeJob.Spec.History:] {
			if err := r.Delete(ctx, &history); err != nil {
				log.Error(err, "Failed to delete old job", "job", history.Name)
			}
			log.V(1).Info("Deleted old job", "job", history.Name)
		}
	}

	// Always requeue to keep polling
	return ctrl.Result{RequeueAfter: r.Config.PollInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangeTriggeredJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index Jobs by their owner UID
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &batchv1.Job{}, "metadata.ownerReferences.uid", func(obj client.Object) []string {
		uids := make([]string, 0, len(obj.GetOwnerReferences()))
		for _, ref := range obj.GetOwnerReferences() {
			uids = append(uids, string(ref.UID))
		}
		return uids
	}); err != nil {
		return fmt.Errorf("failed to setup field indexer for Jobs: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&triggersv1alpha.ChangeTriggeredJob{}).
		Named("changetriggeredjob").
		Complete(r)
}
