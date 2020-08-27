package configmap

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

type ConfigMapFileTemplate struct {
	Namespace string
	Mapkey    string
	Directory string
	File      string
}

func (template *ConfigMapFileTemplate) createConfigmap() (configMap corev1.ConfigMap, err error) {
	config := Config{}

	err = nil
	err = config.LoadConfig(template.Directory + string(os.PathSeparator) + template.File)

	configMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      template.Mapkey,
			Namespace: template.Namespace,
		},
		Data: map[string]string{
			template.File: config.Message,
		},
	}

	return configMap, err
}
