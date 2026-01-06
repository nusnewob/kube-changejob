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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
	"k8s.io/utils/ptr"

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
		obj = &triggersv1alpha.ChangeTriggeredJob{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
			},
		}
		oldObj = &triggersv1alpha.ChangeTriggeredJob{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
			},
		}

		// Create RESTMapper for validation
		dc, err := discovery.NewDiscoveryClientForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		groupResources, err := restmapper.GetAPIGroupResources(dc)
		Expect(err).NotTo(HaveOccurred())
		mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

		validator = ChangeTriggeredJobCustomValidator{Mapper: mapper, Client: k8sClient}
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
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Condition = nil
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			Expect(obj.Spec.Condition).To(HaveValue(Equal(DefaultValues.DefaultCondition)))
		})

		It("Should deny creation when condition is not 'All' or 'Any'", func() {
			By("Creating a ChangeTriggeredJob without condition")
			obj.Spec.Condition = ptr.To(triggersv1alpha.TriggerCondition("Invalid"))
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})

		It("Should add changed-at annotation", func() {
			By("Creating a ChangeTriggeredJob without annotations")
			obj.Annotations = nil
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Cooldown = &metav1.Duration{Duration: customCooldown}
			obj.Spec.Condition = ptr.To(triggersv1alpha.TriggerConditionAll)
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			Expect(obj.Spec.Condition).To(HaveValue(Equal(triggersv1alpha.TriggerConditionAll)))
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
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			oldObj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			oldObj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			oldObj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
				},
			}

			By("Creating new object with updated resources")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.History = ptr.To(int32(0))
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			Expect(err).To(HaveOccurred())
		})

		It("Should admit creation if history is valid", func() {
			By("Creating a ChangeTriggeredJob with history = 3")
			obj.Spec.History = ptr.To(int32(3))
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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

	Context("Additional validation tests for edge cases", func() {
		It("Should apply default history when not specified", func() {
			By("Creating a ChangeTriggeredJob without history")
			obj.Spec.History = nil
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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

			By("Verifying default history is applied")
			Expect(obj.Spec.History).To(HaveValue(Equal(DefaultValues.DefaultHistory)))
		})

		It("Should handle resources with fields array", func() {
			By("Creating a ChangeTriggeredJob with specific fields")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
					Fields:     []string{"data.key1", "data.key2"},
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should handle resources with wildcard field", func() {
			By("Creating a ChangeTriggeredJob with wildcard field")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
					Fields:     []string{"*"},
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should preserve existing annotations when applying defaults", func() {
			By("Creating a ChangeTriggeredJob with existing annotations")
			obj.Annotations = map[string]string{
				"custom-annotation": "custom-value",
			}
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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

			By("Verifying existing annotations are preserved")
			Expect(obj.Annotations["custom-annotation"]).To(Equal("custom-value"))
			Expect(obj.Annotations[DefaultValues.ChangedAtAnnotationKey]).NotTo(BeEmpty())
		})

		It("Should handle cluster-scoped resources without namespace", func() {
			By("Creating a ChangeTriggeredJob watching a cluster-scoped resource")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "Namespace",
					Name:       "test-namespace",
					// No namespace for cluster-scoped resources
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should validate invalid APIVersion format", func() {
			By("Creating a ChangeTriggeredJob with invalid APIVersion")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "invalid/version/format",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting validation error")
			Expect(err).To(HaveOccurred())
		})

		It("Should validate unknown Kind", func() {
			By("Creating a ChangeTriggeredJob with unknown Kind")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "NonExistentKind",
					Name:       "test-resource",
					Namespace:  "default",
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting validation error for unknown kind")
			Expect(err).To(HaveOccurred())
		})

		It("Should update changed-at annotation on each default call", func() {
			By("Creating a ChangeTriggeredJob")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling Default method first time")
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			firstTimestamp := obj.Annotations[DefaultValues.ChangedAtAnnotationKey]

			By("Waiting a moment")
			time.Sleep(1100 * time.Millisecond)

			By("Calling Default method second time")
			err = defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			secondTimestamp := obj.Annotations[DefaultValues.ChangedAtAnnotationKey]

			By("Verifying timestamp was updated")
			Expect(secondTimestamp).NotTo(Equal(firstTimestamp))
		})

		It("Should validate TriggerCondition 'All'", func() {
			By("Creating a ChangeTriggeredJob with condition 'All'")
			obj.Spec.Condition = ptr.To(triggersv1alpha.TriggerConditionAll)
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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

		It("Should validate TriggerCondition 'Any'", func() {
			By("Creating a ChangeTriggeredJob with condition 'Any'")
			obj.Spec.Condition = ptr.To(triggersv1alpha.TriggerConditionAny)
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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

		It("Should deny negative history values", func() {
			By("Creating a ChangeTriggeredJob with negative history")
			obj.Spec.History = ptr.To(int32(-1))
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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

			By("Expecting validation error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be >= 1"))
		})

		It("Should deny creation with invalid jobTemplate - missing RestartPolicy", func() {
			By("Creating a ChangeTriggeredJob with invalid jobTemplate")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "busybox:latest",
								},
							},
							// Missing RestartPolicy which should be Never or OnFailure for Jobs
						},
					},
				},
			}
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

			By("Expecting validation error for invalid jobTemplate")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobTemplate"))
		})

		It("Should deny creation with invalid jobTemplate - missing containers", func() {
			By("Creating a ChangeTriggeredJob with no containers")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			}
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

			By("Expecting validation error for invalid jobTemplate")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobTemplate"))
		})

		It("Should admit creation with valid jobTemplate using OnFailure restart policy", func() {
			By("Creating a ChangeTriggeredJob with OnFailure restart policy")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "busybox:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			}
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

		It("Should admit creation with jobTemplate containing multiple containers", func() {
			By("Creating a ChangeTriggeredJob with multiple containers")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "main",
									Image: "busybox:latest",
								},
								{
									Name:  "sidecar",
									Image: "nginx:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			}
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

		It("Should handle defaulter with wrong object type", func() {
			By("Calling Default with wrong type")
			wrongObj := &corev1.ConfigMap{}
			err := defaulter.Default(ctx, wrongObj)

			By("Expecting type error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected an ChangeTriggeredJob object"))
		})

		It("Should handle validator ValidateCreate with wrong object type", func() {
			By("Calling ValidateCreate with wrong type")
			wrongObj := &corev1.ConfigMap{}
			_, err := validator.ValidateCreate(ctx, wrongObj)

			By("Expecting type error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a ChangeTriggeredJob object"))
		})

		It("Should handle validator ValidateUpdate with wrong object type", func() {
			By("Calling ValidateUpdate with wrong type")
			wrongObj := &corev1.ConfigMap{}
			_, err := validator.ValidateUpdate(ctx, oldObj, wrongObj)

			By("Expecting type error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a ChangeTriggeredJob object"))
		})

		It("Should handle validator ValidateDelete with wrong object type", func() {
			By("Calling ValidateDelete with wrong type")
			wrongObj := &corev1.ConfigMap{}
			_, err := validator.ValidateDelete(ctx, wrongObj)

			By("Expecting type error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a ChangeTriggeredJob object"))
		})

		It("Should deny creation with negative cooldown", func() {
			By("Creating a ChangeTriggeredJob with negative cooldown")
			obj.Spec.Cooldown = &metav1.Duration{Duration: -1 * time.Second}
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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

			By("Expecting validation error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be >= 0"))
		})

		It("Should admit creation with zero cooldown", func() {
			By("Creating a ChangeTriggeredJob with zero cooldown")
			obj.Spec.Cooldown = &metav1.Duration{Duration: 0}
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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

		It("Should validate resources with empty fields array", func() {
			By("Creating a ChangeTriggeredJob with empty fields array")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
					Namespace:  "default",
					Fields:     []string{}, // Empty fields array
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should use custom defaulter values", func() {
			By("Creating a ChangeTriggeredJob and verifying default defaulter values first")
			testObj := &triggersv1alpha.ChangeTriggeredJob{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
			}
			testObj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			testObj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling Default with standard defaulter")
			err := defaulter.Default(ctx, testObj)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying standard defaults are applied")
			Expect(testObj.Spec.Cooldown.Duration).To(Equal(DefaultValues.DefaultCooldown))
			Expect(testObj.Spec.Condition).To(HaveValue(Equal(DefaultValues.DefaultCondition)))
			Expect(testObj.Spec.History).To(HaveValue(Equal(DefaultValues.DefaultHistory)))
			Expect(testObj.Annotations[DefaultValues.ChangedAtAnnotationKey]).NotTo(BeEmpty())
		})

		It("Should validate Secret resource type", func() {
			By("Creating a ChangeTriggeredJob watching a Secret")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       "test-secret",
					Namespace:  "default",
					Fields:     []string{"data.password"},
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should validate Pod resource type", func() {
			By("Creating a ChangeTriggeredJob watching a Pod")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "test-pod",
					Namespace:  "default",
					Fields:     []string{"status.phase"},
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should validate Service resource type", func() {
			By("Creating a ChangeTriggeredJob watching a Service")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       "test-service",
					Namespace:  "default",
					Fields:     []string{"spec.type"},
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should validate Deployment resource type", func() {
			By("Creating a ChangeTriggeredJob watching a Deployment")
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "test-deployment",
					Namespace:  "default",
					Fields:     []string{"spec.replicas"},
				},
			}

			By("Calling ValidateCreate")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Expecting no validation error")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should apply defaults correctly when all fields are nil", func() {
			By("Creating a ChangeTriggeredJob with all default fields nil")
			obj.Spec.Cooldown = nil
			obj.Spec.Condition = nil
			obj.Spec.History = nil
			obj.Annotations = nil
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
			obj.Spec.Resources = []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "test-cm",
				},
			}

			By("Calling Default")
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying all defaults are applied")
			Expect(obj.Spec.Cooldown).NotTo(BeNil())
			Expect(obj.Spec.Condition).NotTo(BeNil())
			Expect(obj.Spec.History).NotTo(BeNil())
			Expect(obj.Annotations).NotTo(BeEmpty())
			Expect(obj.Annotations[DefaultValues.ChangedAtAnnotationKey]).NotTo(BeEmpty())
		})

		It("Should accept large history values", func() {
			By("Creating a ChangeTriggeredJob with large history")
			obj.Spec.History = ptr.To(int32(1000))
			obj.Spec.JobTemplate = batchv1.JobTemplateSpec{
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
