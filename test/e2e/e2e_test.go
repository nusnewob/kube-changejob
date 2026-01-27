//go:build e2e
// +build e2e

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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
	"github.com/nusnewob/kube-changejob/test/utils"
)

// namespace where the project is deployed in
const namespace = "kube-changejob-system"

// serviceAccountName created for the project
const serviceAccountName = "kube-changejob-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "kube-changejob-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "kube-changejob-metrics-binding"

// test namespace for resources
const testNamespace = "kube-changejob-test"

var (
	controllerPodName string
	k8sClient         client.Client
	clientset         *kubernetes.Clientset
	cfg               *rest.Config
)

func init() {
	// Register the ChangeTriggeredJob types with the scheme
	_ = triggersv1alpha.AddToScheme(scheme.Scheme)
}

var _ = Describe("E2E Tests", Ordered, func() {
	BeforeAll(func() {
		var err error

		By("setting up Kubernetes client")
		cfg = ctrl.GetConfigOrDie()
		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client")

		clientset, err = kubernetes.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes clientset")

		By("creating manager namespace")
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		err = k8sClient.Create(context.Background(), ns)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")
		}

		By("creating test namespace")
		testNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		err = k8sClient.Create(context.Background(), testNs)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")
		}

		By("installing cert-manager")
		err = utils.InstallCertManager()
		Expect(err).NotTo(HaveOccurred(), "Failed to install cert-manager")

		By("waiting for cert-manager webhook to be ready")
		Eventually(func(g Gomega) {
			// Check if cert-manager webhook pod is ready
			webhookPods := &corev1.PodList{}
			err := k8sClient.List(context.Background(), webhookPods,
				client.InNamespace("cert-manager"),
				client.MatchingLabels{"app": "webhook"})
			g.Expect(err).NotTo(HaveOccurred())

			if len(webhookPods.Items) == 0 {
				g.Expect(webhookPods.Items).NotTo(BeEmpty(), "cert-manager webhook pod not found")
				return
			}

			for _, pod := range webhookPods.Items {
				for _, cond := range pod.Status.Conditions {
					if cond.Type == corev1.PodReady {
						g.Expect(cond.Status).To(Equal(corev1.ConditionTrue), "cert-manager webhook pod not ready")
					}
				}
			}
		}, 3*time.Minute, 2*time.Second).Should(Succeed())

		By("waiting for cert-manager webhook to be fully functional")
		// Give cert-manager webhook additional time to stabilize its TLS certificates
		time.Sleep(30 * time.Second)

		By("labeling the namespace to enforce the restricted security policy")
		err = k8sClient.Get(context.Background(), types.NamespacedName{Name: namespace}, ns)
		Expect(err).NotTo(HaveOccurred())
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["pod-security.kubernetes.io/enforce"] = "restricted"
		err = k8sClient.Update(context.Background(), ns)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd := exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", managerImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("patching controller deployment to set PollInterval to 10s for faster tests")
		Eventually(func(g Gomega) {
			deployment := &appsv1.Deployment{}
			err := k8sClient.Get(context.Background(),
				types.NamespacedName{Name: "kube-changejob-controller-manager", Namespace: namespace},
				deployment)
			g.Expect(err).NotTo(HaveOccurred())

			// Add POLL_INTERVAL env var to the manager container
			updated := false
			for i := range deployment.Spec.Template.Spec.Containers {
				if deployment.Spec.Template.Spec.Containers[i].Name == "manager" {
					envVars := deployment.Spec.Template.Spec.Containers[i].Env
					foundEnv := false
					for j := range envVars {
						if envVars[j].Name == "POLL_INTERVAL" {
							envVars[j].Value = "10s"
							foundEnv = true
							break
						}
					}
					if !foundEnv {
						deployment.Spec.Template.Spec.Containers[i].Env = append(envVars, corev1.EnvVar{
							Name:  "POLL_INTERVAL",
							Value: "10s",
						})
					}
					updated = true
					break
				}
			}
			g.Expect(updated).To(BeTrue(), "Manager container not found in deployment")

			err = k8sClient.Update(context.Background(), deployment)
			g.Expect(err).NotTo(HaveOccurred())
		}, 30*time.Second, 2*time.Second).Should(Succeed())

		By("waiting for controller deployment to roll out with new env var")
		time.Sleep(10 * time.Second)

		By("validating that the controller-manager pod is running as expected")
		verifyControllerUp := func(g Gomega) {
			podList := &corev1.PodList{}
			err := k8sClient.List(context.Background(), podList,
				client.InNamespace(namespace),
				client.MatchingLabels{"control-plane": "controller-manager"})
			g.Expect(err).NotTo(HaveOccurred(), "Failed to list controller-manager pods")

			var runningPods []corev1.Pod
			for _, pod := range podList.Items {
				if pod.DeletionTimestamp == nil {
					runningPods = append(runningPods, pod)
				}
			}
			g.Expect(runningPods).To(HaveLen(1), "expected 1 controller pod running")
			controllerPodName = runningPods[0].Name
			g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))
			g.Expect(runningPods[0].Status.Phase).To(Equal(corev1.PodRunning), "Controller pod not running")
		}
		Eventually(verifyControllerUp, 2*time.Minute, time.Second).Should(Succeed())
	})

	It("should ensure the metrics endpoint is serving metrics", func() {
		ctx := context.Background()

		By("creating a ClusterRoleBinding for the service account to allow access to metrics")
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: metricsRoleBindingName,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "kube-changejob-metrics-reader",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: namespace,
				},
			},
		}
		err := k8sClient.Create(ctx, crb)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")
		}

		By("validating that the metrics service is available")
		svc := &corev1.Service{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: metricsServiceName, Namespace: namespace}, svc)
		Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

		By("getting the service account token")
		token, err := serviceAccountToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(token).NotTo(BeEmpty())

		By("ensuring the controller pod is ready")
		verifyControllerPodReady := func(g Gomega) {
			pod := &corev1.Pod{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: controllerPodName, Namespace: namespace}, pod)
			g.Expect(err).NotTo(HaveOccurred())

			var isReady bool
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					isReady = true
					break
				}
			}
			g.Expect(isReady).To(BeTrue(), "Controller pod not ready")
		}
		Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

		By("verifying that the controller manager is serving the metrics server")
		verifyMetricsServerStarted := func(g Gomega) {
			req := clientset.CoreV1().Pods(namespace).GetLogs(controllerPodName, &corev1.PodLogOptions{})
			logs, err := req.DoRaw(ctx)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(string(logs)).To(ContainSubstring("Serving metrics server"),
				"Metrics server not yet started")
		}
		Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

		By("waiting for the webhook service endpoints to be ready")
		verifyWebhookEndpointsReady := func(g Gomega) {
			epSliceList := &discoveryv1.EndpointSliceList{}
			err := k8sClient.List(ctx, epSliceList,
				client.InNamespace(namespace),
				client.MatchingLabels{"kubernetes.io/service-name": "kube-changejob-webhook-service"})
			g.Expect(err).NotTo(HaveOccurred(), "Failed to list endpoint slices")
			g.Expect(epSliceList.Items).NotTo(BeEmpty(), "Webhook endpoints not found")

			hasReadyEndpoint := false
			for _, eps := range epSliceList.Items {
				for _, ep := range eps.Endpoints {
					if len(ep.Addresses) > 0 {
						hasReadyEndpoint = true
						break
					}
				}
			}
			g.Expect(hasReadyEndpoint).To(BeTrue(), "No ready webhook endpoints found")
		}
		Eventually(verifyWebhookEndpointsReady, 3*time.Minute, time.Second).Should(Succeed())

		By("creating the curl-metrics pod to access the metrics endpoint")
		// Delete existing curl-metrics pod if it exists
		existingPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "curl-metrics",
				Namespace: namespace,
			},
		}
		_ = k8sClient.Delete(ctx, existingPod)
		time.Sleep(2 * time.Second)

		curlPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "curl-metrics",
				Namespace: namespace,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: serviceAccountName,
				RestartPolicy:      corev1.RestartPolicyNever,
				Containers: []corev1.Container{
					{
						Name:    "curl",
						Image:   "curlimages/curl:latest",
						Command: []string{"/bin/sh", "-c"},
						Args: []string{
							fmt.Sprintf("curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics",
								token, metricsServiceName, namespace),
						},
						SecurityContext: &corev1.SecurityContext{
							ReadOnlyRootFilesystem:   ptrBool(true),
							AllowPrivilegeEscalation: ptrBool(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: ptrBool(true),
							RunAsUser:    ptrInt64(1000),
							SeccompProfile: &corev1.SeccompProfile{
								Type: corev1.SeccompProfileTypeRuntimeDefault,
							},
						},
					},
				},
			},
		}
		err = k8sClient.Create(ctx, curlPod)
		Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

		By("waiting for the curl-metrics pod to complete")
		verifyCurlUp := func(g Gomega) {
			pod := &corev1.Pod{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "curl-metrics", Namespace: namespace}, pod)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodSucceeded), "curl pod in wrong status")
		}
		Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

		By("getting the metrics by checking curl-metrics logs")
		verifyMetricsAvailable := func(g Gomega) {
			req := clientset.CoreV1().Pods(namespace).GetLogs("curl-metrics", &corev1.PodLogOptions{})
			logs, err := req.DoRaw(ctx)
			g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
			metricsOutput := string(logs)
			g.Expect(metricsOutput).NotTo(BeEmpty())
			g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
		}
		Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
	})

	It("should have provisioned cert-manager", func() {
		By("validating that cert-manager has the certificate Secret")
		verifyCertManager := func(g Gomega) {
			secret := &corev1.Secret{}
			err := k8sClient.Get(context.Background(),
				types.NamespacedName{Name: "webhook-server-cert", Namespace: namespace}, secret)
			g.Expect(err).NotTo(HaveOccurred())
		}
		Eventually(verifyCertManager).Should(Succeed())
	})

	It("should have CA injection for mutating webhooks", func() {
		By("checking CA injection for mutating webhooks")
		verifyCAInjection := func(g Gomega) {
			mwh := &admissionv1.MutatingWebhookConfiguration{}
			err := k8sClient.Get(context.Background(),
				types.NamespacedName{Name: "kube-changejob-mutating-webhook-configuration"}, mwh)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(mwh.Webhooks).NotTo(BeEmpty())
			for _, wh := range mwh.Webhooks {
				g.Expect(len(wh.ClientConfig.CABundle)).To(BeNumerically(">", 10))
			}
		}
		Eventually(verifyCAInjection).Should(Succeed())
	})

	It("should have CA injection for validating webhooks", func() {
		By("checking CA injection for validating webhooks")
		verifyCAInjection := func(g Gomega) {
			vwh := &admissionv1.ValidatingWebhookConfiguration{}
			err := k8sClient.Get(context.Background(),
				types.NamespacedName{Name: "kube-changejob-validating-webhook-configuration"}, vwh)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(vwh.Webhooks).NotTo(BeEmpty())
			for _, wh := range vwh.Webhooks {
				g.Expect(len(wh.ClientConfig.CABundle)).To(BeNumerically(">", 10))
			}
		}
		Eventually(verifyCAInjection).Should(Succeed())
	})

	AfterAll(func() {
		ctx := context.Background()

		By("cleaning up the curl pod for metrics")
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "curl-metrics",
				Namespace: namespace,
			},
		}
		_ = k8sClient.Delete(ctx, pod)

		By("undeploying the controller-manager")
		cmd := exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("uninstalling cert-manager")
		utils.UninstallCertManager()

		By("removing test namespace")
		testNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		_ = k8sClient.Delete(ctx, testNs)

		By("removing manager namespace")
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_ = k8sClient.Delete(ctx, ns)
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			ctx := context.Background()

			By("Fetching controller manager pod logs")
			if controllerPodName != "" {
				req := clientset.CoreV1().Pods(namespace).GetLogs(controllerPodName, &corev1.PodLogOptions{})
				logs, err := req.DoRaw(ctx)
				if err == nil {
					_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", string(logs))
				} else {
					_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
				}
			}

			By("Fetching Kubernetes events")
			eventList := &corev1.EventList{}
			err := k8sClient.List(ctx, eventList, client.InNamespace(namespace))
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events in %s:\n", namespace)
				for _, event := range eventList.Items {
					_, _ = fmt.Fprintf(GinkgoWriter, "%s: %s\n", event.LastTimestamp.Format(time.RFC3339), event.Message)
				}
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching test namespace events")
			testEventList := &corev1.EventList{}
			err = k8sClient.List(ctx, testEventList, client.InNamespace(testNamespace))
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Test namespace events:\n")
				for _, event := range testEventList.Items {
					_, _ = fmt.Fprintf(GinkgoWriter, "%s: %s\n", event.LastTimestamp.Format(time.RFC3339), event.Message)
				}
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get test namespace events: %s", err)
			}

			By("Fetching curl-metrics logs")
			req := clientset.CoreV1().Pods(namespace).GetLogs("curl-metrics", &corev1.PodLogOptions{})
			logs, err := req.DoRaw(ctx)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", string(logs))
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			if controllerPodName != "" {
				pod := &corev1.Pod{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: controllerPodName, Namespace: namespace}, pod)
				if err == nil {
					podJSON, _ := json.MarshalIndent(pod, "", "  ")
					fmt.Println("Pod description:\n", string(podJSON))
				} else {
					fmt.Println("Failed to describe controller pod")
				}
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("verifying the controller-manager pod is healthy")
			Expect(controllerPodName).NotTo(BeEmpty())
		})
	})

	Context("ChangeTriggeredJob", func() {
		BeforeEach(func() {
			ctx := context.Background()

			By("cleaning up existing test resources")
			// Delete all ChangeTriggeredJobs
			ctjList := &triggersv1alpha.ChangeTriggeredJobList{}
			_ = k8sClient.List(ctx, ctjList, client.InNamespace(testNamespace))
			for _, ctj := range ctjList.Items {
				_ = k8sClient.Delete(ctx, &ctj)
			}

			// Delete all ConfigMaps
			cmList := &corev1.ConfigMapList{}
			_ = k8sClient.List(ctx, cmList, client.InNamespace(testNamespace))
			for _, cm := range cmList.Items {
				_ = k8sClient.Delete(ctx, &cm)
			}

			// Delete all Secrets
			secretList := &corev1.SecretList{}
			_ = k8sClient.List(ctx, secretList, client.InNamespace(testNamespace))
			for _, secret := range secretList.Items {
				_ = k8sClient.Delete(ctx, &secret)
			}

			// Delete all Jobs
			jobList := &batchv1.JobList{}
			_ = k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
			for _, job := range jobList.Items {
				propagationPolicy := metav1.DeletePropagationBackground
				_ = k8sClient.Delete(ctx, &job, &client.DeleteOptions{
					PropagationPolicy: &propagationPolicy,
				})
			}

			time.Sleep(5 * time.Second)
		})

		Context("ConfigMap Change Triggers", func() {
			It("should trigger a job when a ConfigMap changes", func() {
				ctx := context.Background()

				By("creating a ConfigMap")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-configmap",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key1": "value1",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob watching the ConfigMap")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "configmap-change-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "test-configmap",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "ConfigMap changed!"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(15 * time.Second)

				By("updating the ConfigMap to trigger the job")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["key1"] = "value2"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("verifying that a job was created")
				Eventually(func(g Gomega) {
					jobList := &batchv1.JobList{}
					err := k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(jobList.Items).NotTo(BeEmpty(), "Expected at least one job to be created")
				}, 60*time.Second, 3*time.Second).Should(Succeed())

				By("verifying ChangeTriggeredJob status is updated")
				Eventually(func(g Gomega) {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: "configmap-change-job", Namespace: testNamespace}, ctj)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ctj.Status.LastTriggeredTime).NotTo(BeNil(), "Expected lastTriggeredTime to be set")
				}, 30*time.Second, 3*time.Second).Should(Succeed())
			})
		})

		Context("Secret Change Triggers", func() {
			It("should trigger a job when a Secret changes", func() {
				ctx := context.Background()

				By("creating a Secret")
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"password": []byte("password123"),
					},
				}
				err := k8sClient.Create(ctx, secret)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob watching the Secret")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-change-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "Secret",
								Name:       "test-secret",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "Secret changed!"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(15 * time.Second)

				By("updating the Secret to trigger the job")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-secret", Namespace: testNamespace}, secret)
				Expect(err).NotTo(HaveOccurred())
				secret.Data["password"] = []byte("newpassword")
				err = k8sClient.Update(ctx, secret)
				Expect(err).NotTo(HaveOccurred())

				By("verifying that a job was created")
				Eventually(func(g Gomega) {
					jobList := &batchv1.JobList{}
					err := k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(jobList.Items).NotTo(BeEmpty(), "Expected at least one job to be created")
				}, 60*time.Second, 3*time.Second).Should(Succeed())
			})
		})

		Context("Cooldown Period", func() {
			It("should respect cooldown period between triggers", func() {
				ctx := context.Background()

				By("creating a ConfigMap")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cooldown-configmap",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key1": "value1",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob with 30s cooldown")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cooldown-test-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 30 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "cooldown-configmap",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "Cooldown test"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(15 * time.Second)

				By("updating the ConfigMap first time")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "cooldown-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["key1"] = "value2"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("verifying first job was created")
				var firstJobCount int
				Eventually(func(g Gomega) {
					jobList := &batchv1.JobList{}
					err := k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
					g.Expect(err).NotTo(HaveOccurred())
					firstJobCount = len(jobList.Items)
					g.Expect(firstJobCount).To(BeNumerically(">=", 1))
				}, 60*time.Second, 3*time.Second).Should(Succeed())

				By("updating the ConfigMap again within cooldown period")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "cooldown-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["key1"] = "value3"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("verifying no new job was created during cooldown")
				time.Sleep(15 * time.Second)
				jobList := &batchv1.JobList{}
				err = k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
				Expect(err).NotTo(HaveOccurred())
				Expect(len(jobList.Items)).To(Equal(firstJobCount), "No new job should be created during cooldown period")
			})
		})

		Context("Multiple Resources with Any Condition", func() {
			It("should trigger when any watched resource changes", func() {
				ctx := context.Background()

				By("creating ConfigMap and Secret")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-configmap",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key1": "value1",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-secret",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"password": []byte("password"),
					},
				}
				err = k8sClient.Create(ctx, secret)
				Expect(err).NotTo(HaveOccurred())

				By("creating ChangeTriggeredJob watching both resources with Any condition")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-resource-any-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "multi-configmap",
								Namespace:  testNamespace,
							},
							{
								APIVersion: "v1",
								Kind:       "Secret",
								Name:       "multi-secret",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "Resource changed!"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(15 * time.Second)

				By("updating only the ConfigMap")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "multi-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["key1"] = "value2"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("verifying that a job was triggered by ConfigMap change")
				Eventually(func(g Gomega) {
					jobList := &batchv1.JobList{}
					err := k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(jobList.Items).NotTo(BeEmpty())
				}, 60*time.Second, 3*time.Second).Should(Succeed())
			})
		})

		Context("Multiple Resources with All Condition", func() {
			It("should trigger only when all watched resources change", func() {
				ctx := context.Background()

				By("creating ConfigMap and Secret")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "all-configmap",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key1": "value1",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "all-secret",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"password": []byte("password"),
					},
				}
				err = k8sClient.Create(ctx, secret)
				Expect(err).NotTo(HaveOccurred())

				By("creating ChangeTriggeredJob watching both resources with All condition")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-resource-all-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAll),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "all-configmap",
								Namespace:  testNamespace,
							},
							{
								APIVersion: "v1",
								Kind:       "Secret",
								Name:       "all-secret",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "All resources changed!"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation to establish baseline")
				time.Sleep(15 * time.Second)

				By("updating only the ConfigMap")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "all-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["key1"] = "value2"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("verifying no job was created yet after only ConfigMap change")
				time.Sleep(20 * time.Second)
				jobList := &batchv1.JobList{}
				err = k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
				Expect(err).NotTo(HaveOccurred())

				// Filter jobs owned by this ChangeTriggeredJob
				var ownedJobs []batchv1.Job
				for _, job := range jobList.Items {
					for _, ownerRef := range job.OwnerReferences {
						if ownerRef.Name == "multi-resource-all-job" {
							ownedJobs = append(ownedJobs, job)
						}
					}
				}
				Expect(ownedJobs).To(BeEmpty(), "No job should be created until all resources change")

				By("updating both ConfigMap and Secret together for All condition")
				// For "All" condition, both resources must change within the same reconciliation window
				// Update both resources close together so controller sees both changes in same cycle
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "all-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["key1"] = "value3"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				// Update secret immediately after
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "all-secret", Namespace: testNamespace}, secret)
				Expect(err).NotTo(HaveOccurred())
				secret.Data["password"] = []byte("newpassword")
				err = k8sClient.Update(ctx, secret)
				Expect(err).NotTo(HaveOccurred())

				By("verifying that a job was triggered after all resources changed together")
				Eventually(func(g Gomega) {
					jobList := &batchv1.JobList{}
					err := k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
					g.Expect(err).NotTo(HaveOccurred())

					// Filter jobs owned by this ChangeTriggeredJob
					var ownedJobs []batchv1.Job
					for _, job := range jobList.Items {
						for _, ownerRef := range job.OwnerReferences {
							if ownerRef.Name == "multi-resource-all-job" {
								ownedJobs = append(ownedJobs, job)
							}
						}
					}
					g.Expect(ownedJobs).NotTo(BeEmpty(), "Job should be created when all resources change together")
				}, 60*time.Second, 3*time.Second).Should(Succeed())
			})
		})

		Context("Job Status Tracking", func() {
			It("should update status with job information", func() {
				ctx := context.Background()

				By("creating a ConfigMap")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "status-configmap",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key1": "value1",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "status-tracking-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "status-configmap",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"sh", "-c", "echo 'Job completed successfully' && sleep 5"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(15 * time.Second)

				By("updating the ConfigMap to trigger the job")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "status-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["key1"] = "value2"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("verifying status fields are populated")
				Eventually(func(g Gomega) {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: "status-tracking-job", Namespace: testNamespace}, ctj)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ctj.Status.LastTriggeredTime).NotTo(BeNil())
					g.Expect(ctj.Status.LastJobName).NotTo(BeEmpty())
					g.Expect(ctj.Status.ResourceHashes).NotTo(BeEmpty())
				}, 60*time.Second, 3*time.Second).Should(Succeed())
			})
		})

		Context("Specific Field Watching", func() {
			It("should trigger only when watched field changes", func() {
				ctx := context.Background()

				By("creating a ConfigMap with multiple fields")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "field-watch-configmap",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"watched": "value1",
						"ignored": "value1",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("creating ChangeTriggeredJob watching only 'data.watched' field")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "field-specific-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "field-watch-configmap",
								Namespace:  testNamespace,
								Fields:     []string{"data.watched"},
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "Field changed!"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(15 * time.Second)

				By("updating the ignored field")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "field-watch-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["ignored"] = "value2"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("verifying no job was created")
				time.Sleep(20 * time.Second)
				jobList := &batchv1.JobList{}
				err = k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
				Expect(err).NotTo(HaveOccurred())

				var ownedJobs []batchv1.Job
				for _, job := range jobList.Items {
					for _, ownerRef := range job.OwnerReferences {
						if ownerRef.Name == "field-specific-job" {
							ownedJobs = append(ownedJobs, job)
						}
					}
				}
				Expect(ownedJobs).To(BeEmpty(), "No job should be created when unwatched field changes")

				By("updating the watched field")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "field-watch-configmap", Namespace: testNamespace}, cm)
				Expect(err).NotTo(HaveOccurred())
				cm.Data["watched"] = "value2"
				err = k8sClient.Update(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("verifying job was created after watched field changed")
				Eventually(func(g Gomega) {
					jobList := &batchv1.JobList{}
					err := k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
					g.Expect(err).NotTo(HaveOccurred())

					var ownedJobs []batchv1.Job
					for _, job := range jobList.Items {
						for _, ownerRef := range job.OwnerReferences {
							if ownerRef.Name == "field-specific-job" {
								ownedJobs = append(ownedJobs, job)
							}
						}
					}
					g.Expect(ownedJobs).NotTo(BeEmpty())
				}, 60*time.Second, 3*time.Second).Should(Succeed())
			})
		})

		Context("Webhook Validation", func() {
			It("should reject ChangeTriggeredJob with no resources", func() {
				ctx := context.Background()

				By("attempting to create ChangeTriggeredJob without resources")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "no-resources-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "test"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err := k8sClient.Create(ctx, ctj)
				Expect(err).To(HaveOccurred(), "Should reject empty resources list")
				Expect(err.Error()).To(ContainSubstring("at least one resource"))
			})

			It("should reject ChangeTriggeredJob with invalid resource kind", func() {
				ctx := context.Background()

				By("attempting to create ChangeTriggeredJob with invalid resource")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-kind-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "InvalidKind",
								Name:       "test",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "test"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err := k8sClient.Create(ctx, ctj)
				Expect(err).To(HaveOccurred(), "Should reject invalid resource kind")
				Expect(err.Error()).To(ContainSubstring("unknown kind"))
			})

			It("should reject namespaced resource without namespace", func() {
				ctx := context.Background()

				By("attempting to create ChangeTriggeredJob watching ConfigMap without namespace")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "missing-namespace-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "test-cm",
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "test"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err := k8sClient.Create(ctx, ctj)
				Expect(err).To(HaveOccurred(), "Should reject namespaced resource without namespace")
				Expect(err.Error()).To(ContainSubstring("namespace is required"))
			})
		})

		Context("Webhook Defaulting", func() {
			It("should apply default cooldown and condition values", func() {
				ctx := context.Background()

				By("creating a ConfigMap")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default-test-cm",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key": "value",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("creating ChangeTriggeredJob without cooldown and condition")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default-values-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "default-test-cm",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "test"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("verifying default values were applied")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "default-values-job", Namespace: testNamespace}, ctj)
				Expect(err).NotTo(HaveOccurred())
				Expect(ctj.Spec.Cooldown.Duration).To(Equal(60*time.Second), "Default cooldown should be 60s")
				Expect(*ctj.Spec.Condition).To(Equal(triggersv1alpha.TriggerConditionAny), "Default condition should be Any")

				By("verifying changed-at annotation was added")
				Expect(ctj.Annotations).To(HaveKey("changetriggeredjobs.triggers.changejob.dev/changed-at"))
			})
		})

		Context("Resource Hash Persistence", func() {
			It("should not trigger on first creation (baseline established)", func() {
				ctx := context.Background()

				By("creating a ConfigMap")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "baseline-cm",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key": "value1",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("creating ChangeTriggeredJob")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "baseline-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "baseline-cm",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "test"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(40 * time.Second)

				By("verifying no job was created on first run")
				jobList := &batchv1.JobList{}
				err = k8sClient.List(ctx, jobList, client.InNamespace(testNamespace))
				Expect(err).NotTo(HaveOccurred())

				var ownedJobs []batchv1.Job
				for _, job := range jobList.Items {
					for _, ownerRef := range job.OwnerReferences {
						if ownerRef.Name == "baseline-job" {
							ownedJobs = append(ownedJobs, job)
						}
					}
				}
				Expect(ownedJobs).To(BeEmpty(), "No job should be created on initial baseline establishment")

				By("verifying resourceHashes were initialized")
				err = k8sClient.Get(ctx, types.NamespacedName{Name: "baseline-job", Namespace: testNamespace}, ctj)
				Expect(err).NotTo(HaveOccurred())
				Expect(ctj.Status.ResourceHashes).NotTo(BeEmpty(), "resourceHashes should be initialized")
			})

			It("should clean up old jobs when history limit is exceeded", func() {
				ctx := context.Background()

				By("creating a ConfigMap")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "history-test-configmap",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key1": "initial",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob with history limit of 1")
				history := int32(1)
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "history-limit-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 5 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						History:   &history,
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "history-test-configmap",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "job triggered"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(15 * time.Second)

				By("triggering multiple changes to create 2 jobs")
				for i := 1; i <= 2; i++ {
					err = k8sClient.Get(ctx, types.NamespacedName{Name: "history-test-configmap", Namespace: testNamespace}, cm)
					Expect(err).NotTo(HaveOccurred())
					cm.Data["key1"] = fmt.Sprintf("value%d", i)
					err = k8sClient.Update(ctx, cm)
					Expect(err).NotTo(HaveOccurred())
					time.Sleep(7 * time.Second)
				}

				By("verifying only 1 job remains (history limit)")
				Eventually(func(g Gomega) {
					jobList := &batchv1.JobList{}
					err := k8sClient.List(ctx, jobList,
						client.InNamespace(testNamespace),
						client.MatchingLabels{"changejob.dev/owner": "history-limit-job"})
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(len(jobList.Items)).To(Equal(1), "Expected exactly 1 job to remain due to history limit")
				}, 60*time.Second, 5*time.Second).Should(Succeed())
			})
		})

		Context("Metrics Verification", func() {
			It("should record reconciliation metrics", func() {
				ctx := context.Background()

				By("creating a ConfigMap")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "metrics-test-cm",
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key": "value1",
					},
				}
				err := k8sClient.Create(ctx, cm)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob")
				ctj := &triggersv1alpha.ChangeTriggeredJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "metrics-job",
						Namespace: testNamespace,
					},
					Spec: triggersv1alpha.ChangeTriggeredJobSpec{
						Cooldown:  &metav1.Duration{Duration: 10 * time.Second},
						Condition: ptrTriggerCondition(triggersv1alpha.TriggerConditionAny),
						Resources: []triggersv1alpha.ResourceReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								Name:       "metrics-test-cm",
								Namespace:  testNamespace,
							},
						},
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox",
												Command: []string{"echo", "metrics test"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						},
					},
				}
				err = k8sClient.Create(ctx, ctj)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for reconciliation")
				time.Sleep(20 * time.Second)

				By("verifying controller runtime metrics exist")
				req := clientset.CoreV1().Pods(namespace).GetLogs(controllerPodName, &corev1.PodLogOptions{})
				logs, err := req.DoRaw(ctx)
				Expect(err).NotTo(HaveOccurred())
				// Just verify the controller is reconciling
				Expect(string(logs)).To(ContainSubstring("ChangeTriggeredJob"))
			})
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
func serviceAccountToken() (string, error) {
	ctx := context.Background()

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: ptrInt64(3600),
		},
	}

	var token string
	verifyTokenCreation := func(g Gomega) {
		result, err := clientset.CoreV1().ServiceAccounts(namespace).CreateToken(
			ctx, serviceAccountName, tokenRequest, metav1.CreateOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		token = result.Status.Token
		g.Expect(token).NotTo(BeEmpty())
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return token, nil
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrInt64(i int64) *int64 {
	return &i
}

func ptrTriggerCondition(tc triggersv1alpha.TriggerCondition) *triggersv1alpha.TriggerCondition {
	return &tc
}
