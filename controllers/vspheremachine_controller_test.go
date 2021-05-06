package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha4"
)

var _ = FDescribe("VsphereMachineReconciler", func() {

	var (
		capiCluster *clusterv1.Cluster
		capiMachine *clusterv1.Machine

		infraCluster *infrav1.VSphereCluster
		infraMachine *infrav1.VSphereMachine
		vm *infrav1.VSphereVM

		testNs *corev1.Namespace
	)

	BeforeEach(func() {
		var err error
		testNs, err = testEnv.CreateNamespace(ctx, "vsphere-machine-reconciler")
		Expect(err).NotTo(HaveOccurred())

		capiCluster = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test1-cluster",
				Namespace:    testNs.Name,
			},
			Spec: clusterv1.ClusterSpec{
				InfrastructureRef: &corev1.ObjectReference{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "VsphereCluster",
					Name:       "vsphere-test1",
				},
			},
		}

		infraCluster = &infrav1.VSphereCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vsphere-test1",
				Namespace: testNs.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "cluster.x-k8s.io/v1alpha4",
						Kind:               "Cluster",
						Name:               capiCluster.Name,
						UID:"blah",
					},
				},
			},
			Spec: infrav1.VSphereClusterSpec{},
		}

		capiMachine = &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				//GenerateName: "machine-created-",
				Name: "capi-machine-1",
				Namespace:    testNs.Name,
				Finalizers:   []string{clusterv1.MachineFinalizer},
				Labels: map[string]string{
					clusterv1.ClusterLabelName: capiCluster.Name,
				},
			},
			Spec: clusterv1.MachineSpec{
				ClusterName: capiCluster.Name,
				InfrastructureRef: corev1.ObjectReference{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "VsphereMachine",
					Name:       "vsphere-machine-1",
				},
			},
			Status: clusterv1.MachineStatus{
				//NodeRef: &corev1.ObjectReference{
				//	Name: "test",
				//},
			},
		}

		infraMachine = &infrav1.VSphereMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name: "vsphere-machine-1",
				Namespace: testNs.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "machine.x-k8s.io/v1alpha4",
						Kind:               "Machine",
						Name:               capiMachine.Name,
						UID:"blah",
					},
				},
			},
			Spec:       infrav1.VSphereMachineSpec{
				VirtualMachineCloneSpec: infrav1.VirtualMachineCloneSpec{
					Template: "ubuntu-k9s-1.19",
					Network:  infrav1.NetworkSpec{
						Devices: []infrav1.NetworkDeviceSpec{
							{NetworkName: "network-1",DHCP4: true},
						},
					},
				},
			},
			Status:     infrav1.VSphereMachineStatus{},
		}

		vm = &infrav1.VSphereVM{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: infraMachine.Namespace,
				Name:      infraMachine.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "infrastructure.cluster.x-k8s.io/v1alpha4",
						Kind:       "VsphereMachine",
						Name:               infraMachine.Name,
						UID:"blah",
					},
				},
			},
			Spec: infrav1.VSphereVMSpec{
				VirtualMachineCloneSpec: infrav1.VirtualMachineCloneSpec{
					Template: "ubuntu-k9s-1.19",
					Network:  infrav1.NetworkSpec{
						Devices: []infrav1.NetworkDeviceSpec{
							{NetworkName: "network-1",DHCP4: true},
						},
					},
					Server: testEnv.VCSimulator.Server.URL.Host,
				},
			},
		}

		Expect(testEnv.Create(ctx, capiCluster)).To(Succeed())
		Expect(testEnv.Create(ctx, capiMachine)).To(Succeed())
		Expect(testEnv.Create(ctx, infraCluster)).To(Succeed())
		Expect(testEnv.Create(ctx, infraMachine)).To(Succeed())
		Expect(testEnv.Create(ctx, vm)).To(Succeed())
	})

	AfterEach(func() {
		defer func(do ...client.Object) {
			Expect(testEnv.Cleanup(ctx, do...)).To(Succeed())
		}(capiCluster, testNs)
	})

	Context("In case of errors on the VsphereVM", func() {
		It("should surface the errors to the Machine", func() {

			By("setting the failure message and reason on the VM")
			Eventually(func() bool {
				ph, err := patch.NewHelper(vm, testEnv)
				Expect(err).ShouldNot(HaveOccurred())
				vm.Status.FailureReason = capierrors.MachineStatusErrorPtr(capierrors.UpdateMachineError)
				vm.Status.FailureMessage = pointer.StringPtr("some failure here")
				Expect(ph.Patch(ctx, vm, patch.WithStatusObservedGeneration{})).ShouldNot(HaveOccurred())
				return true
			}, timeout).Should(BeTrue())

			Eventually(func() bool {
				key := client.ObjectKey{Namespace: testNs.Name, Name: infraMachine.Name}
				if err := testEnv.Get(ctx, key, infraMachine); err != nil {
					return false
				}
				//Expect(testEnv.Get(ctx, key, infraMachine)).NotTo(HaveOccurred())
				//Expect(infraMachine.Status.FailureReason).NotTo(BeNil())
				return infraMachine.Status.FailureReason != nil
			}, timeout).Should(BeTrue())
		})
	})
})
