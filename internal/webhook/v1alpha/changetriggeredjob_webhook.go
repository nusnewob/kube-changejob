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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
)

// nolint:unused
// log is for logging in this package.
var changetriggeredjoblog = logf.Log.WithName("changetriggeredjob-resource")

// SetupChangeTriggeredJobWebhookWithManager registers the webhook for ChangeTriggeredJob in the manager.
func SetupChangeTriggeredJobWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&triggersv1alpha.ChangeTriggeredJob{}).
		WithValidator(&ChangeTriggeredJobCustomValidator{}).
		WithDefaulter(&ChangeTriggeredJobCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-triggers-changejob-io-v1alpha-changetriggeredjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=triggers.changejob.io,resources=changetriggeredjobs,verbs=create;update,versions=v1alpha,name=mchangetriggeredjob-v1alpha.kb.io,admissionReviewVersions=v1

// ChangeTriggeredJobCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind ChangeTriggeredJob when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type ChangeTriggeredJobCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &ChangeTriggeredJobCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind ChangeTriggeredJob.
func (d *ChangeTriggeredJobCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	changetriggeredjob, ok := obj.(*triggersv1alpha.ChangeTriggeredJob)

	if !ok {
		return fmt.Errorf("expected an ChangeTriggeredJob object but got %T", obj)
	}
	changetriggeredjoblog.Info("Defaulting for ChangeTriggeredJob", "name", changetriggeredjob.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-triggers-changejob-io-v1alpha-changetriggeredjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=triggers.changejob.io,resources=changetriggeredjobs,verbs=create;update,versions=v1alpha,name=vchangetriggeredjob-v1alpha.kb.io,admissionReviewVersions=v1

// ChangeTriggeredJobCustomValidator struct is responsible for validating the ChangeTriggeredJob resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ChangeTriggeredJobCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &ChangeTriggeredJobCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type ChangeTriggeredJob.
func (v *ChangeTriggeredJobCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	changetriggeredjob, ok := obj.(*triggersv1alpha.ChangeTriggeredJob)
	if !ok {
		return nil, fmt.Errorf("expected a ChangeTriggeredJob object but got %T", obj)
	}
	changetriggeredjoblog.Info("Validation for ChangeTriggeredJob upon creation", "name", changetriggeredjob.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type ChangeTriggeredJob.
func (v *ChangeTriggeredJobCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	changetriggeredjob, ok := newObj.(*triggersv1alpha.ChangeTriggeredJob)
	if !ok {
		return nil, fmt.Errorf("expected a ChangeTriggeredJob object for the newObj but got %T", newObj)
	}
	changetriggeredjoblog.Info("Validation for ChangeTriggeredJob upon update", "name", changetriggeredjob.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type ChangeTriggeredJob.
func (v *ChangeTriggeredJobCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	changetriggeredjob, ok := obj.(*triggersv1alpha.ChangeTriggeredJob)
	if !ok {
		return nil, fmt.Errorf("expected a ChangeTriggeredJob object but got %T", obj)
	}
	changetriggeredjoblog.Info("Validation for ChangeTriggeredJob upon deletion", "name", changetriggeredjob.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
