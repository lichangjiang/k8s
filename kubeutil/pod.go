package kubeutil

import (
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func VolumeMount(name, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{Name: name, MountPath: mountPath}
}

func EnvVar(name, value string) corev1.EnvVar {
	return corev1.EnvVar{Name: name, Value: value}
}

func getPodIP(timeout int, podFunc func() (*corev1.Pod, error)) (string, error) {
	retries := int(timeout / 5)
	if retries == 0 {
		retries = 1
	}

	for i := 0; i < retries; i++ {
		pod, err := podFunc()
		if err != nil {
			if !errors.IsNotFound(err) {
				return "", err
			}
		}

		if pod != nil {
			ip := pod.Status.PodIP
			if ip != "" {
				return ip, err
			}
		}
		time.Sleep(5 * time.Second)
	}
	return "", fmt.Errorf("waiting for pod ip timeout")
}

func GetPodIp(clientset kubernetes.Interface, namespace, name string, timeout int) (string, error) {
	return getPodIP(timeout, func() (*corev1.Pod, error) {
		return clientset.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	})
}

func GetPodIpWithLabel(clientset kubernetes.Interface, namespace, label string, timeout int) (string, error) {
	return getPodIP(timeout, func() (*corev1.Pod, error) {
		pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: label})
		if err != nil {
			return nil, err
		}
		if len(pods.Items) > 0 {
			return &pods.Items[0], nil
		} else {
			return nil, nil
		}
	})
}

func PodsRunningWithLabel(clientset kubernetes.Interface, namespace, label string) (int, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return 0, err
	}

	running := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == v1.PodRunning {
			running++
		}
	}
	return running, nil
}

func GetPodLog(clientset kubernetes.Interface, namespace string, labelSelector string) (string, error) {
	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(opts)
	if err != nil {
		return "", fmt.Errorf("failed to get version pod. %+v", err)
	}
	for _, pod := range pods.Items {
		req := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &v1.PodLogOptions{})
		readCloser, err := req.Stream()
		if err != nil {
			return "", fmt.Errorf("failed to read from stream. %+v", err)
		}

		builder := &strings.Builder{}
		defer readCloser.Close()
		_, err = io.Copy(builder, readCloser)
		return builder.String(), err
	}

	return "", fmt.Errorf("did not find any pods with label %s", labelSelector)
}

func WaitingForLabeledPodsToRun(k8sClient kubernetes.Interface, label string, namespace string, waitTime int) error {
	retries := int(waitTime / 5)
	if retries == 0 {
		retries = 1
	}

	klog.Infof("retries time %d", retries)
	options := metav1.ListOptions{LabelSelector: label}
	var lastPod corev1.Pod
	for i := 0; i < retries; i++ {
		pods, err := k8sClient.CoreV1().Pods(namespace).List(options)
		lastStatus := ""
		running := 0
		if err == nil && len(pods.Items) > 0 {
			for _, pod := range pods.Items {
				if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" {
					running++
				}
				lastPod = pod
				lastStatus = string(pod.Status.Phase)
			}
			if running == len(pods.Items) {
				klog.Infof("All %d pod(s) with label %s are running", len(pods.Items), label)
				return nil
			}
		}
		klog.Infof("waiting for pod(s) with label %s in namespace %s to be running. status=%s, running=%d/%d, err=%+v",
			label, namespace, lastStatus, running, len(pods.Items), err)

		time.Sleep(5 * time.Second)
	}

	if len(lastPod.Name) == 0 {
		klog.Infof("no pod was found with label %s", label)
	}
	return fmt.Errorf("timeout when waiting for pod with label %s in namespace %s to be running", label, namespace)

}
