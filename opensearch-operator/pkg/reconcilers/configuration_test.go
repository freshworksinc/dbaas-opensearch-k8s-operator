package reconcilers

import (
	"context"
	"strings"

	opsterv1 "github.com/Opster/opensearch-k8s-operator/opensearch-operator/api/v1"
	"github.com/Opster/opensearch-k8s-operator/opensearch-operator/mocks/github.com/Opster/opensearch-k8s-operator/opensearch-operator/pkg/reconcilers/k8s"
	"github.com/Opster/opensearch-k8s-operator/opensearch-operator/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func newConfigurationReconciler(
	client *k8s.MockK8sClient,
	recorder record.EventRecorder,
	reconcilerContext *ReconcilerContext,
	instance *opsterv1.OpenSearchCluster,
) *ConfigurationReconciler {
	return &ConfigurationReconciler{
		client:            client,
		reconcilerContext: reconcilerContext,
		recorder:          recorder,
		instance:          instance,
	}
}

var _ = Describe("Configuration Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		clusterName = "configuration-test"
	)

	Context("When reconciling  with AdditionalVolumes", func() {
		It("should create emptyDir volumes for conf, plugins and logs", func() {
			mockClient := k8s.NewMockK8sClient(GinkgoT())
			mockClient.On("Context").Return(context.Background())

			spec := opsterv1.OpenSearchCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: clusterName,
					UID:       "dummyuid",
				},
				Spec: opsterv1.ClusterSpec{
					General: opsterv1.GeneralConfig{
						AdditionalVolumes: []opsterv1.AdditionalVolume{
							{
								Name:     "rw-config",
								Path:     "/usr/share/opensearch/config",
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
							{
								Name:     "rw-plugins",
								Path:     "/usr/share/opensearch/plugins",
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
							{
								Name:     "rw-logs",
								Path:     "/usr/share/opensearch/logs",
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					NodePools: []opsterv1.NodePool{
						{
							Component: "test",
							Roles:     []string{"master", "data"},
						},
					},
				},
			}

			reconcilerContext := NewReconcilerContext(&helpers.MockEventRecorder{}, &spec, spec.Spec.NodePools)

			underTest := newConfigurationReconciler(
				mockClient,
				&helpers.MockEventRecorder{},
				&reconcilerContext,
				&spec,
			)

			_, err := underTest.Reconcile()
			Expect(err).ToNot(HaveOccurred())

			// Verify volumes were created
			Expect(len(reconcilerContext.Volumes)).To(Equal(3))
			Expect(len(reconcilerContext.VolumeMounts)).To(Equal(3))

			// Check volume mounts
			configVolume := reconcilerContext.Volumes[0]
			Expect(configVolume.Name).To(Equal("rw-config"))
			Expect(configVolume.EmptyDir).ToNot(BeNil())

			pluginsVolume := reconcilerContext.Volumes[1]
			Expect(pluginsVolume.Name).To(Equal("rw-plugins"))
			Expect(pluginsVolume.EmptyDir).ToNot(BeNil())

			logsVolume := reconcilerContext.Volumes[2]
			Expect(logsVolume.Name).To(Equal("rw-logs"))
			Expect(logsVolume.EmptyDir).ToNot(BeNil())

			// Check volume mounts
			configMount := reconcilerContext.VolumeMounts[0]
			Expect(configMount.Name).To(Equal("rw-config"))
			Expect(configMount.MountPath).To(Equal("/usr/share/opensearch/config"))

			pluginsMount := reconcilerContext.VolumeMounts[1]
			Expect(pluginsMount.Name).To(Equal("rw-plugins"))
			Expect(pluginsMount.MountPath).To(Equal("/usr/share/opensearch/plugins"))

			logsMount := reconcilerContext.VolumeMounts[2]
			Expect(logsMount.Name).To(Equal("rw-logs"))
			Expect(logsMount.MountPath).To(Equal("/usr/share/opensearch/logs"))
		})

	})

	Context("When Reconciling the configuration controller with no configuration snippets", func() {
		It("should not create a configmap ", func() {
			mockClient := k8s.NewMockK8sClient(GinkgoT())

			spec := opsterv1.OpenSearchCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: clusterName,
					UID:       "dummyuid",
				},
				Spec: opsterv1.ClusterSpec{
					General: opsterv1.GeneralConfig{},
					NodePools: []opsterv1.NodePool{
						{
							Component: "test",
							Roles: []string{
								"master",
								"data",
							},
						},
					},
				},
			}

			reconcilerContext := NewReconcilerContext(&helpers.MockEventRecorder{}, &spec, spec.Spec.NodePools)

			underTest := newConfigurationReconciler(
				mockClient,
				&helpers.MockEventRecorder{},
				&reconcilerContext,
				&spec,
			)
			_, err := underTest.Reconcile()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When Reconciling the configuration controller with some configuration snippets", func() {
		It("should create a configmap ", func() {
			mockClient := k8s.NewMockK8sClient(GinkgoT())

			spec := opsterv1.OpenSearchCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: clusterName,
					UID:       "dummyuid",
				},
				Spec: opsterv1.ClusterSpec{
					General: opsterv1.GeneralConfig{},
					NodePools: []opsterv1.NodePool{
						{
							Component: "test",
							Roles: []string{
								"master",
								"data",
							},
						},
					},
				},
			}

			mockClient.EXPECT().Scheme().Return(scheme.Scheme)
			mockClient.EXPECT().Context().Return(context.Background())
			var createdConfigMap *corev1.ConfigMap
			mockClient.On("CreateConfigMap", mock.Anything).
				Return(func(cm *corev1.ConfigMap) (*ctrl.Result, error) {
					createdConfigMap = cm
					return &ctrl.Result{}, nil
				})

			reconcilerContext := NewReconcilerContext(&helpers.MockEventRecorder{}, &spec, spec.Spec.NodePools)

			underTest := newConfigurationReconciler(
				mockClient,
				&helpers.MockEventRecorder{},
				&reconcilerContext,
				&spec,
			)
			reconcilerContext.AddConfig("foo", "bar")
			reconcilerContext.AddConfig("bar", "something")
			reconcilerContext.AddConfig("bar", "baz")
			_, err := underTest.Reconcile()
			Expect(err).ToNot(HaveOccurred())

			Expect(createdConfigMap).ToNot(BeNil())

			data, exists := createdConfigMap.Data["opensearch.yml"]
			Expect(exists).To(BeTrue())
			Expect(strings.Contains(data, "foo: bar\n")).To(BeTrue())
			Expect(strings.Contains(data, "bar: baz\n")).To(BeTrue())
		})
	})
})
