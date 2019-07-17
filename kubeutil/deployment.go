package kubeutil

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetDeploymentImage(k8sClient kubernetes.Interface, namespace, name, container string) (string, error) {
	deploy, err := k8sClient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to find deployment %s/%s. %v", namespace, name, err)
	}
	return GetDeploymentSpecImage(k8sClient, *deploy, container)
}

func GetDeploymentSpecImage(k8sClient kubernetes.Interface, deploy appsv1.Deployment, container string) (string, error) {
	image, err := GetSpecContainerImage(deploy.Spec.Template.Spec, container)
	if err != nil {
		return "", err
	}

	return image, nil
}

func GetSpecContainerImage(spec corev1.PodSpec, name string) (string, error) {
	image, err := GetMatchingContainer(spec.Containers, name)
	if err != nil {
		return "", err
	}
	return image.Image, nil
}

func GetMatchingContainer(containers []corev1.Container, name string) (corev1.Container, error) {
	var result *corev1.Container
	if len(containers) == 1 {
		// if there is only one pod, use its image rather than require a set container name
		result = &containers[0]
	} else {
		// if there are multiple pods, we require the container to have the expected name
		for _, container := range containers {
			if container.Name == name {
				result = &container
				break
			}
		}
	}

	if result == nil {
		return corev1.Container{}, fmt.Errorf("failed to find image for container %s", name)
	}

	return *result, nil
}
