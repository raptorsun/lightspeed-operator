package controller

import (
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	olsv1alpha1 "github.com/openshift/lightspeed-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

var _ = Describe("App server assets", func() {

	var cr *olsv1alpha1.OLSConfig
	deploymentSelectorLabels := map[string]string{
		"app.kubernetes.io/component":  "application-server",
		"app.kubernetes.io/managed-by": "lightspeed-operator",
		"app.kubernetes.io/name":       "lightspeed-service-api",
		"app.kubernetes.io/part-of":    "openshift-lightspeed",
	}

	Context("complete custome resource", func() {
		BeforeEach(func() {
			cr = getCompleteOLSConfigCR()
		})

		It("should generate a service account", func() {
			r := OLSConfigReconciler{}
			sa, err := r.generateServiceAccount(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(sa.Name).To(Equal(OLSAppServerServiceAccountName))
			Expect(sa.Namespace).To(Equal(cr.Namespace))
		})

		It("should generate the olsconfig config map", func() {
			r := OLSConfigReconciler{}
			cm, err := r.generateOLSConfigMap(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Name).To(Equal(OLSConfigCmName))
			Expect(cm.Namespace).To(Equal(cr.Namespace))
			olsconfigGenerated := AppSrvConfigFile{}
			err = yaml.Unmarshal([]byte(cm.Data[OLSConfigFilename]), &olsconfigGenerated)
			Expect(err).NotTo(HaveOccurred())

			olsConfigExpected := AppSrvConfigFile{
				OLSConfig: OLSConfig{
					DefaultModel:    "testModel",
					DefaultProvider: "testProvider",
					Logging: LoggingConfig{
						AppLogLevel: "INFO",
						LibLogLevel: "INFO",
					},
					ConversationCache: ConversationCacheConfig{
						Type:   "memory",
						Memory: MemoryCacheConfig{MaxEntries: 1},
					},
				},
				LLMProviders: []ProviderConfig{
					{
						Name:            "testProvider",
						URL:             "testURL",
						CredentialsPath: "/etc/apikeys/test-secret/apitoken",
						Models: []ModelConfig{
							{
								Name: "testModel",
								URL:  "testURL",
							},
						},
					},
				},
			}

			Expect(olsconfigGenerated).To(Equal(olsConfigExpected))

			cmHash, err := hashBytes([]byte(cm.Data[OLSConfigFilename]))
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.ObjectMeta.Annotations[OLSConfigHashKey]).To(Equal(cmHash))

		})

		It("should generate the OLS deployment", func() {
			options := OLSConfigReconcilerOptions{
				LightspeedServiceImage: "repo/lightspeed-service:latest",
			}
			r := OLSConfigReconciler{Options: options}
			dep, err := r.generateOLSDeployment(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(dep.Name).To(Equal(OLSAppServerDeploymentName))
			Expect(dep.Namespace).To(Equal(cr.Namespace))
			Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal(options.LightspeedServiceImage))
			Expect(dep.Spec.Template.Spec.Containers[0].Name).To(Equal("lightspeed-service-api"))
			Expect(dep.Spec.Template.Spec.Containers[0].Ports).To(Equal([]corev1.ContainerPort{
				{
					ContainerPort: OLSAppServerContainerPort,
					Name:          "http",
					Protocol:      corev1.ProtocolTCP,
				},
			}))
			Expect(dep.Spec.Template.Spec.Containers[0].Env).To(Equal([]corev1.EnvVar{
				{
					Name:  "OLS_CONFIG_FILE",
					Value: path.Join("/etc/ols", OLSConfigFilename),
				},
			}))
			Expect(dep.Spec.Template.Spec.Containers[0].VolumeMounts).To(Equal([]corev1.VolumeMount{
				{
					Name:      "secret-test-secret",
					MountPath: path.Join(APIKeyMountRoot, "test-secret"),
					ReadOnly:  true,
				},
				{
					Name:      "cm-olsconfig",
					MountPath: "/etc/ols",
					ReadOnly:  true,
				},
			}))
			Expect(dep.Spec.Template.Spec.Volumes).To(Equal([]corev1.Volume{
				{
					Name: "secret-test-secret",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "test-secret",
						},
					},
				},
				{
					Name: "cm-olsconfig",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: OLSConfigCmName},
						},
					},
				},
			}))
			Expect(dep.Spec.Selector.MatchLabels).To(Equal(deploymentSelectorLabels))
		})

		It("should generate the OLS service", func() {
			r := OLSConfigReconciler{}
			service, err := r.generateService(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.Name).To(Equal(OLSAppServerServiceName))
			Expect(service.Namespace).To(Equal(cr.Namespace))
			Expect(service.Spec.Selector).To(Equal(deploymentSelectorLabels))
			Expect(service.Spec.Ports).To(Equal([]corev1.ServicePort{
				{
					Name:       "http",
					Port:       OLSAppServerServicePort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.Parse("http"),
				},
			}))
		})

	})

	Context("empty custome resource", func() {
		BeforeEach(func() {
			cr = getEmptyOLSConfigCR()
		})

		It("should generate a service account", func() {
			r := OLSConfigReconciler{}
			sa, err := r.generateServiceAccount(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(sa.Name).To(Equal(OLSAppServerServiceAccountName))
			Expect(sa.Namespace).To(Equal("openshift-lightspeed"))
		})

		It("should generate the olsconfig config map", func() {
			// todo: this test is not complete
			// generateOLSConfigMap should return an error if the CR is missing required fields
			r := OLSConfigReconciler{}
			cm, err := r.generateOLSConfigMap(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Name).To(Equal(OLSConfigCmName))
			Expect(cm.Namespace).To(Equal("openshift-lightspeed"))
			const expectedConfigStr = `llm_providers: []
ols_config:
  conversation_cache:
    memory: {}
    type: ""
  logging_config:
    app_log_level: ""
    lib_log_level: ""
`
			Expect(cm.Data[OLSConfigFilename]).To(Equal(expectedConfigStr))
		})

		It("should generate the OLS service", func() {
			r := OLSConfigReconciler{}
			service, err := r.generateService(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.Name).To(Equal(OLSAppServerServiceName))
			Expect(service.Namespace).To(Equal("openshift-lightspeed"))
			Expect(service.Spec.Selector).To(Equal(deploymentSelectorLabels))
			Expect(service.Spec.Ports).To(Equal([]corev1.ServicePort{
				{
					Name:       "http",
					Port:       OLSAppServerServicePort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.Parse("http"),
				},
			}))
		})

		It("should generate the OLS deployment", func() {
			// todo: update this test after updating the test for generateOLSConfigMap
			options := OLSConfigReconcilerOptions{
				LightspeedServiceImage: "repo/lightspeed-service:latest",
			}
			r := OLSConfigReconciler{Options: options}
			dep, err := r.generateOLSDeployment(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(dep.Name).To(Equal(OLSAppServerDeploymentName))
			Expect(dep.Namespace).To(Equal("openshift-lightspeed"))
			Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal(options.LightspeedServiceImage))
			Expect(dep.Spec.Template.Spec.Containers[0].Name).To(Equal("lightspeed-service-api"))
			Expect(dep.Spec.Template.Spec.Containers[0].Ports).To(Equal([]corev1.ContainerPort{
				{
					ContainerPort: OLSAppServerContainerPort,
					Name:          "http",
					Protocol:      corev1.ProtocolTCP,
				},
			}))
			Expect(dep.Spec.Template.Spec.Containers[0].Env).To(Equal([]corev1.EnvVar{
				{
					Name:  "OLS_CONFIG_FILE",
					Value: path.Join("/etc/ols", OLSConfigFilename),
				},
			}))
			Expect(dep.Spec.Template.Spec.Containers[0].VolumeMounts).To(Equal([]corev1.VolumeMount{
				{
					Name:      "cm-olsconfig",
					MountPath: "/etc/ols",
					ReadOnly:  true,
				},
			}))
			Expect(dep.Spec.Template.Spec.Volumes).To(Equal([]corev1.Volume{
				{
					Name: "cm-olsconfig",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: OLSConfigCmName},
						},
					},
				},
			}))
			Expect(dep.Spec.Selector.MatchLabels).To(Equal(deploymentSelectorLabels))

		})
	})

})

func getCompleteOLSConfigCR() *olsv1alpha1.OLSConfig {
	// fill the CR with all implemented fields in the configuration file
	return &olsv1alpha1.OLSConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster",
			Namespace: "openshift-lightspeed",
		},
		Spec: olsv1alpha1.OLSConfigSpec{
			LLMConfig: olsv1alpha1.LLMSpec{
				Providers: []olsv1alpha1.ProviderSpec{
					{
						Name: "testProvider",
						URL:  "testURL",
						CredentialsSecretRef: corev1.LocalObjectReference{
							Name: "test-secret",
						},
						Models: []olsv1alpha1.ModelSpec{
							{
								Name: "testModel",
								URL:  "testURL",
							},
						},
					},
				},
			},
			OLSConfig: olsv1alpha1.OLSSpec{
				DefaultModel:    "testModel",
				DefaultProvider: "testProvider",
				LogLevel:        "INFO",
				ConversationCache: olsv1alpha1.ConversationCacheSpec{
					Type: olsv1alpha1.Memory,
					Memory: olsv1alpha1.MemorySpec{
						MaxEntries: 1,
					},
				},
			},
		},
	}
}

func getEmptyOLSConfigCR() *olsv1alpha1.OLSConfig {
	// The CR has no fields set in its specs
	return &olsv1alpha1.OLSConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster",
			Namespace: "openshift-lightspeed",
		},
	}

}
