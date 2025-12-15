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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
)

var _ = Describe("ChangeTriggeredJob Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		changetriggeredjob := &triggersv1alpha.ChangeTriggeredJob{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ChangeTriggeredJob")
			err := k8sClient.Get(ctx, typeNamespacedName, changetriggeredjob)
			if err != nil && errors.IsNotFound(err) {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-configmap",
						Namespace: "default",
					},
					Data: map[string]string{
						"key": "value",
					},
				}
				Expect(k8sClient.Create(ctx, cm)).To(Succeed())

				resource := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Condition: triggersv1alpha.TriggerConditionAny,
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										RestartPolicy: corev1.RestartPolicyNever,
										Containers: []corev1.Container{
											{
												Name:  "test-container",
												Image: "busybox",
												Command: []string{
													"echo",
													"hello world",
												},
											},
										},
									},
								},
							},
						},
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "test-configmap",
								Namespace:  "default",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &triggersv1alpha.ChangeTriggeredJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ChangeTriggeredJob")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
