package controller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
)

var _ = Describe("ChangeTriggeredJob Controller", func() {
	Context("When reconciling a ChangeTriggeredJob", func() {
		var (
			ctjName      string
			cmName       string
			ctjNamespace = "default"
			ctx          = context.Background()
		)

		BeforeEach(func() {
			// Use unique names for each test run
			ctjName = fmt.Sprintf("test-ctj-%d", time.Now().UnixNano())
			cmName = fmt.Sprintf("test-cm-%d", time.Now().UnixNano())
		})

		AfterEach(func() {
			// Force delete all jobs first
			jobList := &batchv1.JobList{}
			if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err == nil {
				for i := range jobList.Items {
					job := &jobList.Items[i]
					propagationPolicy := metav1.DeletePropagationBackground
					_ = k8sClient.Delete(ctx, job, &client.DeleteOptions{
						PropagationPolicy: &propagationPolicy,
					})
				}
			}

			// Clean up ChangeTriggeredJob
			ctj := &triggersv1alpha.ChangeTriggeredJob{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: ctjName, Namespace: ctjNamespace}, ctj); err == nil {
				_ = k8sClient.Delete(ctx, ctj)
			}

			// Clean up ConfigMaps
			cmList := &corev1.ConfigMapList{}
			if err := k8sClient.List(ctx, cmList, client.InNamespace(ctjNamespace)); err == nil {
				for i := range cmList.Items {
					cm := &cmList.Items[i]
					_ = k8sClient.Delete(ctx, cm)
				}
			}

			// Wait a moment for cleanup to complete
			time.Sleep(100 * time.Millisecond)
		})

		It("Should detect field changes in watched resources", func() {
			By("Creating a ChangeTriggeredJob")
			ctj := &triggersv1alpha.ChangeTriggeredJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctjName,
					Namespace: ctjNamespace,
				},
				Spec: triggersv1alpha.ChangeTriggeredJobSpec{
					Resources: []triggersv1alpha.ResourceReference{
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       cmName,
							Namespace:  ctjNamespace,
							Fields:     []string{"data.config"},
						},
					},
					Condition: triggersv1alpha.TriggerConditionAny,
					Cooldown:  metav1.Duration{Duration: 1 * time.Second},
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
				},
			}
			Expect(k8sClient.Create(ctx, ctj)).Should(Succeed())

			By("Creating a ConfigMap")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{
					"config": "initial-value",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			By("First reconciliation - initial state")
			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying first job was created")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(1))

			By("Updating the watched field")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "updated-value"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			By("Waiting for cooldown to pass")
			time.Sleep(2 * time.Second)

			By("Second reconciliation - after change")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying second job was created")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(2))
		})

		It("Should respect cooldown period", func() {
			By("Creating a ChangeTriggeredJob with 3s cooldown")
			ctj := &triggersv1alpha.ChangeTriggeredJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctjName,
					Namespace: ctjNamespace,
				},
				Spec: triggersv1alpha.ChangeTriggeredJobSpec{
					Resources: []triggersv1alpha.ResourceReference{
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       cmName,
							Namespace:  ctjNamespace,
							Fields:     []string{"data.config"},
						},
					},
					Condition: triggersv1alpha.TriggerConditionAny,
					Cooldown:  metav1.Duration{Duration: 3 * time.Second},
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
				},
			}
			Expect(k8sClient.Create(ctx, ctj)).Should(Succeed())

			By("Creating watched resource")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{"config": "value1"},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("First change triggers job")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying first job was created")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(1))

			By("Second change within cooldown does not trigger")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "value2"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying job count remains 1")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(HaveLen(1))

			By("Change after cooldown triggers new job")
			time.Sleep(4 * time.Second) // Wait for cooldown

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "value3"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying second job was created")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(2))
		})

		It("Should only monitor specified fields", func() {
			By("Creating a ChangeTriggeredJob that only watches data.config")
			ctj := &triggersv1alpha.ChangeTriggeredJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctjName,
					Namespace: ctjNamespace,
				},
				Spec: triggersv1alpha.ChangeTriggeredJobSpec{
					Resources: []triggersv1alpha.ResourceReference{
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       cmName,
							Namespace:  ctjNamespace,
							Fields:     []string{"data.config"}, // Only watch this field
						},
					},
					Condition: triggersv1alpha.TriggerConditionAny,
					Cooldown:  metav1.Duration{Duration: 1 * time.Second},
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
				},
			}
			Expect(k8sClient.Create(ctx, ctj)).Should(Succeed())

			By("Creating watched resource")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{
					"config":      "monitored",
					"other-field": "not-monitored",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Initial reconciliation establishes baseline")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying first job was created")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(1))

			By("Change to non-monitored field does not trigger job")
			time.Sleep(2 * time.Second) // Wait for cooldown
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["other-field"] = "changed-too"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying job count remains 1")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(HaveLen(1))

			By("Change to monitored field triggers second job")
			time.Sleep(2 * time.Second) // Wait for cooldown
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "changed"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying second job was created")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(2))
		})

		It("Should handle TriggerConditionAll with multiple resources", func() {
			cmName2 := fmt.Sprintf("test-cm2-%d", time.Now().UnixNano())

			By("Creating a ChangeTriggeredJob with TriggerConditionAll")
			ctj := &triggersv1alpha.ChangeTriggeredJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctjName,
					Namespace: ctjNamespace,
				},
				Spec: triggersv1alpha.ChangeTriggeredJobSpec{
					Resources: []triggersv1alpha.ResourceReference{
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       cmName,
							Namespace:  ctjNamespace,
							Fields:     []string{"data.config"},
						},
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       cmName2,
							Namespace:  ctjNamespace,
							Fields:     []string{"data.config"},
						},
					},
					Condition: triggersv1alpha.TriggerConditionAll,
					Cooldown:  metav1.Duration{Duration: 1 * time.Second},
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
				},
			}
			Expect(k8sClient.Create(ctx, ctj)).Should(Succeed())

			By("Creating first ConfigMap")
			cm1 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{"config": "value1"},
			}
			Expect(k8sClient.Create(ctx, cm1)).Should(Succeed())

			By("Creating second ConfigMap")
			cm2 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName2,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{"config": "value1"},
			}
			Expect(k8sClient.Create(ctx, cm2)).Should(Succeed())

			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Initial reconciliation establishes baseline")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying first job was created")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(1))

			Expect(k8sClient.Delete(ctx, cm1)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cm2)).To(Succeed())
		})

		It("Should handle non-existent watched resources gracefully", func() {
			By("Creating a ChangeTriggeredJob watching non-existent resource")
			ctj := &triggersv1alpha.ChangeTriggeredJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctjName,
					Namespace: ctjNamespace,
				},
				Spec: triggersv1alpha.ChangeTriggeredJobSpec{
					Resources: []triggersv1alpha.ResourceReference{
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "non-existent-cm",
							Namespace:  ctjNamespace,
							Fields:     []string{"data.config"},
						},
					},
					Condition: triggersv1alpha.TriggerConditionAny,
					Cooldown:  metav1.Duration{Duration: 1 * time.Second},
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
				},
			}
			Expect(k8sClient.Create(ctx, ctj)).Should(Succeed())

			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Reconciling with non-existent resource")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})

			By("Expecting an error due to missing resource")
			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("Should trigger even when no fields are specified (watches whole resource)", func() {
			By("Creating a ChangeTriggeredJob with no fields specified")
			ctj := &triggersv1alpha.ChangeTriggeredJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctjName,
					Namespace: ctjNamespace,
				},
				Spec: triggersv1alpha.ChangeTriggeredJobSpec{
					Resources: []triggersv1alpha.ResourceReference{
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       cmName,
							Namespace:  ctjNamespace,
							Fields:     []string{}, // No fields - watches whole resource
						},
					},
					Condition: triggersv1alpha.TriggerConditionAny,
					Cooldown:  metav1.Duration{Duration: 1 * time.Second},
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
				},
			}
			Expect(k8sClient.Create(ctx, ctj)).Should(Succeed())

			By("Creating a ConfigMap")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{"config": "value1"},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("First reconciliation with empty fields triggers initial job")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying a job was created on first reconciliation")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(1))
		})
	})
})
