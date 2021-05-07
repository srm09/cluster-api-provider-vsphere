/*
Copyright 2021 The Kubernetes Authors.

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

package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha4"
)

const (
	timeout = time.Second * 5
)

var _ = FDescribe("ClusterReconciler", func() {
	BeforeEach(func() {})
	AfterEach(func() {})

	Context("Reconcile an VSphereCluster", func() {
		It("should create a cluster", func() {
			ctx := context.Background()

			capiCluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test1-",
					Namespace:    "default",
				},
				Spec: clusterv1.ClusterSpec{
					InfrastructureRef: &corev1.ObjectReference{
						APIVersion: infrav1.GroupVersion.String(),
						Kind:       "VSphereCluster",
						Name:       "vsphere-test1",
					},
				},
			}
			// Create the CAPI cluster (owner) object
			Expect(testEnv.Create(ctx, capiCluster)).To(Succeed())
			defer func() {
				Expect(testEnv.Cleanup(ctx, capiCluster)).To(Succeed())
			}()
			Expect(testEnv.CreateKubeconfigSecret(ctx, capiCluster)).To(Succeed())

			instance := &infrav1.VSphereCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vsphere-test1",
					Namespace: "default",
				},
				Spec: infrav1.VSphereClusterSpec{
					CloudProviderConfiguration: infrav1.CPIConfig{
						// To ensure that the CloudProvider config available check passes
						ProviderConfig: infrav1.CPIProviderConfig{
							Cloud: &infrav1.CPICloudConfig{
								ControllerImage: "gcr.io/cloud-provider-vsphere/cpi/release/manager:v1.18.1",
							},
						},
						VCenter: map[string]infrav1.CPIVCenterConfig{
							testEnv.VCSimulator.Server.URL.Host : {},
						},
					},
					Server: testEnv.VCSimulator.Server.URL.Host,
				},
			}

			// Create the VSphereCluster object
			Expect(testEnv.Create(ctx, instance)).To(Succeed())
			key := client.ObjectKey{Namespace: instance.Namespace, Name: instance.Name}
			defer func() {
				Expect(testEnv.Delete(ctx, instance)).To(Succeed())
			}()

			By("setting the OwnerRef on the VSphereCluster")
			Eventually(func() error {
				ph, err := patch.NewHelper(instance, testEnv)
				Expect(err).ShouldNot(HaveOccurred())
				instance.OwnerReferences = append(instance.OwnerReferences, metav1.OwnerReference{
					Kind: "Cluster",
					APIVersion: clusterv1.GroupVersion.String(),
					Name: capiCluster.Name,
					UID: "blah",
				})
				return ph.Patch(ctx, instance, patch.WithStatusObservedGeneration{})
			}, timeout).Should(BeNil())

			Eventually(func() bool {
				// Make sure the VSphereCluster exists.
				if err := testEnv.Get(ctx, key, instance); err != nil {
					return false
				}
				return len(instance.Finalizers) > 0 &&
					conditions.IsTrue(instance, infrav1.VCenterAvailableCondition) &&
					instance.Status.Ready
			}, timeout).Should(BeTrue())

			By("setting the ControlPlane endpoint on VSphereCluster")
			Eventually(func() error {
				ph, err := patch.NewHelper(instance, testEnv)
				Expect(err).ShouldNot(HaveOccurred())
				instance.Spec.ControlPlaneEndpoint = infrav1.APIEndpoint{
					Host: "1.2.3.4", Port: 1234,
				}
				return ph.Patch(ctx, instance, patch.WithStatusObservedGeneration{})
			}, timeout).Should(BeNil())

			Eventually(func() bool {
				// Make sure the VSphereCluster exists.
				if err := testEnv.Get(ctx, key, instance); err != nil {
					return false
				}
				return len(instance.Finalizers) > 0 &&
					conditions.IsTrue(instance, infrav1.VCenterAvailableCondition) &&
					instance.Status.Ready
			}, timeout).Should(BeTrue())

			//TODO: reconcileCloudConfigSecret here
		})
	})
})
