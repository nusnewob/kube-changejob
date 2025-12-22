package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
)

var _ = Describe("ChangeTriggeredJob Controller", func() {
	const (
		testValue2       = "value2"
		testValueChanged = "changed"
	)

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

			By("First reconciliation - establishes baseline, no job triggered")
			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Log:    logr.New(zap.New(zap.UseDevMode(true)).GetSink()),
			}

			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no job was created on initial reconciliation")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(BeEmpty())

			By("Updating the watched field")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "updated-value"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			By("Second reconciliation - first change triggers job")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
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

			By("Updating the watched field again")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "another-update"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			By("Waiting for cooldown to pass")
			time.Sleep(2 * time.Second)

			By("Third reconciliation - after cooldown triggers second job")
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
				Log:    logr.New(zap.New(zap.UseDevMode(true)).GetSink()),
			}

			By("Initial reconciliation establishes baseline")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no job created on initial reconciliation")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(BeEmpty())

			By("First change triggers job")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "value3"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
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
			cm.Data["config"] = testValue2
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying job count remains 1")
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(HaveLen(1))

			By("Third change after cooldown triggers new job")
			time.Sleep(4 * time.Second) // Wait for cooldown

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "value4"
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
				Log:    logr.New(zap.New(zap.UseDevMode(true)).GetSink()),
			}

			By("Initial reconciliation establishes baseline")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no job created on initial reconciliation")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(BeEmpty())

			By("Change to monitored field triggers first job")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "first-change"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
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
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(HaveLen(1))

			By("Change to monitored field triggers second job")
			time.Sleep(2 * time.Second) // Wait for cooldown
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = testValueChanged
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
				Log:    logr.New(zap.New(zap.UseDevMode(true)).GetSink()),
			}

			By("Initial reconciliation establishes baseline")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no job created on initial reconciliation")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(BeEmpty())

			By("Changing both ConfigMaps")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm1)).Should(Succeed())
			cm1.Data["config"] = "changed1"
			Expect(k8sClient.Update(ctx, cm1)).Should(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName2, Namespace: ctjNamespace}, cm2)).Should(Succeed())
			cm2.Data["config"] = "changed2"
			Expect(k8sClient.Update(ctx, cm2)).Should(Succeed())

			By("Second reconciliation with all resources changed")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying job was created when all resources changed")
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
				Log:    logr.New(zap.New(zap.UseDevMode(true)).GetSink()),
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

			By("First reconciliation establishes baseline")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no job created on initial reconciliation")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(BeEmpty())

			By("Updating ConfigMap triggers job")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "value2"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying a job was created after change")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(1))
		})

		It("Should handle TriggerConditionAll with partial changes (no trigger)", func() {
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

			By("Creating both ConfigMaps")
			cm1 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{"config": "value1"},
			}
			Expect(k8sClient.Create(ctx, cm1)).Should(Succeed())

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

			By("Changing only one ConfigMap (not all)")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm1)).Should(Succeed())
			cm1.Data["config"] = "changed1"
			Expect(k8sClient.Update(ctx, cm1)).Should(Succeed())

			By("Second reconciliation with partial change")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no job was created (TriggerConditionAll requires all resources to change)")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(BeEmpty())

			Expect(k8sClient.Delete(ctx, cm1)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cm2)).To(Succeed())
		})

		It("Should handle zero cooldown period", func() {
			By("Creating a ChangeTriggeredJob with zero cooldown")
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
					Cooldown:  metav1.Duration{Duration: 0},
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

			By("Initial reconciliation establishes baseline")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("First change triggers job")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = testValue2
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
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

			By("Second change immediately triggers another job (zero cooldown)")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "value3"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying second job was created immediately")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(2))
		})

		It("Should handle nested field paths", func() {
			By("Creating a ChangeTriggeredJob watching nested fields")
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
							Fields:     []string{"data.nested.value"},
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

			By("Creating ConfigMap with nested structure in data")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{
					"nested.value": "initial",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Initial reconciliation")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Changing the nested field")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["nested.value"] = "updated"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			By("Second reconciliation after change")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying job was created for nested field change")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(1))
		})

		It("Should propagate labels and annotations to created jobs", func() {
			By("Creating a ChangeTriggeredJob with labels and annotations")
			ctj := &triggersv1alpha.ChangeTriggeredJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctjName,
					Namespace: ctjNamespace,
					Labels: map[string]string{
						"app":         "test-app",
						"environment": "test",
					},
					Annotations: map[string]string{
						"description": "test job",
						"owner":       "test-team",
					},
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

			By("Initial reconciliation")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Triggering a change")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = testValue2
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying job has correct labels and annotations")
			Eventually(func() bool {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return false
				}
				if len(jobList.Items) != 1 {
					return false
				}
				job := jobList.Items[0]
				return job.Labels["app"] == "test-app" &&
					job.Labels["environment"] == "test" &&
					job.Annotations["description"] == "test job" &&
					job.Annotations["owner"] == "test-team"
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("Should not trigger on changes to unwatched fields when specific fields are monitored", func() {
			By("Creating a ChangeTriggeredJob watching only data.watched")
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
							Fields:     []string{"data.watched"},
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

			By("Creating watched resource with multiple fields")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: ctjNamespace,
				},
				Data: map[string]string{
					"watched":   "initial",
					"unwatched": "initial",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Initial reconciliation")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Changing only unwatched field")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["unwatched"] = "changed"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			By("Reconciling after unwatched field change")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no job was created")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			Expect(jobList.Items).To(BeEmpty())

			By("Now changing the watched field")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["watched"] = testValueChanged
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			By("Reconciling after watched field change")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying job was created")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace)); err != nil {
					return 0
				}
				return len(jobList.Items)
			}, time.Second*5, time.Millisecond*500).Should(Equal(1))
		})

		It("Should clean up old jobs when history limit is exceeded", func() {
			By("Creating a ChangeTriggeredJob with history limit of 1")
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
					History:   1,
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

			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Initial reconciliation establishes baseline")
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Creating 2 jobs by triggering changes")
			for i := 1; i <= 2; i++ {
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ctjNamespace}, cm)).Should(Succeed())
				cm.Data["config"] = fmt.Sprintf("value-%d", i)
				Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
				})
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(1200 * time.Millisecond)
			}

			By("Verifying 2 jobs were created before cleanup")
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace), client.MatchingLabels{"changejob.dev/owner": ctjName})).Should(Succeed())
			fmt.Printf("Jobs before cleanup: %d\n", len(jobList.Items))
			for _, job := range jobList.Items {
				fmt.Printf("  - Job: %s, Created: %v\n", job.Name, job.CreationTimestamp)
			}
			Expect(jobList.Items).To(HaveLen(2))

			By("Triggering additional reconciliation to cleanup old jobs")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: ctjName, Namespace: ctjNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that only 1 jobs remain (history limit)")
			Eventually(func() int {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace), client.MatchingLabels{"changejob.dev/owner": ctjName}); err != nil {
					return -1
				}
				fmt.Printf("Jobs count in Eventually: %d\n", len(jobList.Items))
				return len(jobList.Items)
			}, time.Second*70, time.Millisecond*1000).Should(Equal(2))

			By("Verifying the oldest jobs were deleted")
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace), client.MatchingLabels{"changejob.dev/owner": ctjName})).Should(Succeed())
			Expect(jobList.Items).To(HaveLen(1))
		})
	})
})
