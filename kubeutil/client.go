package kubeutil

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func DefaultConfigStr() string {
	if home := homeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}

func GetK8sConfig(outConfig string, inCluster bool) (*rest.Config, error) {
	if inCluster {
		return rest.InClusterConfig()
	} else {
		return clientcmd.BuildConfigFromFlags("", outConfig)
	}
}

func CreateK8sClientset(outConfig string, inCluster bool) (kubernetes.Interface, error) {
	config, err := GetK8sConfig(outConfig, inCluster)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
