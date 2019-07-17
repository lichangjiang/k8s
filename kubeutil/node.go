package kubeutil

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	LabelHostname = "kubernetes.io/hostname"
)

func GetPodImage(k8sClient kubernetes.Interface, ns, name, container string) (string, error) {

	pod, err := k8sClient.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to find pod %s/%s. %v", ns, name, err)
	}
	return GetSpecContainerImage(pod.Spec, container)
}

func GetNodeNameFromHostname(k8sClient kubernetes.Interface, hostName string) (string, error) {
	options := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", LabelHostname, hostName)}
	nodes, err := k8sClient.CoreV1().Nodes().List(options)
	if err != nil {
		return hostName, err
	}

	for _, node := range nodes.Items {
		return node.Name, nil
	}
	return hostName, fmt.Errorf("node not found")
}

func GetNodeHostNames(k8sClient kubernetes.Interface) (map[string]string, error) {
	nodes, err := k8sClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodeMap := map[string]string{}
	for _, node := range nodes.Items {
		nodeMap[node.Name] = node.Labels[LabelHostname]
	}
	return nodeMap, nil
}
