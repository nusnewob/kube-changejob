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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	triggersv1alpha "github.com/nusnewob/kube-changejob/api/v1alpha"
)

var _ = Describe("Poller", func() {
	var (
		poller    Poller
		ctx       context.Context
		namespace = "default"
	)

	BeforeEach(func() {
		ctx = context.Background()
		poller = Poller{Client: k8sClient}
	})

	Context("When polling resources", func() {
		It("Should poll a ConfigMap and hash its fields", func() {
			cmName := fmt.Sprintf("test-cm-%d", time.Now().UnixNano())

			By("Creating a ConfigMap")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: namespace,
				},
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			By("Polling the ConfigMap")
			ref := triggersv1alpha.ResourceReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       cmName,
				Namespace:  namespace,
				Fields:     []string{"data.key1"},
			}

			status, err := poller.Poll(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the status")
			Expect(status.APIVersion).To(Equal("v1"))
			Expect(status.Kind).To(Equal("ConfigMap"))
			Expect(status.Name).To(Equal(cmName))
			Expect(status.Namespace).To(Equal(namespace))
			Expect(status.Fields).To(HaveLen(1))
			Expect(status.Fields[0].Field).To(Equal("data.key1"))
			Expect(status.Fields[0].LastHash).NotTo(BeEmpty())

			Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		})

		It("Should detect changes in field values", func() {
			cmName := fmt.Sprintf("test-cm-%d", time.Now().UnixNano())

			By("Creating a ConfigMap")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: namespace,
				},
				Data: map[string]string{
					"key1": "value1",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			By("First poll")
			ref := triggersv1alpha.ResourceReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       cmName,
				Namespace:  namespace,
				Fields:     []string{"data.key1"},
			}

			status1, err := poller.Poll(ctx, ref)
			Expect(err).NotTo(HaveOccurred())
			hash1 := status1.Fields[0].LastHash

			By("Updating the ConfigMap")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: namespace}, cm)).Should(Succeed())
			cm.Data["key1"] = "value2"
			Expect(k8sClient.Update(ctx, cm)).Should(Succeed())

			By("Second poll")
			status2, err := poller.Poll(ctx, ref)
			Expect(err).NotTo(HaveOccurred())
			hash2 := status2.Fields[0].LastHash

			By("Verifying hash changed")
			Expect(hash2).NotTo(Equal(hash1))

			Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		})

		It("Should handle multiple fields", func() {
			cmName := fmt.Sprintf("test-cm-%d", time.Now().UnixNano())

			By("Creating a ConfigMap with multiple fields")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: namespace,
				},
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			By("Polling multiple fields")
			ref := triggersv1alpha.ResourceReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       cmName,
				Namespace:  namespace,
				Fields:     []string{"data.key1", "data.key2"},
			}

			status, err := poller.Poll(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying all fields are hashed")
			Expect(status.Fields).To(HaveLen(2))
			fieldMap := make(map[string]string)
			for _, f := range status.Fields {
				fieldMap[f.Field] = f.LastHash
			}
			Expect(fieldMap).To(HaveKey("data.key1"))
			Expect(fieldMap).To(HaveKey("data.key2"))

			Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		})

		It("Should handle missing fields gracefully", func() {
			cmName := fmt.Sprintf("test-cm-%d", time.Now().UnixNano())

			By("Creating a ConfigMap without the watched field")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: namespace,
				},
				Data: map[string]string{
					"other-key": "value",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			By("Polling a non-existent field")
			ref := triggersv1alpha.ResourceReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       cmName,
				Namespace:  namespace,
				Fields:     []string{"data.missing-key"},
			}

			status, err := poller.Poll(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no fields are returned")
			Expect(status.Fields).To(BeEmpty())

			Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		})
	})

	Context("Helper functions", func() {
		It("Should generate correct resource keys", func() {
			By("Testing with namespace")
			key := ResourceKey("v1", "ConfigMap", "default", "my-cm")
			Expect(key).To(Equal("v1/ConfigMap/default/my-cm"))

			By("Testing without namespace (cluster-scoped)")
			key = ResourceKey("v1", "Node", "", "my-node")
			Expect(key).To(Equal("v1/Node/my-node"))
		})

		It("Should index resource statuses correctly", func() {
			By("Creating resource statuses")
			statuses := []triggersv1alpha.ResourceReferenceStatus{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "cm1",
					Namespace:  "default",
				},
				{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       "secret1",
					Namespace:  "default",
				},
			}

			By("Indexing the statuses")
			index := IndexResourceStatuses(statuses)

			By("Verifying index contains correct keys")
			Expect(index).To(HaveLen(2))
			Expect(index).To(HaveKey("v1/ConfigMap/default/cm1"))
			Expect(index).To(HaveKey("v1/Secret/default/secret1"))
		})

		It("Should index resource references correctly", func() {
			By("Creating resource references")
			references := []triggersv1alpha.ResourceReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "cm1",
					Namespace:  "default",
				},
				{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       "secret1",
					Namespace:  "default",
				},
			}

			By("Indexing the references")
			index := IndexResourceReferences(references)

			By("Verifying index contains correct keys")
			Expect(index).To(HaveLen(2))
			Expect(index).To(HaveKey("v1/ConfigMap/default/cm1"))
			Expect(index).To(HaveKey("v1/Secret/default/secret1"))
		})

		It("Should hash objects consistently", func() {
			By("Hashing the same object twice")
			obj := map[string]any{
				"key1": "value1",
				"key2": 123,
			}

			hash1, err := HashObject(obj)
			Expect(err).NotTo(HaveOccurred())

			hash2, err := HashObject(obj)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying hashes are identical")
			Expect(hash1).To(Equal(hash2))
		})

		It("Should produce different hashes for different objects", func() {
			By("Hashing two different objects")
			obj1 := map[string]any{"key": "value1"}
			obj2 := map[string]any{"key": "value2"}

			hash1, err := HashObject(obj1)
			Expect(err).NotTo(HaveOccurred())

			hash2, err := HashObject(obj2)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying hashes are different")
			Expect(hash1).NotTo(Equal(hash2))
		})
	})
})
