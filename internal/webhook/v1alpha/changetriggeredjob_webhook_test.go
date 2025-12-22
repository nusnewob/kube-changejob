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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("ChangeTriggeredJob Webhook", func() {
	var (
		obj       *triggersv1alpha.ChangeTriggeredJob
		oldObj    *triggersv1alpha.ChangeTriggeredJob
		validator ChangeTriggeredJobCustomValidator
		defaulter ChangeTriggeredJobCustomDefaulter
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		obj = &triggersv1alpha.ChangeTriggeredJob{}
		oldObj = &triggersv1alpha.ChangeTriggeredJob{}

		// Create RESTMapper for validation
		dc, err := discovery.NewDiscoveryClientForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		groupResources, err := restmapper.GetAPIGroupResources(dc)
		Expect(err).NotTo(HaveOccurred())
		mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

		validator = ChangeTriggeredJobCustomValidator{Mapper: mapper}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = ChangeTriggeredJobCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// Cleanup after tests
	})

	Context("When creating ChangeTriggeredJob under Defaulting Webhook", func() {
		It("Should apply default cooldown when not specified", func() {
			By("Creating a ChangeTriggeredJob without cooldown")
			obj.Spec.Cooldown = metav1.Duration{Duration: 0}
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling the Default method")
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying default cooldown is applied")
			Expect(obj.Spec.Cooldown.Duration).To(Equal(DefaultValues.DefaultCooldown))
		})

		It("Should apply default condition when not specified", func() {
			By("Creating a ChangeTriggeredJob without condition")
			obj.Spec.Condition = ""
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling the Default method")
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying default condition is applied")
			Expect(obj.Spec.Condition).To(Equal(DefaultValues.DefaultCondition))
		})

		It("Should add changed-at annotation", func() {
			By("Creating a ChangeTriggeredJob without annotations")
			obj.Annotations = nil
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling the Default method")
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying changed-at annotation is added")
			Expect(obj.Annotations).NotTo(BeNil())
			Expect(obj.Annotations[DefaultValues.ChangedAtAnnotationKey]).NotTo(BeEmpty())
		})

		It("Should not override existing cooldown and condition", func() {
			By("Creating a ChangeTriggeredJob with custom values")
			customCooldown := 120 * time.Second
			obj.Spec.Cooldown = metav1.Duration{Duration: customCooldown}
			obj.Spec.Condition = triggersv1alpha.TriggerConditionAll
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling the Default method")
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying custom values are preserved")
			Expect(obj.Spec.Cooldown.Duration).To(Equal(customCooldown))
			Expect(obj.Spec.Condition).To(Equal(triggersv1alpha.TriggerConditionAll))
		})
	})

	Context("When creating or updating ChangeTriggeredJob under Validating Webhook", func() {
		It("Should deny creation if no resources are specified", func() {
			By("Creating a ChangeTriggeredJob with no resources")
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting validation error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one resource must be specified"))
		})

		It("Should admit creation if at least one resource is specified", func() {
			By("Creating a ChangeTriggeredJob with valid resources")
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
					Fields:     []string{"data.config"},
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should admit creation with multiple resources", func() {
			By("Creating a ChangeTriggeredJob with multiple resources")
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm1",
					Namespace:  "default",
				},
				{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       "test-secret",
					Namespace:  "default",
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should deny update if resources are removed", func() {
			By("Creating old object with resources")
			oldObj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Creating new object without resources")
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{}

			By("Calling ValidateUpdate")
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)

			By("Expecting validation error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one resource must be specified"))
		})

		It("Should admit update with valid resources", func() {
			By("Creating old object with resources")
			oldObj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
				},
			}

			By("Creating new object with updated resources")
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm-updated",
					Namespace:  "default",
				},
			}

			By("Calling ValidateUpdate")
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should admit deletion without errors", func() {
			By("Creating object to delete")
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling ValidateDelete")
			_, err := validator.ValidateDelete(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should deny creation if history is less than 1", func() {
			By("Creating a ChangeTriggeredJob with history = 0")
			obj.Spec.History = 0
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should admit creation if history is valid", func() {
			By("Creating a ChangeTriggeredJob with history = 3")
			obj.Spec.History = 3
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
