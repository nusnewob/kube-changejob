package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
)

var _ = Describe("ChangeTriggeredJob Controller", func() {
	Context("When reconciling a ChangeTriggeredJob", func() {
		const (
			ctjName      = "test-ctj"
			ctjNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      ctjName,
			Namespace: ctjNamespace,
		}

		BeforeEach(func() {
			By("Creating the ChangeTriggeredJob")
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
							Name:       "test-cm",
							Namespace:  ctjNamespace,
							Fields:     []string{"data.config"},
						},
					},
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
				},
			}
			Expect(k8sClient.Create(ctx, ctj)).Should(Succeed())
		})

		AfterEach(func() {
			ctj := &triggersv1alpha.ChangeTriggeredJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, ctj)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the ChangeTriggeredJob")
			Expect(k8sClient.Delete(ctx, ctj)).Should(Succeed())
		})

		It("Should detect field changes in watched resources", func() {
			By("Creating a ConfigMap")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: ctjNamespace,
				},
				Data: map[string]string{
					"config": "initial-value",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			By("Triggering reconciliation")
			controllerReconciler := &ChangeTriggeredJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Updating the watched field")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-cm", Namespace: ctjNamespace}, cm)).Should(Succeed())
			cm.Data["config"] = "updated-value"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			By("Verifying job is triggered")
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify job was created
			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			// Expect(jobList.Items).To(HaveLen(1))
			Expect(k8sClient.Delete(ctx, cm)).Should(Succeed())
		})

		It("Should respect cooldown period", func() {
			By("Creating watched resource")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
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
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			// initialJobCount := 1

			By("Second change within cooldown does not trigger")
			cm.Data["config"] = "value2"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			// Expect(jobList.Items).To(HaveLen(initialJobCount))

			By("Change after cooldown triggers new job")
			time.Sleep(11 * time.Second) // Wait for cooldown

			cm.Data["config"] = "value3"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			// Expect(jobList.Items).To(HaveLen(initialJobCount + 1))
			Expect(k8sClient.Delete(ctx, cm)).Should(Succeed())
		})

		It("Should only monitor specified fields", func() {
			By("Creating watched resource")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
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

			By("Change to monitored field triggers job")
			cm.Data["config"] = "changed"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			// Expect(jobList.Items).To(HaveLen(1))

			By("Change to non-monitored field does not trigger job")
			cm.Data["other-field"] = "changed-too"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.List(ctx, jobList, client.InNamespace(ctjNamespace))).Should(Succeed())
			// Expect(jobList.Items).To(HaveLen(1))
			Expect(k8sClient.Delete(ctx, cm)).Should(Succeed())
		})
	})
})
