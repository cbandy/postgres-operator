package kubeapi

import (
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a new namespace.
func (k *KubeAPI) CreateNamespace(name string) (*core_v1.Namespace, error) {
	return k.Client.CoreV1().Namespaces().Create(&core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{Name: name},
	})
}

// DeleteNamespace deletes an existing namespace.
func (k *KubeAPI) DeleteNamespace(name string) error {
	return k.Client.CoreV1().Namespaces().Delete(name, nil)
}

// GenerateNamespace creates a new namespace with a random name that begins with prefix.
func (k *KubeAPI) GenerateNamespace(prefix string) (*core_v1.Namespace, error) {
	return k.Client.CoreV1().Namespaces().Create(&core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{GenerateName: prefix},
	})
}
