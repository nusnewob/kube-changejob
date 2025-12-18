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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

var controllerPodName string

var _ = Describe("E2E Tests", Ordered, func() {
	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)
		// Ignore error if namespace already exists

		By("creating test namespace")
		cmd = exec.Command("kubectl", "create", "ns", testNamespace)
		_, _ = utils.Run(cmd)
		// Ignore error if namespace already exists

		By("installing cert-manager")
		err := utils.InstallCertManager()
		Expect(err).NotTo(HaveOccurred(), "Failed to install cert-manager")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("validating that the controller-manager pod is running as expected")
		verifyControllerUp := func(g Gomega) {
			// Get the name of the controller-manager pod
			cmd := exec.Command("kubectl", "get",
				"pods", "-l", "control-plane=controller-manager",
				"-o", "go-template={{ range .items }}"+
					"{{ if not .metadata.deletionTimestamp }}"+
					"{{ .metadata.name }}"+
					"{{ \"\\n\" }}{{ end }}{{ end }}",
				"-n", namespace,
			)

			podOutput, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
			podNames := utils.GetNonEmptyLines(podOutput)
			g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
			controllerPodName = podNames[0]
			g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

			// Validate the pod's status
			cmd = exec.Command("kubectl", "get",
				"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
				"-n", namespace,
			)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
		}
		Eventually(verifyControllerUp, 2*time.Minute, time.Second).Should(Succeed())
	})

	It("should ensure the metrics endpoint is serving metrics", func() {
		By("creating a ClusterRoleBinding for the service account to allow access to metrics")
		cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
			"--clusterrole=kube-changejob-metrics-reader",
			fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
		)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

		By("validating that the metrics service is available")
		cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

		By("getting the service account token")
		token, err := serviceAccountToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(token).NotTo(BeEmpty())

		By("ensuring the controller pod is ready")
		verifyControllerPodReady := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
				"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("True"), "Controller pod not ready")
		}
		Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

		By("verifying that the controller manager is serving the metrics server")
		verifyMetricsServerStarted := func(g Gomega) {
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(ContainSubstring("Serving metrics server"),
				"Metrics server not yet started")
		}
		Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

		By("waiting for the webhook service endpoints to be ready")
		verifyWebhookEndpointsReady := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "endpointslices.discovery.k8s.io", "-n", namespace,
				"-l", "kubernetes.io/service-name=kube-changejob-webhook-service",
				"-o", "jsonpath={range .items[*]}{range .endpoints[*]}{.addresses[*]}{end}{end}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred(), "Webhook endpoints should exist")
			g.Expect(output).ShouldNot(BeEmpty(), "Webhook endpoints not yet ready")
		}
		Eventually(verifyWebhookEndpointsReady, 3*time.Minute, time.Second).Should(Succeed())

		// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

		By("creating the curl-metrics pod to access the metrics endpoint")
		cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
			"--namespace", namespace,
			"--image=curlimages/curl:latest",
			"--overrides",
			fmt.Sprintf(`{
				"spec": {
					"containers": [{
						"name": "curl",
						"image": "curlimages/curl:latest",
						"command": ["/bin/sh", "-c"],
						"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
						"securityContext": {
							"readOnlyRootFilesystem": true,
							"allowPrivilegeEscalation": false,
							"capabilities": {
								"drop": ["ALL"]
							},
							"runAsNonRoot": true,
							"runAsUser": 1000,
							"seccompProfile": {
								"type": "RuntimeDefault"
							}
						}
					}],
					"serviceAccountName": "%s"
				}
			}`, token, metricsServiceName, namespace, serviceAccountName))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

		By("waiting for the curl-metrics pod to complete.")
		verifyCurlUp := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
				"-o", "jsonpath={.status.phase}",
				"-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
		}
		Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

		By("getting the metrics by checking curl-metrics logs")
		verifyMetricsAvailable := func(g Gomega) {
			metricsOutput, err := getMetricsOutput()
			g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
			g.Expect(metricsOutput).NotTo(BeEmpty())
			g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
		}
		Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
	})

	It("should provisioned cert-manager", func() {
		By("validating that cert-manager has the certificate Secret")
		verifyCertManager := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "secrets", "webhook-server-cert", "-n", namespace)
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
		}
		Eventually(verifyCertManager).Should(Succeed())
	})

	It("should have CA injection for mutating webhooks", func() {
		By("checking CA injection for mutating webhooks")
		verifyCAInjection := func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"mutatingwebhookconfigurations.admissionregistration.k8s.io",
				"kube-changejob-mutating-webhook-configuration",
				"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
			mwhOutput, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))
		}
		Eventually(verifyCAInjection).Should(Succeed())
	})

	It("should have CA injection for validating webhooks", func() {
		By("checking CA injection for validating webhooks")
		verifyCAInjection := func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"validatingwebhookconfigurations.admissionregistration.k8s.io",
				"kube-changejob-validating-webhook-configuration",
				"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
			vwhOutput, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
		}
		Eventually(verifyCAInjection).Should(Succeed())
	})

	// +kubebuilder:scaffold:e2e-webhooks-checks

	// TODO: Customize the e2e test suite with scenarios specific to your project.
	// Consider applying sample/CR(s) and check their status and/or verifying
	// the reconciliation by using the metrics, i.e.:
	// metricsOutput, err := getMetricsOutput()
	// Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	// Expect(metricsOutput).To(ContainSubstring(
	//    fmt.Sprintf(`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
	//    strings.ToLower(<Kind>),
	// ))

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("uninstalling cert-manager")
		utils.UninstallCertManager()

		By("removing test namespace")
		cmd = exec.Command("kubectl", "delete", "ns", testNamespace)
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching test namespace events")
			cmd = exec.Command("kubectl", "get", "events", "-n", testNamespace, "--sort-by=.lastTimestamp")
			eventsOutput, err = utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Test namespace events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get test namespace events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
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
			// Clean up any existing resources before each test
			By("cleaning up existing test resources")
			cmd := exec.Command("kubectl", "delete", "changetriggeredjob", "--all", "-n", testNamespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "configmap", "--all", "-n", testNamespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "secret", "--all", "-n", testNamespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "job", "--all", "-n", testNamespace)
			_, _ = utils.Run(cmd)
			time.Sleep(5 * time.Second)
		})

		Context("ConfigMap Change Triggers", func() {
			It("should trigger a job when a ConfigMap changes", func() {
				By("creating a ConfigMap")
				configMapJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "test-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value1"
				  }
				}`, testNamespace)
				cmd := exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(configMapJSON))
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob watching the ConfigMap")
				changeJobJSON := fmt.Sprintf(`{
				  "apiVersion": "triggers.changejob.dev/v1alpha",
				  "kind": "ChangeTriggeredJob",
				  "metadata": {
				    "name": "configmap-change-job",
				    "namespace": "%s"
				  },
				  "spec": {
				    "cooldown": "10s",
				    "condition": "Any",
				    "resources": [
				      {
				        "apiVersion": "v1",
				        "kind": "ConfigMap",
				        "name": "test-configmap",
				        "namespace": "%s"
				      }
				    ],
				    "jobTemplate": {
				      "spec": {
				        "template": {
				          "spec": {
				            "containers": [
				              {
				                "name": "test",
				                "image": "busybox",
				                "command": ["echo", "ConfigMap changed!"]
				              }
				            ],
				            "restartPolicy": "Never"
				          }
				        }
				      }
				    }
				  }
				}`, testNamespace, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(changeJobJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(10 * time.Second)

				By("updating the ConfigMap to trigger the job")
				updateJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "test-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value2"
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(updateJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("verifying that a job was created")
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "jobs", "-n", testNamespace, "-o", "jsonpath={.items[*].metadata.name}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).NotTo(BeEmpty(), "Expected at least one job to be created")
				}, 90*time.Second, 5*time.Second).Should(Succeed())

				By("verifying ChangeTriggeredJob status is updated")
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "changetriggeredjob", "configmap-change-job", "-n", testNamespace, "-o", "jsonpath={.status.lastTriggeredTime}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).NotTo(BeEmpty(), "Expected lastTriggeredTime to be set")
				}, 30*time.Second, 3*time.Second).Should(Succeed())
			})
		})

		Context("Secret Change Triggers", func() {
			It("should trigger a job when a Secret changes", func() {
				By("creating a Secret")
				secretJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "Secret",
				  "metadata": {
				    "name": "test-secret",
				    "namespace": "%s"
				  },
				  "type": "Opaque",
				  "data": {
				    "password": "cGFzc3dvcmQxMjM="
				  }
				}`, testNamespace)
				cmd := exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(secretJSON))
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob watching the Secret")
				changeJobJSON := fmt.Sprintf(`{
				  "apiVersion": "triggers.changejob.dev/v1alpha",
				  "kind": "ChangeTriggeredJob",
				  "metadata": {
				    "name": "secret-change-job",
				    "namespace": "%s"
				  },
				  "spec": {
				    "cooldown": "10s",
				    "condition": "Any",
				    "resources": [
				      {
				        "apiVersion": "v1",
				        "kind": "Secret",
				        "name": "test-secret",
				        "namespace": "%s"
				      }
				    ],
				    "jobTemplate": {
				      "spec": {
				        "template": {
				          "spec": {
				            "containers": [
				              {
				                "name": "test",
				                "image": "busybox",
				                "command": ["echo", "Secret changed!"]
				              }
				            ],
				            "restartPolicy": "Never"
				          }
				        }
				      }
				    }
				  }
				}`, testNamespace, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(changeJobJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(10 * time.Second)

				By("updating the Secret to trigger the job")
				updateJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "Secret",
				  "metadata": {
				    "name": "test-secret",
				    "namespace": "%s"
				  },
				  "type": "Opaque",
				  "data": {
				    "password": "bmV3cGFzc3dvcmQ="
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(updateJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("verifying that a job was created")
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "jobs", "-n", testNamespace, "-o", "jsonpath={.items[*].metadata.name}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).NotTo(BeEmpty(), "Expected at least one job to be created")
				}, 90*time.Second, 5*time.Second).Should(Succeed())
			})
		})

		Context("Cooldown Period", func() {
			It("should respect cooldown period between triggers", func() {
				By("creating a ConfigMap")
				configMapJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "cooldown-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value1"
				  }
				}`, testNamespace)
				cmd := exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(configMapJSON))
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob with 30s cooldown")
				changeJobJSON := fmt.Sprintf(`{
				  "apiVersion": "triggers.changejob.dev/v1alpha",
				  "kind": "ChangeTriggeredJob",
				  "metadata": {
				    "name": "cooldown-test-job",
				    "namespace": "%s"
				  },
				  "spec": {
				    "cooldown": "30s",
				    "condition": "Any",
				    "resources": [
				      {
				        "apiVersion": "v1",
				        "kind": "ConfigMap",
				        "name": "cooldown-configmap",
				        "namespace": "%s"
				      }
				    ],
				    "jobTemplate": {
				      "spec": {
				        "template": {
				          "spec": {
				            "containers": [
				              {
				                "name": "test",
				                "image": "busybox",
				                "command": ["echo", "Cooldown test"]
				              }
				            ],
				            "restartPolicy": "Never"
				          }
				        }
				      }
				    }
				  }
				}`, testNamespace, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(changeJobJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(10 * time.Second)

				By("updating the ConfigMap first time")
				updateJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "cooldown-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value2"
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(updateJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("verifying first job was created")
				var firstJobCount int
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "jobs", "-n", testNamespace, "-o", "jsonpath={.items[*].metadata.name}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					jobs := utils.GetNonEmptyLines(output)
					firstJobCount = len(jobs)
					g.Expect(firstJobCount).To(BeNumerically(">=", 1))
				}, 90*time.Second, 5*time.Second).Should(Succeed())

				By("updating the ConfigMap again within cooldown period")
				updateJSON2 := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "cooldown-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value3"
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(updateJSON2))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("verifying no new job was created during cooldown")
				time.Sleep(15 * time.Second)
				cmd = exec.Command("kubectl", "get", "jobs", "-n", testNamespace, "-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())
				jobs := utils.GetNonEmptyLines(output)
				Expect(len(jobs)).To(Equal(firstJobCount), "No new job should be created during cooldown period")
			})
		})

		Context("Multiple Resources with Any Condition", func() {
			It("should trigger when any watched resource changes", func() {
				By("creating ConfigMap and Secret")
				configMapJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "multi-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value1"
				  }
				}`, testNamespace)
				cmd := exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(configMapJSON))
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				secretJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "Secret",
				  "metadata": {
				    "name": "multi-secret",
				    "namespace": "%s"
				  },
				  "type": "Opaque",
				  "data": {
				    "password": "cGFzc3dvcmQ="
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(secretJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("creating ChangeTriggeredJob watching both resources with Any condition")
				changeJobJSON := fmt.Sprintf(`{
				  "apiVersion": "triggers.changejob.dev/v1alpha",
				  "kind": "ChangeTriggeredJob",
				  "metadata": {
				    "name": "multi-resource-any-job",
				    "namespace": "%s"
				  },
				  "spec": {
				    "cooldown": "10s",
				    "condition": "Any",
				    "resources": [
				      {
				        "apiVersion": "v1",
				        "kind": "ConfigMap",
				        "name": "multi-configmap",
				        "namespace": "%s"
				      },
				      {
				        "apiVersion": "v1",
				        "kind": "Secret",
				        "name": "multi-secret",
				        "namespace": "%s"
				      }
				    ],
				    "jobTemplate": {
				      "spec": {
				        "template": {
				          "spec": {
				            "containers": [
				              {
				                "name": "test",
				                "image": "busybox",
				                "command": ["echo", "Resource changed!"]
				              }
				            ],
				            "restartPolicy": "Never"
				          }
				        }
				      }
				    }
				  }
				}`, testNamespace, testNamespace, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(changeJobJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(10 * time.Second)

				By("updating only the ConfigMap")
				updateJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "multi-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value2"
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(updateJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("verifying that a job was triggered by ConfigMap change")
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "jobs", "-n", testNamespace, "-o", "jsonpath={.items[*].metadata.name}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).NotTo(BeEmpty())
				}, 90*time.Second, 5*time.Second).Should(Succeed())
			})
		})

		Context("Multiple Resources with All Condition", func() {
			It("should trigger only when all watched resources change", func() {
				By("creating ConfigMap and Secret")
				configMapJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "all-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value1"
				  }
				}`, testNamespace)
				cmd := exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(configMapJSON))
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				secretJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "Secret",
				  "metadata": {
				    "name": "all-secret",
				    "namespace": "%s"
				  },
				  "type": "Opaque",
				  "data": {
				    "password": "cGFzc3dvcmQ="
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(secretJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("creating ChangeTriggeredJob watching both resources with All condition")
				changeJobJSON := fmt.Sprintf(`{
				  "apiVersion": "triggers.changejob.dev/v1alpha",
				  "kind": "ChangeTriggeredJob",
				  "metadata": {
				    "name": "multi-resource-all-job",
				    "namespace": "%s"
				  },
				  "spec": {
				    "cooldown": "10s",
				    "condition": "All",
				    "resources": [
				      {
				        "apiVersion": "v1",
				        "kind": "ConfigMap",
				        "name": "all-configmap",
				        "namespace": "%s"
				      },
				      {
				        "apiVersion": "v1",
				        "kind": "Secret",
				        "name": "all-secret",
				        "namespace": "%s"
				      }
				    ],
				    "jobTemplate": {
				      "spec": {
				        "template": {
				          "spec": {
				            "containers": [
				              {
				                "name": "test",
				                "image": "busybox",
				                "command": ["echo", "All resources changed!"]
				              }
				            ],
				            "restartPolicy": "Never"
				          }
				        }
				      }
				    }
				  }
				}`, testNamespace, testNamespace, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(changeJobJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(10 * time.Second)

				By("updating only the ConfigMap")
				updateJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "all-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value2"
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(updateJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("verifying no job was created yet")
				time.Sleep(20 * time.Second)
				cmd = exec.Command("kubectl", "get", "jobs", "-n", testNamespace, "-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(BeEmpty(), "No job should be created until all resources change")

				By("updating the Secret as well")
				updateSecretJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "Secret",
				  "metadata": {
				    "name": "all-secret",
				    "namespace": "%s"
				  },
				  "type": "Opaque",
				  "data": {
				    "password": "bmV3cGFzc3dvcmQ="
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(updateSecretJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("verifying that a job was triggered after all resources changed")
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "jobs", "-n", testNamespace, "-o", "jsonpath={.items[*].metadata.name}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).NotTo(BeEmpty())
				}, 90*time.Second, 5*time.Second).Should(Succeed())
			})
		})

		Context("Job Status Tracking", func() {
			It("should update status with job information", func() {
				By("creating a ConfigMap")
				configMapJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "status-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value1"
				  }
				}`, testNamespace)
				cmd := exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(configMapJSON))
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("creating a ChangeTriggeredJob")
				changeJobJSON := fmt.Sprintf(`{
				  "apiVersion": "triggers.changejob.dev/v1alpha",
				  "kind": "ChangeTriggeredJob",
				  "metadata": {
				    "name": "status-tracking-job",
				    "namespace": "%s"
				  },
				  "spec": {
				    "cooldown": "10s",
				    "condition": "Any",
				    "resources": [
				      {
				        "apiVersion": "v1",
				        "kind": "ConfigMap",
				        "name": "status-configmap",
				        "namespace": "%s"
				      }
				    ],
				    "jobTemplate": {
				      "spec": {
				        "template": {
				          "spec": {
				            "containers": [
				              {
				                "name": "test",
				                "image": "busybox",
				                "command": ["sh", "-c", "echo 'Job completed successfully' && sleep 5"]
				              }
				            ],
				            "restartPolicy": "Never"
				          }
				        }
				      }
				    }
				  }
				}`, testNamespace, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(changeJobJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for initial reconciliation")
				time.Sleep(10 * time.Second)

				By("updating the ConfigMap to trigger the job")
				updateJSON := fmt.Sprintf(`{
				  "apiVersion": "v1",
				  "kind": "ConfigMap",
				  "metadata": {
				    "name": "status-configmap",
				    "namespace": "%s"
				  },
				  "data": {
				    "key1": "value2"
				  }
				}`, testNamespace)
				cmd = exec.Command("kubectl", "apply", "-f", "-")
				cmd.Stdin = bytes.NewReader([]byte(updateJSON))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("verifying status fields are populated")
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "changetriggeredjob", "status-tracking-job", "-n", testNamespace, "-o", "json")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(ContainSubstring("lastTriggeredTime"))
					g.Expect(output).To(ContainSubstring("lastJobName"))
					g.Expect(output).To(ContainSubstring("resourceHashes"))
				}, 90*time.Second, 5*time.Second).Should(Succeed())
			})
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

// Helper function to convert string to io.Reader
func stringReader(s string) *stringReaderImpl {
	return &stringReaderImpl{content: s, pos: 0}
}

type stringReaderImpl struct {
	content string
	pos     int
}

func (r *stringReaderImpl) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.content) {
		return 0, nil
	}
	n = copy(p, r.content[r.pos:])
	r.pos += n
	return n, nil
}
