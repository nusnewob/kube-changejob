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
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
	controller "github.com/nusnewob/kube-changejob/internal/controller"
)

// nolint:unused
// log is for logging in this package.
var changetriggeredjoblog = logf.Log.WithName("changetriggeredjob-resource")

// SetupChangeTriggeredJobWebhookWithManager registers the webhook for ChangeTriggeredJob in the manager.
func SetupChangeTriggeredJobWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&triggersv1alpha.ChangeTriggeredJob{}).
		WithValidator(&ChangeTriggeredJobCustomValidator{
			Mapper: mgr.GetRESTMapper(),
		}).
		WithDefaulter(&ChangeTriggeredJobCustomDefaulter{
			DefaultCooldown:        DefaultValues.DefaultCooldown,
			DefaultCondition:       DefaultValues.DefaultCondition,
			DefaultHistory:         DefaultValues.DefaultHistory,
			ChangedAtAnnotationKey: DefaultValues.ChangedAtAnnotationKey,
		}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-triggers-changejob-dev-v1alpha-changetriggeredjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=triggers.changejob.dev,resources=changetriggeredjobs,verbs=create;update,versions=v1alpha,name=mchangetriggeredjob-v1alpha.kb.io,admissionReviewVersions=v1

// ChangeTriggeredJobCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind ChangeTriggeredJob when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type ChangeTriggeredJobCustomDefaulter struct {
	DefaultCooldown        time.Duration
	DefaultCondition       triggersv1alpha.TriggerCondition
	DefaultHistory         int32
	ChangedAtAnnotationKey string
}

var DefaultValues = ChangeTriggeredJobCustomDefaulter{
	DefaultCooldown:        60 * time.Second,
	DefaultCondition:       triggersv1alpha.TriggerConditionAny,
	DefaultHistory:         5,
	ChangedAtAnnotationKey: "changetriggeredjobs.triggers.changejob.dev/changed-at",
}

var _ webhook.CustomDefaulter = &ChangeTriggeredJobCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind ChangeTriggeredJob.
func (d *ChangeTriggeredJobCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	changetriggeredjob, ok := obj.(*triggersv1alpha.ChangeTriggeredJob)

	if !ok {
		return fmt.Errorf("expected an ChangeTriggeredJob object but got %T", obj)
	}
	changetriggeredjoblog.Info("Defaulting for ChangeTriggeredJob", "name", changetriggeredjob.GetName())

	// Optional: default cooldown if unset
	if changetriggeredjob.Spec.Cooldown == nil {
		changetriggeredjob.Spec.Cooldown = &metav1.Duration{Duration: DefaultValues.DefaultCooldown}
	}

	// Optional: default trigger condition if unset
	if changetriggeredjob.Spec.Condition == nil {
		changetriggeredjob.Spec.Condition = &DefaultValues.DefaultCondition
	}

	// Optional: default history if unset
	if changetriggeredjob.Spec.History == nil {
		changetriggeredjob.Spec.History = &DefaultValues.DefaultHistory
	}

	if changetriggeredjob.Annotations == nil {
		changetriggeredjob.Annotations = make(map[string]string)
	}
	changetriggeredjob.Annotations[DefaultValues.ChangedAtAnnotationKey] = time.Now().UTC().Format(time.RFC3339)

	return nil
}

// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-triggers-changejob-dev-v1alpha-changetriggeredjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=triggers.changejob.dev,resources=changetriggeredjobs,verbs=create;update,versions=v1alpha,name=vchangetriggeredjob-v1alpha.kb.io,admissionReviewVersions=v1

// ChangeTriggeredJobCustomValidator struct is responsible for validating the ChangeTriggeredJob resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ChangeTriggeredJobCustomValidator struct {
	Triggers []triggersv1alpha.ResourceReference
	Mapper   meta.RESTMapper
}

var _ webhook.CustomValidator = &ChangeTriggeredJobCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type ChangeTriggeredJob.
func (v *ChangeTriggeredJobCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	changetriggeredjob, ok := obj.(*triggersv1alpha.ChangeTriggeredJob)
	if !ok {
		return nil, fmt.Errorf("expected a ChangeTriggeredJob object but got %T", obj)
	}
	changetriggeredjoblog.Info("Validation for ChangeTriggeredJob upon creation", "name", changetriggeredjob.GetName())

	if len(changetriggeredjob.Spec.Resources) == 0 {
		return nil, fmt.Errorf("at least one resource must be specified")
	}

	for i, ref := range changetriggeredjob.Spec.Resources {
		_, err := controller.ValidateGVK(ctx, v.Mapper, ref.APIVersion, ref.Kind, ref.Namespace)
		if err != nil {
			return nil, field.Invalid(
				field.NewPath("spec", "resources").Index(i),
				fmt.Sprintf("%s/%s", ref.APIVersion, ref.Kind),
				err.Error(),
			)
		}
	}

	validCondition := map[triggersv1alpha.TriggerCondition]struct{}{
		triggersv1alpha.TriggerConditionAll: {},
		triggersv1alpha.TriggerConditionAny: {},
	}

	if changetriggeredjob.Spec.Condition != nil {
		if _, ok := validCondition[*changetriggeredjob.Spec.Condition]; !ok {
			return nil, field.Invalid(
				field.NewPath("spec").Child("condition"),
				*changetriggeredjob.Spec.Condition,
				"must be 'All' or 'Any'",
			)
		}
	}

	if changetriggeredjob.Spec.History != nil && *changetriggeredjob.Spec.History < 1 {
		return nil, field.Invalid(
			field.NewPath("spec").Child("history"),
			*changetriggeredjob.Spec.History,
			"must be >= 1",
		)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type ChangeTriggeredJob.
func (v *ChangeTriggeredJobCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.ValidateCreate(ctx, newObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type ChangeTriggeredJob.
func (v *ChangeTriggeredJobCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	changetriggeredjob, ok := obj.(*triggersv1alpha.ChangeTriggeredJob)
	if !ok {
		return nil, fmt.Errorf("expected a ChangeTriggeredJob object but got %T", obj)
	}
	changetriggeredjoblog.Info("Validation for ChangeTriggeredJob upon deletion", "name", changetriggeredjob.GetName())

	return nil, nil
}
